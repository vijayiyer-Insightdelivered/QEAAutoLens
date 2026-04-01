import json
import logging
import re
from urllib.parse import urljoin, urlparse

from bs4 import BeautifulSoup

import config
from email_validator import normalize_email, validate_email
from utils import RateLimiter

log = logging.getLogger(__name__)

# Shared Playwright browser instance (lazy-initialized)
_browser = None
_playwright = None


def _get_browser():
    """Lazy-initialize a shared Playwright browser."""
    global _browser, _playwright
    if _browser is None:
        try:
            from playwright.sync_api import sync_playwright
            _playwright = sync_playwright().start()
            _browser = _playwright.chromium.launch(headless=True)
            log.info("Playwright browser initialized")
        except Exception as e:
            log.warning(f"Playwright not available, using requests only: {e}")
    return _browser


def extract_emails_from_website(base_url, session, rate_limiter, verify_mx=False):
    """Scrape a website for email addresses. Uses requests first, Playwright as fallback."""
    all_emails = {}
    visited_urls = set()

    # Phase 1: Scrape homepage with requests
    blocked = _scrape_page_requests(base_url, 'homepage', session, rate_limiter, all_emails)
    visited_urls.add(base_url.rstrip('/'))

    if not blocked:
        # Phase 2: Scrape known contact paths
        for path in config.CONTACT_PATHS:
            page_url = urljoin(base_url.rstrip('/') + '/', path.lstrip('/'))
            normalized = page_url.rstrip('/')
            if normalized not in visited_urls:
                visited_urls.add(normalized)
                _scrape_page_requests(page_url, path.strip('/'), session, rate_limiter, all_emails)

        # Phase 3: Discover contact links from the homepage HTML
        discovered_urls = _discover_contact_links(base_url, session, rate_limiter)
        for disc_url, label in discovered_urls:
            normalized = disc_url.rstrip('/')
            if normalized not in visited_urls:
                visited_urls.add(normalized)
                _scrape_page_requests(disc_url, f'discovered/{label}', session, rate_limiter, all_emails)
    else:
        # Requests blocked (403/cloudflare) — use Playwright for all pages
        log.info(f"Requests blocked for {base_url}, switching to Playwright")
        urls_to_scrape = [base_url] + [
            urljoin(base_url.rstrip('/') + '/', path.lstrip('/'))
            for path in config.CONTACT_PATHS
        ]
        _scrape_with_playwright(urls_to_scrape, all_emails)
        visited_urls.update(u.rstrip('/') for u in urls_to_scrape)

    # Phase 4: If requests found nothing, try Playwright on key pages
    if not all_emails:
        log.debug(f"No emails via requests for {base_url}, trying Playwright")
        pw_urls = [base_url] + [
            urljoin(base_url.rstrip('/') + '/', path.lstrip('/'))
            for path in ['/contact', '/contact-us', '/get-in-touch', '/about', '/enquiry']
        ]
        _scrape_with_playwright(pw_urls, all_emails)

    # Optional MX verification
    if verify_mx and all_emails:
        from email_validator import verify_mx as check_mx
        verified = {}
        for email, source in all_emails.items():
            domain = email.rsplit('@', 1)[1]
            if check_mx(domain):
                verified[email] = source
        return verified

    return all_emails


def _discover_contact_links(base_url, session, rate_limiter):
    """Scan the homepage HTML for links to contact-related pages on the same domain."""
    try:
        rate_limiter.wait('website', config.RATE_LIMIT_WEBSITE, config.JITTER_WEBSITE)
        import requests as _req
        resp = _req.get(base_url, timeout=10, headers=session.headers, allow_redirects=True)
        if resp.status_code != 200:
            return []

        soup = BeautifulSoup(resp.text, 'lxml')
        base_domain = urlparse(base_url).netloc.lower()
        discovered = []

        for a in soup.find_all('a', href=True):
            href = a['href']
            text = (a.get_text(strip=True) or '').lower()
            full_url = urljoin(base_url, href)

            # Must be same domain
            link_domain = urlparse(full_url).netloc.lower()
            if link_domain != base_domain:
                continue

            # Check if link text or URL path contains contact keywords
            path = urlparse(full_url).path.lower()
            for kw in config.CONTACT_LINK_KEYWORDS:
                if kw in text or kw in path:
                    label = kw.replace(' ', '-')
                    discovered.append((full_url, label))
                    break

        # Deduplicate
        seen = set()
        unique = []
        for url, label in discovered:
            norm = url.rstrip('/')
            if norm not in seen:
                seen.add(norm)
                unique.append((url, label))

        if unique:
            log.debug(f"Discovered {len(unique)} contact-related links on {base_url}")
        return unique[:10]  # cap to avoid over-crawling

    except Exception as e:
        log.debug(f"Link discovery failed for {base_url}: {e}")
        return []


