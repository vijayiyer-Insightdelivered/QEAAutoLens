import { useState } from "react";

const BRAND = {
  navy: "#003366",
  orange: "#E86E29",
  orangeLight: "#F28C4E",
  white: "#FFFFFF",
  lightGrey: "#F4F6F8",
  midGrey: "#8899AA",
  dark: "#0A1628",
};

// The core AutoLens icon - a stylised lens/eye with data beams
const AutoLensIcon = ({ size = 60, color = BRAND.navy, accent = BRAND.orange }) => (
  <svg width={size} height={size} viewBox="0 0 120 120" fill="none" xmlns="http://www.w3.org/2000/svg">
    {/* Outer lens ring */}
    <circle cx="60" cy="60" r="52" stroke={color} strokeWidth="5" fill="none" />
    
    {/* Inner lens aperture blades - creates camera/lens feel */}
    <path d="M60 18 L72 42 L60 36 Z" fill={accent} opacity="0.9" />
    <path d="M96.4 39 L78 51 L81 39 Z" fill={accent} opacity="0.75" />
    <path d="M96.4 81 L81 81 L78 69 Z" fill={accent} opacity="0.6" />
    <path d="M60 102 L48 78 L60 84 Z" fill={accent} opacity="0.9" />
    <path d="M23.6 81 L42 69 L39 81 Z" fill={accent} opacity="0.75" />
    <path d="M23.6 39 L39 39 L42 51 Z" fill={accent} opacity="0.6" />
    
    {/* Central eye/lens core */}
    <circle cx="60" cy="60" r="18" fill={color} />
    <circle cx="60" cy="60" r="11" fill={accent} />
    <circle cx="60" cy="60" r="5" fill={BRAND.white} opacity="0.95" />
    
    {/* Highlight reflection */}
    <circle cx="54" cy="54" r="3" fill={BRAND.white} opacity="0.6" />
    
    {/* Data scan lines */}
    <line x1="10" y1="60" x2="38" y2="60" stroke={accent} strokeWidth="2" strokeDasharray="3 3" opacity="0.5" />
    <line x1="82" y1="60" x2="110" y2="60" stroke={accent} strokeWidth="2" strokeDasharray="3 3" opacity="0.5" />
    
    {/* Corner tech brackets */}
    <path d="M15 30 L15 15 L30 15" stroke={color} strokeWidth="3" fill="none" strokeLinecap="round" />
    <path d="M90 15 L105 15 L105 30" stroke={color} strokeWidth="3" fill="none" strokeLinecap="round" />
    <path d="M105 90 L105 105 L90 105" stroke={color} strokeWidth="3" fill="none" strokeLinecap="round" />
    <path d="M30 105 L15 105 L15 90" stroke={color} strokeWidth="3" fill="none" strokeLinecap="round" />
  </svg>
);

// Full logo with text
const LogoPrimary = ({ scale = 1, dark = false }) => {
  const bg = dark ? BRAND.dark : "transparent";
  const textColor = dark ? BRAND.white : BRAND.navy;
  const subColor = dark ? BRAND.midGrey : "#667788";
  
  return (
    <div style={{
      display: "inline-flex",
      alignItems: "center",
      gap: 14 * scale,
      padding: `${16 * scale}px ${20 * scale}px`,
      background: bg,
      borderRadius: 8,
    }}>
      <AutoLensIcon 
        size={64 * scale} 
        color={dark ? BRAND.white : BRAND.navy} 
        accent={BRAND.orange} 
      />
      <div style={{ display: "flex", flexDirection: "column", gap: 0 }}>
        <div style={{
          fontFamily: "'Segoe UI', 'SF Pro Display', system-ui, sans-serif",
          fontWeight: 300,
          fontSize: 13 * scale,
          letterSpacing: 4 * scale,
          color: BRAND.orange,
          textTransform: "uppercase",
          lineHeight: 1,
          marginBottom: 2 * scale,
        }}>
          QEA
        </div>
        <div style={{
          fontFamily: "'Segoe UI', 'SF Pro Display', system-ui, sans-serif",
          fontWeight: 700,
          fontSize: 30 * scale,
          letterSpacing: 1.5 * scale,
          color: textColor,
          lineHeight: 1,
          textTransform: "uppercase",
        }}>
          Auto<span style={{ color: BRAND.orange }}>Lens</span>
        </div>
        <div style={{
          fontFamily: "'Segoe UI', 'SF Pro Display', system-ui, sans-serif",
          fontWeight: 400,
          fontSize: 8.5 * scale,
          letterSpacing: 3.5 * scale,
          color: subColor,
          textTransform: "uppercase",
          marginTop: 3 * scale,
          lineHeight: 1,
        }}>
          Dealer Intelligence
        </div>
      </div>
    </div>
  );
};