def _scrape_page_requests(url, source_label, session, rate_limiter, results):
    """Scrape a page with requests. Returns True if blocked (403/captcha)."""
    try:
        rate_limiter.wait('website', config.RATE_LIMIT_WEBSITE, config.JITTER_WEBSITE)
        import requests as _req
        resp = _req.get(url, timeout=10, headers=session.headers, allow_redirects=True)

        if resp.status_code in (403, 503):
            return True  # blocked
        if resp.status_code != 200:
            return False

        _extract_emails_from_html(resp.text, source_label, results)
        return False

    except Exception as e:
        log.debug(f"Failed to scrape {url}: {e}")
        return False


def _scrape_with_playwright(urls, results):
    """Scrape multiple URLs using a headless browser."""
    browser = _get_browser()
    if not browser:
        return

    try:
        page = browser.new_page()
        page.set_default_timeout(15000)

        for url in urls:
            try:
                resp = page.goto(url, wait_until='domcontentloaded', timeout=15000)
                if resp and resp.status == 200:
                    # Wait for JS to render — longer wait for better results
                    page.wait_for_timeout(3000)
                    html = page.content()
                    path = urlparse(url).path.strip('/') or 'homepage'
                    _extract_emails_from_html(html, f"{path}/browser", results)
            except Exception as e:
                log.debug(f"Playwright failed for {url}: {e}")

        page.close()
    except Exception as e:
        log.warning(f"Playwright session error: {e}")


def _extract_emails_from_html(html, source_label, results):
    """Extract emails from raw HTML string."""
    soup = BeautifulSoup(html, 'lxml')

    # 1. Extract mailto: links
    for a in soup.find_all('a', href=True):
        href = a['href']
        if href.startswith('mailto:'):
            email = href.replace('mailto:', '').split('?')[0].strip()
            email = normalize_email(email)
            if validate_email(email) and email not in results:
                results[email] = source_label

    # 2. Regex scan full HTML text
    for match in config.EMAIL_REGEX.findall(html):
        email = normalize_email(match)
        if validate_email(email) and email not in results:
            results[email] = source_label

    # 3. Decode obfuscated emails: "info [at] domain [dot] com"
    text_content = soup.get_text(separator=' ')
    for match in config.OBFUSCATED_EMAIL_PATTERN.finditer(text_content):
        local, domain, tld = match.groups()
        email = normalize_email(f"{local}@{domain}.{tld}")
        if validate_email(email) and email not in results:
            results[email] = f"{source_label}/obfuscated"

    # 4. Check for emails in HTML comments (some CMS put them there)
    for comment in soup.find_all(string=lambda t: isinstance(t, type(soup.new_string(''))) and '<!--' not in str(t)):
        pass  # already covered by regex scan
    # Scan raw HTML for comments containing emails
    import re as _re
    for comment_match in _re.finditer(r'<!--(.*?)-->', html, _re.DOTALL):
        comment_text = comment_match.group(1)
        for email_match in config.EMAIL_REGEX.findall(comment_text):
            email = normalize_email(email_match)
            if validate_email(email) and email not in results:
                results[email] = f"{source_label}/comment"

    # 5. Check data attributes (data-email, data-contact, etc.)
    for tag in soup.find_all(attrs={"data-email": True}):
        email = normalize_email(tag['data-email'])
        if validate_email(email) and email not in results:
            results[email] = f"{source_label}/data-attr"
    for tag in soup.find_all(attrs={"data-contact": True}):
        email = normalize_email(tag['data-contact'])
        if validate_email(email) and email not in results:
            results[email] = f"{source_label}/data-attr"

    # 6. Check JSON-LD structured data
    for script in soup.find_all('script', type='application/ld+json'):
        try:
            data = json.loads(script.string or '')
            _extract_from_jsonld(data, source_label + '/jsonld', results)
        except (json.JSONDecodeError, TypeError):
            pass

    # 7. Check meta tags (some sites put contact email in meta)
    for meta in soup.find_all('meta'):
        content = meta.get('content', '')
        if '@' in content:
            for email_match in config.EMAIL_REGEX.findall(content):
                email = normalize_email(email_match)
                if validate_email(email) and email not in results:
                    results[email] = f"{source_label}/meta"

    # 8. Decode HTML-entity-encoded emails (&#64; = @, &#46; = .)
    import html as html_mod
    decoded_html = html_mod.unescape(html)
    if decoded_html != html:
        for match in config.EMAIL_REGEX.findall(decoded_html):
            email = normalize_email(match)
            if validate_email(email) and email not in results:
                results[email] = f"{source_label}/decoded"


def _extract_from_jsonld(data, source_label, results):
    if isinstance(data, dict):
        for key in ('email', 'contactPoint', 'author', 'publisher'):
            if key in data:
                val = data[key]
                if isinstance(val, str):
                    email = normalize_email(val.replace('mailto:', ''))
                    if validate_email(email) and email not in results:
                        results[email] = source_label
                elif isinstance(val, dict):
                    if 'email' in val:
                        email = normalize_email(val['email'].replace('mailto:', ''))
                        if validate_email(email) and email not in results:
                            results[email] = source_label
                    _extract_from_jsonld(val, source_label, results)
                elif isinstance(val, list):
                    for item in val:
                        _extract_from_jsonld(item, source_label, results)
    elif isinstance(data, list):
        for item in data:
            _extract_from_jsonld(item, source_label, results)