// Stacked/vertical variant
const LogoStacked = ({ scale = 1, dark = false }) => {
  const textColor = dark ? BRAND.white : BRAND.navy;
  const subColor = dark ? BRAND.midGrey : "#667788";
  
  return (
    <div style={{
      display: "inline-flex",
      flexDirection: "column",
      alignItems: "center",
      gap: 10 * scale,
      padding: `${20 * scale}px ${28 * scale}px`,
      background: dark ? BRAND.dark : "transparent",
      borderRadius: 8,
    }}>
      <AutoLensIcon 
        size={80 * scale} 
        color={dark ? BRAND.white : BRAND.navy} 
        accent={BRAND.orange} 
      />
      <div style={{ textAlign: "center" }}>
        <div style={{
          fontFamily: "'Segoe UI', 'SF Pro Display', system-ui, sans-serif",
          fontWeight: 300,
          fontSize: 12 * scale,
          letterSpacing: 5 * scale,
          color: BRAND.orange,
          textTransform: "uppercase",
          marginBottom: 2 * scale,
        }}>
          QEA
        </div>
        <div style={{
          fontFamily: "'Segoe UI', 'SF Pro Display', system-ui, sans-serif",
          fontWeight: 700,
          fontSize: 32 * scale,
          letterSpacing: 2 * scale,
          color: textColor,
          textTransform: "uppercase",
          lineHeight: 1.1,
        }}>
          Auto<span style={{ color: BRAND.orange }}>Lens</span>
        </div>
        <div style={{
          fontFamily: "'Segoe UI', 'SF Pro Display', system-ui, sans-serif",
          fontWeight: 400,
          fontSize: 8 * scale,
          letterSpacing: 4 * scale,
          color: subColor,
          textTransform: "uppercase",
          marginTop: 4 * scale,
        }}>
          Dealer Intelligence
        </div>
      </div>
    </div>
  );
};

// Compact badge for merchandise / favicon
const LogoBadge = ({ size = 80, dark = false }) => {
  const bg = dark ? BRAND.dark : BRAND.navy;
  return (
    <div style={{
      display: "inline-flex",
      alignItems: "center",
      justifyContent: "center",
      width: size,
      height: size,
      borderRadius: size * 0.18,
      background: bg,
      boxShadow: "0 4px 16px rgba(0,0,0,0.2)",
    }}>
      <AutoLensIcon size={size * 0.68} color={BRAND.white} accent={BRAND.orange} />
    </div>
  );
};

// Circular badge variant
const LogoCircleBadge = ({ size = 80, dark = false }) => {
  const bg = dark ? BRAND.dark : BRAND.navy;
  return (
    <div style={{
      display: "inline-flex",
      alignItems: "center",
      justifyContent: "center",
      width: size,
      height: size,
      borderRadius: "50%",
      background: bg,
      boxShadow: "0 4px 16px rgba(0,0,0,0.2)",
      border: `3px solid ${BRAND.orange}`,
    }}>
      <AutoLensIcon size={size * 0.6} color={BRAND.white} accent={BRAND.orange} />
    </div>
  );
};

export default function QEAAutoLensLogos() {
  const [activeBg, setActiveBg] = useState("light");
  
  const bgStyles = {
    light: { background: "#FFFFFF", color: BRAND.navy },
    grey: { background: BRAND.lightGrey, color: BRAND.navy },
    dark: { background: BRAND.dark, color: BRAND.white },
    navy: { background: BRAND.navy, color: BRAND.white },
  };
  
  const isDark = activeBg === "dark" || activeBg === "navy";
  
  return (
    <div style={{
      minHeight: "100vh",
      background: "#0D1117",
      fontFamily: "'Segoe UI', 'SF Pro Display', system-ui, sans-serif",
      color: BRAND.white,
    }}>
      {/* Header */}
      <div style={{
        padding: "32px 40px 24px",
        borderBottom: `1px solid rgba(255,255,255,0.08)`,
      }}>
        <div style={{ fontSize: 11, letterSpacing: 4, color: BRAND.orange, textTransform: "uppercase", marginBottom: 6 }}>
          Brand Identity
        </div>
        <h1 style={{ fontSize: 28, fontWeight: 700, margin: 0, letterSpacing: 0.5 }}>
          QEA AutoLens — Logo System
        </h1>
        <p style={{ fontSize: 13, color: BRAND.midGrey, marginTop: 6, maxWidth: 600, lineHeight: 1.6 }}>
          Complete logo suite for web, print, merchandise, and digital media. Built on the Insight Delivered brand palette.
        </p>
        
        {/* Background selector */}
        <div style={{ display: "flex", gap: 8, marginTop: 16 }}>
          {Object.entries(bgStyles).map(([key, val]) => (
            <button
              key={key}
              onClick={() => setActiveBg(key)}
              style={{
                padding: "6px 16px",
                borderRadius: 6,
                border: activeBg === key ? `2px solid ${BRAND.orange}` : "2px solid rgba(255,255,255,0.12)",
                background: val.background,
                color: val.color,
                fontSize: 11,
                fontWeight: 600,
                cursor: "pointer",
                textTransform: "uppercase",
                letterSpacing: 1,
              }}
            >
              {key}
            </button>
          ))}
        </div>
      </div>
      
      {/* Logo Variants */}
      <div style={{ padding: "32px 40px", display: "flex", flexDirection: "column", gap: 32 }}>
        
        {/* Primary Horizontal */}
        <Section title="Primary Logo — Horizontal Lockup" subtitle="Websites, presentations, letterheads, email signatures">
          <PreviewBox bg={bgStyles[activeBg].background}>
            <LogoPrimary scale={1.1} dark={isDark} />
          </PreviewBox>
        </Section>
        
        {/* Stacked */}
        <Section title="Stacked Logo — Vertical Lockup" subtitle="Social media profiles, app icons, square placements, merchandise">
          <PreviewBox bg={bgStyles[activeBg].background}>
            <LogoStacked scale={1.1} dark={isDark} />
          </PreviewBox>
        </Section>
        
        {/* Icon Variants */}
        <Section title="Icon / Badge Variants" subtitle="Favicons, app icons, merchandise embroidery, social media avatars">
          <div style={{ display: "flex", gap: 24, flexWrap: "wrap" }}>
            <PreviewBox bg={bgStyles[activeBg].background} style={{ padding: 24 }}>
              <div style={{ display: "flex", gap: 20, alignItems: "center" }}>
                <LogoBadge size={90} dark={isDark} />
                <LogoCircleBadge size={90} dark={isDark} />
                <LogoBadge size={56} dark={isDark} />
                <LogoCircleBadge size={56} dark={isDark} />
                <LogoBadge size={36} dark={isDark} />
                <LogoCircleBadge size={36} dark={isDark} />
              </div>
            </PreviewBox>
          </div>
        </Section>
        
        {/* Icon Only */}
        <Section title="Standalone Icon — No Container" subtitle="Watermarks, patterns, subtle branding">
          <PreviewBox bg={bgStyles[activeBg].background}>
            <div style={{ display: "flex", gap: 24, alignItems: "center" }}>
              <AutoLensIcon size={80} color={isDark ? BRAND.white : BRAND.navy} accent={BRAND.orange} />
              <AutoLensIcon size={56} color={isDark ? BRAND.white : BRAND.navy} accent={BRAND.orange} />
              <AutoLensIcon size={36} color={isDark ? BRAND.white : BRAND.navy} accent={BRAND.orange} />
            </div>
          </PreviewBox>
        </Section>

        {/* Colour Palette */}
        <Section title="Brand Colour Palette" subtitle="Primary and accent colours for all QEA AutoLens materials">
          <div style={{ display: "flex", gap: 12, flexWrap: "wrap" }}>
            {[
              { name: "Navy Blue", hex: BRAND.navy, text: "#FFF" },
              { name: "AutoLens Orange", hex: BRAND.orange, text: "#FFF" },
              { name: "Dark Background", hex: BRAND.dark, text: "#FFF" },
              { name: "White", hex: BRAND.white, text: BRAND.navy },
              { name: "Light Grey", hex: BRAND.lightGrey, text: BRAND.navy },
              { name: "Mid Grey", hex: BRAND.midGrey, text: "#FFF" },
            ].map((c) => (
              <div key={c.hex} style={{
                width: 120, borderRadius: 8, overflow: "hidden",
                boxShadow: "0 2px 8px rgba(0,0,0,0.3)",
              }}>
                <div style={{ height: 64, background: c.hex, border: c.hex === "#FFFFFF" ? "1px solid #ddd" : "none" }} />
                <div style={{ padding: "8px 10px", background: "#1A1F2E" }}>
                  <div style={{ fontSize: 11, fontWeight: 600, color: "#FFF" }}>{c.name}</div>
                  <div style={{ fontSize: 10, color: BRAND.midGrey, fontFamily: "monospace" }}>{c.hex}</div>
                </div>
              </div>
            ))}
          </div>
        </Section>

        {/* Usage examples */}
        <Section title="Application Preview" subtitle="How the logo works in context">
          <div style={{ display: "flex", gap: 16, flexWrap: "wrap" }}>
            {/* Business card mockup */}
            <div style={{
              width: 320, height: 190, borderRadius: 10, overflow: "hidden",
              background: `linear-gradient(135deg, ${BRAND.navy} 0%, ${BRAND.dark} 100%)`,
              padding: 24, display: "flex", flexDirection: "column", justifyContent: "space-between",
              boxShadow: "0 8px 32px rgba(0,0,0,0.4)",
            }}>
              <LogoPrimary scale={0.55} dark={true} />
              <div>
                <div style={{ fontSize: 12, fontWeight: 600, color: BRAND.white }}>Alex Castle</div>
                <div style={{ fontSize: 10, color: BRAND.midGrey, marginTop: 2 }}>Sales Director</div>
                <div style={{ fontSize: 9, color: BRAND.orange, marginTop: 6 }}>alex.castle@insightdelivered.com</div>
              </div>
            </div>
            
            {/* Merch / cap mockup */}
            <div style={{
              width: 190, height: 190, borderRadius: 10, overflow: "hidden",
              background: BRAND.navy,
              display: "flex", alignItems: "center", justifyContent: "center",
              boxShadow: "0 8px 32px rgba(0,0,0,0.4)",
              position: "relative",
            }}>
              <div style={{
                position: "absolute", top: 10, left: 0, right: 0, textAlign: "center",
                fontSize: 8, letterSpacing: 3, color: BRAND.midGrey, textTransform: "uppercase",
              }}>Merchandise</div>
              <LogoStacked scale={0.65} dark={true} />
            </div>
            
            {/* Website header mockup */}
            <div style={{
              width: 320, height: 190, borderRadius: 10, overflow: "hidden",
              background: "#FFFFFF",
              boxShadow: "0 8px 32px rgba(0,0,0,0.2)",
              display: "flex", flexDirection: "column",
            }}>
              <div style={{
                padding: "10px 16px",
                borderBottom: "1px solid #eee",
                display: "flex",
                alignItems: "center",
                justifyContent: "space-between",
              }}>
                <LogoPrimary scale={0.38} dark={false} />
                <div style={{ display: "flex", gap: 12 }}>
                  {["Solutions", "Pricing", "Contact"].map(t => (
                    <span key={t} style={{ fontSize: 9, color: BRAND.navy, fontWeight: 500 }}>{t}</span>
                  ))}
                </div>
              </div>
              <div style={{
                flex: 1,
                background: `linear-gradient(135deg, ${BRAND.navy} 0%, #0A2244 100%)`,
                display: "flex", alignItems: "center", justifyContent: "center",
                flexDirection: "column", gap: 6,
              }}>
                <div style={{ fontSize: 14, fontWeight: 700, color: BRAND.white, textAlign: "center" }}>
                  See Every Deal. Know Every Margin.
                </div>
                <div style={{ fontSize: 8, color: BRAND.midGrey }}>
                  Real-time dealer intelligence powered by QEA
                </div>
                <div style={{
                  marginTop: 6, padding: "5px 18px", borderRadius: 4,
                  background: BRAND.orange, color: BRAND.white,
                  fontSize: 9, fontWeight: 600,
                }}>
                  Book a Demo
                </div>
              </div>
            </div>
          </div>
        </Section>
      </div>
      
      {/* Footer */}
      <div style={{
        padding: "20px 40px",
        borderTop: "1px solid rgba(255,255,255,0.06)",
        fontSize: 10,
        color: "rgba(255,255,255,0.3)",
      }}>
        QEA AutoLens Logo System — Insight Delivered © 2026. Brand colours: Navy #003366, Orange #E86E29.
      </div>
    </div>
  );
}

function Section({ title, subtitle, children }) {
  return (
    <div>
      <h2 style={{ fontSize: 16, fontWeight: 700, margin: 0, color: "#FFF", letterSpacing: 0.3 }}>{title}</h2>
      <p style={{ fontSize: 11, color: BRAND.midGrey, margin: "4px 0 14px", letterSpacing: 0.2 }}>{subtitle}</p>
      {children}
    </div>
  );
}

function PreviewBox({ bg, children, style = {} }) {
  return (
    <div style={{
      background: bg,
      borderRadius: 10,
      padding: 32,
      display: "flex",
      alignItems: "center",
      justifyContent: "center",
      border: bg === "#FFFFFF" ? "1px solid #ddd" : "1px solid rgba(255,255,255,0.06)",
      transition: "background 0.3s ease",
      ...style,
    }}>
      {children}
    </div>
  );
}
