import React, { useState, useEffect } from 'react';
import {
  Box,
  Tab,
  Tabs,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Card,
  CardContent,
  Typography,
  MenuItem,
  Select,
  FormControl,
  Tooltip,
  Skeleton,
  Stack,
  Chip,
  Button,
  alpha,
  useTheme
} from '@mui/material';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip as RechartsTooltip, Legend, ResponsiveContainer } from 'recharts';
import MainCard from '../../../components/MainCard';
import useAuth from '@hooks/useAuth';
import { Empty } from 'antd';
import { Download as DownloadIcon } from '@mui/icons-material';
import * as XLSX from 'xlsx';
import showSnackbar from '../../extra-pages/Others/Showsnackbar';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { formatUKCurrency } from '@utils/formatters';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
const getDateRangeForVatPeriod = (vatPeriod) => {
  const match = vatPeriod?.match(/Q(\d)\s*(\d{4})/);
  if (!match) return { start: '2025-04-01', end: '2025-06-30' };
  const quarter = parseInt(match[1]);
  const year = parseInt(match[2]);
  const quarterDates = {
    1: { start: `${year}-01-01`, end: `${year}-03-31` },
    2: { start: `${year}-04-01`, end: `${year}-06-30` },
    3: { start: `${year}-07-01`, end: `${year}-09-30` },
    4: { start: `${year}-10-01`, end: `${year}-12-31` }
  };
  return quarterDates[quarter] || { start: '2025-04-01', end: '2025-06-30' };
};

const mapApiDataToVatCalculations = (apiData) => {
  if (!apiData) return [];
  return [
    { box: '1', category: 'VAT due in this period on sales and other outputs', q2_2025: formatUKCurrency(apiData.vatDueSales || 0) },
    { box: '2', category: 'VAT due in the period on acquisitions of goods made in Northern Ireland from EU Member States', q2_2025: formatUKCurrency(apiData.vatDueAcquisitions || 0) },
    { box: '3', category: 'Total VAT due (the sum of boxes 1 and 2)', q2_2025: formatUKCurrency(apiData.totalVatDue || 0) },
    { box: '4', category: 'VAT reclaimed in this period on purchases and other inputs (including acquisitions in Northern Ireland from EU Member States)', q2_2025: formatUKCurrency(apiData.vatReclaimedCurrPeriod || 0) },
    { box: '5', category: 'Net VAT to be paid to customs or reclaimed by you (Difference between boxes 3 and 4)', q2_2025: formatUKCurrency(apiData.netVatDue || 0) },
    { box: '6', category: 'Total value of sales and all other outputs excluding any VAT. Include your box 8 figure', q2_2025: formatUKCurrency(apiData.totalValueSalesExVAT || 0) },
    { box: '7', category: 'Total value of purchases and all other inputs excluding any VAT. Include your box 9 figure', q2_2025: formatUKCurrency(apiData.totalValuePurchasesExVAT || 0) },
    { box: '8', category: 'Total value of dispatches of goods and related costs (excluding VAT) from Northern Ireland to EU Member States', q2_2025: formatUKCurrency(apiData.totalValueGoodsSuppliedExVAT || 0) },
    { box: '9', category: 'Total value of acquisitions of goods and related costs (excluding VAT) made in Northern Ireland from EU Member States', q2_2025: formatUKCurrency(apiData.totalAcquisitionsExVAT || 0) }
  ];
};

// ---------------------------------------------------------------------------
// Compact KPI card for the summary strip
// ---------------------------------------------------------------------------
const KpiCard = ({ label, value, color, subtitle }) => (
  <Card
    sx={{
      flex: 1,
      minWidth: 0,
      position: 'relative',
      overflow: 'hidden',
      borderRadius: 2.5,
      boxShadow: `0 1px 3px ${alpha(color, 0.1)}, 0 6px 16px ${alpha(color, 0.06)}`,
      transition: 'transform 0.2s ease, box-shadow 0.2s ease',
      '&:hover': { transform: 'translateY(-2px)', boxShadow: `0 8px 24px ${alpha(color, 0.14)}` }
    }}
  >
    <Box sx={{ position: 'absolute', top: 0, left: 0, right: 0, height: 3, background: `linear-gradient(90deg, ${color}, ${alpha(color, 0.35)})` }} />
    <CardContent sx={{ py: 1.75, px: 2, '&:last-child': { pb: 1.75 } }}>
      <Typography variant="caption" sx={{ color: 'text.secondary', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.08em', fontSize: '0.62rem', lineHeight: 1 }}>
        {label}
      </Typography>
      <Typography variant="h5" sx={{ fontWeight: 800, mt: 0.5, lineHeight: 1.1, color }}>
        {value}
      </Typography>
      {subtitle && (
        <Typography variant="caption" sx={{ color: 'text.disabled', fontSize: '0.65rem', mt: 0.25, display: 'block' }}>
          {subtitle}
        </Typography>
      )}
    </CardContent>
  </Card>
);

// ---------------------------------------------------------------------------
// Shared table header style factory
// ---------------------------------------------------------------------------
const theadRowSx = (theme) => ({
  background: theme.palette.mode === 'dark'
    ? theme.palette.action.hover
    : `linear-gradient(135deg, ${alpha(theme.palette.primary.main, 0.06)}, ${alpha(theme.palette.primary.main, 0.02)})`,
  '& .MuiTableCell-root': { fontWeight: 700, fontSize: '0.78rem', letterSpacing: '0.02em', py: 1.5, whiteSpace: 'nowrap' }
});

const bodyRowSx = (theme) => ({
  '&:nth-of-type(even)': { bgcolor: alpha(theme.palette.primary.main, 0.02) },
  '&:hover': { bgcolor: alpha(theme.palette.primary.main, 0.05) },
  transition: 'background-color 0.15s ease',
  '& .MuiTableCell-root': { py: 1.5, fontSize: '0.84rem' }
});

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------
function Vat_Page() {
  const theme = useTheme();
  const [mainTab, setMainTab] = useState(0);
  const [chartTab, setChartTab] = useState(0);
  const [selectedYear, setSelectedYear] = useState(() => {
    const stored = localStorage.getItem('vatPageSelectedYear');
    return stored ? parseInt(stored) : new Date().getFullYear();
  });
  const [vatPeriod, setVatPeriod] = useState(() => localStorage.getItem('vatPageSelectedPeriod') || '');
  const [selectedSummaryRow, setSelectedSummaryRow] = useState(0);
  const [vatConfigurations, setVatConfigurations] = useState(() => {
    const stored = localStorage.getItem('vatPageConfigurations');
    return stored ? JSON.parse(stored) : [];
  });

  const BaseUrl = import.meta.env.VITE_Base_URL;
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const availableYears = [2025, 2026];

  // Persist to localStorage
  useEffect(() => { localStorage.setItem('vatPageSelectedYear', selectedYear.toString()); }, [selectedYear]);
  useEffect(() => { localStorage.setItem('vatPageSelectedPeriod', vatPeriod); }, [vatPeriod]);
  useEffect(() => { if (vatConfigurations.length > 0) localStorage.setItem('vatPageConfigurations', JSON.stringify(vatConfigurations)); }, [vatConfigurations]);

  // ── Data fetching ─────────────────────────────────────────────────────
  const { data: configData, isLoading: configLoading } = useQuery({
    queryKey: ['vatConfigs', selectedYear, user?.tenantId],
    queryFn: async () => {
      const res = await fetch(`${BaseUrl}vat/tenant/configs?tenant_info_id=${user?.tenantId}&year=${selectedYear}`);
      if (!res.ok) throw new Error('Failed to fetch VAT configurations');
      return res.json();
    },
    enabled: !!user?.tenantId,
    staleTime: 5 * 60 * 1000
  });

  useEffect(() => {
    if (configData?.vat_configurations && Array.isArray(configData.vat_configurations)) {
      setVatConfigurations(configData.vat_configurations);
      if (configData.vat_configurations.length > 0 && !vatPeriod) {
        setVatPeriod(configData.vat_configurations[0].label);
      }
    }
  }, [configData, vatPeriod]);

  const { data: quarterlyResponse, isLoading: quarterlyLoading, error: quarterlyError } = useQuery({
    queryKey: ['quarterlyData', selectedYear, user?.tenantId],
    queryFn: async () => {
      const res = await fetch(`${BaseUrl}vat/quaterly/return?tenant_info_id=${user?.tenantId}&year=${selectedYear}`);
      if (!res.ok) throw new Error('Failed to fetch quarterly data');
      return res.json();
    },
    enabled: !!user?.tenantId && !!selectedYear,
    staleTime: 5 * 60 * 1000
  });
  const quarterlyData = quarterlyResponse?.quarters || [];

  const { data: apiData, isLoading: loading, error } = useQuery({
    queryKey: ['vatData', vatPeriod, user?.tenantId],
    queryFn: async () => {
      const res = await fetch(`${BaseUrl}vat/return?tenant_info_id=${user?.tenantId}&label=${encodeURIComponent(vatPeriod)}`);
      if (!res.ok) throw new Error('Failed to fetch VAT data');
      return res.json();
    },
    enabled: !!vatPeriod && !!user?.tenantId,
    staleTime: 5 * 60 * 1000
  });
  const vatCalculationsData = apiData ? mapApiDataToVatCalculations(apiData) : [];

  const { data: summaryResponse, isLoading: summaryLoading, error: summaryError } = useQuery({
    queryKey: ['summaryData', vatPeriod, user?.tenantId],
    queryFn: async () => {
      const res = await fetch(`${BaseUrl}vat/transactions/summary/quarterly?tenant_info_id=${user?.tenantId}`);
      if (!res.ok) throw new Error('Failed to fetch summary data');
      return res.json();
    },
    enabled: !!vatPeriod && !!user?.tenantId,
    staleTime: 5 * 60 * 1000,
    select: (data) => {
      const filtered = data?.quarters?.filter(
        (item) => item.summary && item.summary.by_transaction && Object.keys(item.summary.by_transaction).length > 0
      );
      return filtered?.sort((a, b) => new Date(b.start_date) - new Date(a.start_date)) || [];
    }
  });
  const summaryData = summaryResponse || [];

  const { data: transactionsResponse, isLoading: transactionsLoading, error: transactionsError } = useQuery({
    queryKey: ['transactionsData', vatPeriod, mainTab, user?.tenantId],
    queryFn: async () => {
      const res = await fetch(`${BaseUrl}vat/transactions?tenant_info_id=${user?.tenantId}&label=${encodeURIComponent(vatPeriod)}`);
      if (!res.ok) throw new Error('Failed to fetch transactions data');
      return res.json();
    },
    enabled: mainTab === 2 && !!vatPeriod && !!user?.tenantId,
    staleTime: 5 * 60 * 1000
  });
  const transactionsData = Array.isArray(transactionsResponse) ? transactionsResponse : transactionsResponse?.transactions || [];

  // ── Mutations ─────────────────────────────────────────────────────────
  const populateAllMutation = useMutation({
    mutationFn: async () => {
      const res = await fetch(`${BaseUrl}vat/transactions/populate?tenant_info_id=${user?.tenantId}`, { method: 'POST' });
      if (!res.ok) throw new Error('Failed to populate transactions');
      return res.json();
    },
    onError: (err) => console.error('Error populating transactions:', err)
  });

  const handleReloadSummary = async () => {
    try {
      await populateAllMutation.mutateAsync();
      await queryClient.invalidateQueries({ queryKey: ['summaryData'] });
      await queryClient.invalidateQueries({ queryKey: ['vatData'] });
      await queryClient.invalidateQueries({ queryKey: ['transactionsData'] });
      setSelectedSummaryRow(0);
      showSnackbar('Reload completed successfully', 'success');
    } catch (err) {
      console.error('Error during reload:', err);
      showSnackbar('Error during reload', 'error');
    }
  };

  // ── Derived values ────────────────────────────────────────────────────
  const selectedSummaryData = Array.isArray(summaryData) && summaryData.length > 0 ? summaryData[selectedSummaryRow] : {};
  const summarySales = selectedSummaryData?.summary?.by_transaction?.sales?.vat_amount || 0;
  const summaryOverhead = selectedSummaryData?.summary?.by_transaction?.overhead?.vat_amount || 0;
  const summaryStock = selectedSummaryData?.summary?.by_transaction?.stock?.vat_amount || 0;
  const summaryAdjustments = selectedSummaryData?.summary?.adjustments?.vat_amount || 0;
  const summaryNet = summarySales - (summaryOverhead + summaryStock) + summaryAdjustments;

  const { start: summaryStart, end: summaryEnd } = getDateRangeForVatPeriod(vatPeriod);
  const summaryStartDate = summaryStart ? new Date(summaryStart).toLocaleDateString('en-GB') : '';
  const summaryEndDate = summaryEnd ? new Date(summaryEnd).toLocaleDateString('en-GB') : '';

  const dateRangeLabel = selectedSummaryData?.start_date
    ? `${new Date(selectedSummaryData.start_date).toLocaleDateString('en-GB')} – ${new Date(selectedSummaryData.end_date).toLocaleDateString('en-GB')}`
    : `${summaryStartDate} – ${summaryEndDate}`;

  const chartData = (quarterlyData || []).map((q) => ({
    name: q.label || `Q${q.quarter} ${q.year}`,
    'VAT Due': parseFloat((q.vatDueSales || 0).toFixed(2)),
    'VAT Reclaimed': parseFloat((q.vatReclaimedCurrPeriod || 0).toFixed(2)),
    'Net VAT': parseFloat((q.netVatDue || 0).toFixed(2))
  }));

  const mainTabs = ['Summary Report', 'VAT Return', 'Audit VAT'];

  // ── Handlers ──────────────────────────────────────────────────────────
  const handleYearChange = (e) => setSelectedYear(e.target.value);
  const handleMainTabChange = (_, v) => setMainTab(v);
  const handleVatPeriodChange = (e) => setVatPeriod(e.target.value);
  const handleChartTabChange = (_, v) => setChartTab(v);

  const handleExportVatReturn = () => {
    const data = vatCalculationsData.map((row) => ({ BOX: row.box, 'VAT Return Categories': row.category, [vatPeriod]: row.q2_2025 }));
    const ws = XLSX.utils.json_to_sheet(data);
    const wb = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(wb, ws, 'VAT Return');
    XLSX.writeFile(wb, `VAT_Return_${vatPeriod}_${new Date().toISOString().split('T')[0]}.xlsx`);
  };

  const handleExportAuditVat = () => {
    const data = transactionsData.map((row) => ({
      Date: row.date ? new Date(row.date).toLocaleDateString() : '-',
      'Sold Price': formatUKCurrency(row.sold_price || 0),
      'Purchase Price': formatUKCurrency(row.purchase_price || 0),
      Margin: formatUKCurrency(row.margin || 0),
      'VAT Amount': formatUKCurrency(row.vat_amount || 0),
      'Net Amount': formatUKCurrency(row.net_amount || 0),
      Name: row.name || '-',
      Detail: row.details || '-'
    }));
    const totals = {
      Date: 'TOTAL',
      'Sold Price': formatUKCurrency(transactionsData.reduce((s, r) => s + (r.sold_price || 0), 0)),
      'Purchase Price': formatUKCurrency(transactionsData.reduce((s, r) => s + (r.purchase_price || 0), 0)),
      Margin: formatUKCurrency(transactionsData.reduce((s, r) => s + (r.margin || 0), 0)),
      'VAT Amount': formatUKCurrency(transactionsData.reduce((s, r) => s + (r.vat_amount || 0), 0)),
      'Net Amount': formatUKCurrency(transactionsData.reduce((s, r) => s + (r.net_amount || 0), 0)),
      Name: '', Detail: ''
    };
    const ws = XLSX.utils.json_to_sheet([totals, ...data]);
    const wb = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(wb, ws, 'Audit VAT');
    XLSX.writeFile(wb, `Audit_VAT_${vatPeriod}_${new Date().toISOString().split('T')[0]}.xlsx`);
  };

  // ── Render ────────────────────────────────────────────────────────────
  return (
    <Box sx={{ width: '100%', minHeight: '100vh', p: 3 }}>

      {/* ================================================================
          COMPACT TOP SECTION — Summary KPIs or VAT Analysis chart
          ================================================================ */}
      <Card sx={{ borderRadius: 3, boxShadow: '0 1px 3px rgba(0,0,0,0.06), 0 8px 24px rgba(0,0,0,0.07)', overflow: 'hidden', mb: 3 }}>
        {/* Header row: tabs + reload */}
        <Box sx={{ px: 2.5, pt: 2, pb: 0, display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderBottom: 1, borderColor: 'divider' }}>
          <Tabs
            value={chartTab}
            onChange={handleChartTabChange}
            sx={{
              minHeight: 40,
              '& .MuiTab-root': { textTransform: 'none', fontWeight: 600, fontSize: '0.85rem', minHeight: 40, py: 1, px: 2 },
              '& .MuiTabs-indicator': { height: 3, borderRadius: '3px 3px 0 0' }
            }}
          >
            <Tab label="Summary" />
            <Tab label="VAT Analysis" />
          </Tabs>
          <Button
            variant="contained"
            size="small"
            onClick={handleReloadSummary}
            disabled={populateAllMutation.isPending}
            sx={{ textTransform: 'none', fontWeight: 600, fontSize: '0.8rem', borderRadius: 2, px: 2, py: 0.75 }}
          >
            {populateAllMutation.isPending ? 'Reloading...' : 'Reload'}
          </Button>
        </Box>

        {/* Summary tab — compact KPI strip */}
        {chartTab === 0 && (
          <Box sx={{ px: 2.5, py: 2.5 }}>
            {summaryLoading ? (
              <Stack direction="row" spacing={2}>
                {[...Array(4)].map((_, i) => <Skeleton key={i} variant="rounded" height={72} sx={{ flex: 1, borderRadius: 2.5 }} />)}
              </Stack>
            ) : summaryError ? (
              <Typography variant="body2" sx={{ color: 'error.main', textAlign: 'center', py: 3 }}>
                Error: {summaryError?.message || 'Failed to load summary data'}
              </Typography>
            ) : (
              <>
                {/* Period selector chips */}
                {summaryData.length > 1 && (
                  <Stack direction="row" spacing={1} sx={{ mb: 2, flexWrap: 'wrap', gap: 0.5 }}>
                    {summaryData.map((item, idx) => (
                      <Chip
                        key={idx}
                        label={item.label}
                        size="small"
                        onClick={() => setSelectedSummaryRow(idx)}
                        variant={selectedSummaryRow === idx ? 'filled' : 'outlined'}
                        color={selectedSummaryRow === idx ? 'primary' : 'default'}
                        sx={{ fontWeight: 600, fontSize: '0.75rem' }}
                      />
                    ))}
                  </Stack>
                )}

                {/* KPI cards row */}
                <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
                  <KpiCard
                    label="Net VAT Payable"
                    value={formatUKCurrency(summaryNet)}
                    color={summaryNet >= 0 ? theme.palette.error.main : theme.palette.success.main}
                    subtitle={dateRangeLabel}
                  />
                  <KpiCard
                    label="Collected on Sales"
                    value={formatUKCurrency(summarySales)}
                    color={theme.palette.primary.main}
                  />
                  <KpiCard
                    label="Paid on Purchases"
                    value={formatUKCurrency(summaryOverhead)}
                    color={theme.palette.success.main}
                  />
                  <KpiCard
                    label="Stock VAT"
                    value={formatUKCurrency(summaryStock)}
                    color={theme.palette.info.main}
                  />
                </Stack>
              </>
            )}
          </Box>
        )}

        {/* VAT Analysis chart tab — reduced height */}
        {chartTab === 1 && (
          <Box sx={{ px: 2.5, py: 2 }}>
            {quarterlyLoading ? (
              <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: 320 }}>
                <Typography sx={{ color: 'text.secondary' }}>Loading quarterly data...</Typography>
              </Box>
            ) : quarterlyError ? (
              <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: 320 }}>
                <Typography sx={{ color: 'error.main' }}>Error: {quarterlyError?.message || 'Failed to load quarterly data'}</Typography>
              </Box>
            ) : (
              <ResponsiveContainer width="100%" height={320}>
                <BarChart data={chartData} margin={{ top: 20, right: 20, left: 40, bottom: 40 }} barCategoryGap="20%">
                  <CartesianGrid strokeDasharray="3 3" vertical={false} stroke={alpha('#000', 0.06)} />
                  <XAxis dataKey="name" tick={{ fill: theme.palette.text.secondary, fontSize: 12, fontWeight: 500 }} axisLine={false} tickLine={false} />
                  <YAxis
                    label={{ value: 'Amount (£)', angle: -90, position: 'insideLeft', offset: 10, style: { fill: theme.palette.text.secondary, fontWeight: 600, fontSize: 12 } }}
                    tick={{ fill: theme.palette.text.secondary, fontSize: 12 }}
                    axisLine={false} tickLine={false}
                  />
                  <RechartsTooltip
                    formatter={(value) => '£' + value.toLocaleString('en-GB', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
                    contentStyle={{ backgroundColor: theme.palette.background.paper, border: `1px solid ${theme.palette.divider}`, borderRadius: 8, boxShadow: '0 8px 32px rgba(0,0,0,0.12)', padding: 12 }}
                    cursor={{ fill: alpha(theme.palette.primary.main, 0.06) }}
                  />
                  <Legend wrapperStyle={{ paddingTop: 12 }} iconType="square" formatter={(v) => <span style={{ color: theme.palette.text.primary, fontSize: 12, fontWeight: 500 }}>{v}</span>} />
                  <Bar dataKey="VAT Due" fill={theme.palette.primary.main} radius={[6, 6, 0, 0]} animationDuration={600} />
                  <Bar dataKey="VAT Reclaimed" fill={theme.palette.success.main} radius={[6, 6, 0, 0]} animationDuration={600} />
                  <Bar dataKey="Net VAT" fill={theme.palette.warning.main} radius={[6, 6, 0, 0]} animationDuration={600} />
                </BarChart>
              </ResponsiveContainer>
            )}
          </Box>
        )}
      </Card>

      {/* ================================================================
          TABLE SECTION — Maximum vertical space
          ================================================================ */}
      <Card sx={{ borderRadius: 3, boxShadow: '0 1px 3px rgba(0,0,0,0.06), 0 8px 24px rgba(0,0,0,0.07)', overflow: 'hidden' }}>
        {/* Toolbar: Tabs + controls */}
        <Box sx={{ px: 2.5, pt: 1.5, pb: 0, display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 1.5, borderBottom: 1, borderColor: 'divider' }}>
          <Tabs
            value={mainTab}
            onChange={handleMainTabChange}
            variant="scrollable"
            scrollButtons="auto"
            sx={{
              minHeight: 42,
              '& .MuiTab-root': { textTransform: 'none', fontWeight: 600, fontSize: '0.85rem', minHeight: 42, py: 1, px: 2 },
              '& .MuiTabs-indicator': { height: 3, borderRadius: '3px 3px 0 0' }
            }}
          >
            {mainTabs.map((tab) => <Tab key={tab} label={tab} />)}
          </Tabs>

          <Stack direction="row" spacing={1.5} sx={{ pb: 1 }}>
            <FormControl size="small" sx={{ minWidth: 100 }}>
              <Select
                disabled={mainTab === 0 || configLoading}
                value={selectedYear}
                onChange={handleYearChange}
                sx={{ borderRadius: 2, fontWeight: 600, fontSize: '0.82rem' }}
              >
                {availableYears.map((y) => <MenuItem key={y} value={y}>{y}</MenuItem>)}
              </Select>
            </FormControl>
            <FormControl size="small" sx={{ minWidth: 160 }}>
              <Select
                value={vatPeriod}
                onChange={handleVatPeriodChange}
                disabled={mainTab === 0 || configLoading || vatConfigurations.length === 0}
                sx={{ borderRadius: 2, fontWeight: 600, fontSize: '0.82rem' }}
              >
                {configLoading ? (
                  <MenuItem disabled>Loading...</MenuItem>
                ) : vatConfigurations.length > 0 ? (
                  vatConfigurations.map((c) => <MenuItem key={c.label} value={c.label}>{c.label}</MenuItem>)
                ) : (
                  <MenuItem disabled>No periods</MenuItem>
                )}
              </Select>
            </FormControl>
            {mainTab !== 0 && (
              <Tooltip title={mainTab === 1 ? 'Export VAT Return' : 'Export Audit VAT'}>
                <span>
                  <Button
                    variant="contained"
                    size="small"
                    onClick={mainTab === 1 ? handleExportVatReturn : handleExportAuditVat}
                    disabled={mainTab === 1 ? vatCalculationsData.length === 0 : transactionsData.length === 0}
                    startIcon={<DownloadIcon sx={{ fontSize: 16 }} />}
                    sx={{ textTransform: 'none', fontWeight: 600, fontSize: '0.8rem', borderRadius: 2, px: 2, py: 0.75, whiteSpace: 'nowrap' }}
                  >
                    Export
                  </Button>
                </span>
              </Tooltip>
            )}
          </Stack>
        </Box>

        {/* Table content */}
        <TableContainer sx={{ maxHeight: 'calc(100vh - 320px)' }}>
          <Table stickyHeader>

            {/* ── Summary Report tab ── */}
            {mainTab === 0 && (
              <>
                <TableHead>
                  <TableRow sx={theadRowSx(theme)}>
                    <TableCell align="center" sx={{ color: theme.palette.primary.main }}>Label</TableCell>
                    <TableCell align="right" sx={{ color: theme.palette.primary.main }}>VAT on Sales</TableCell>
                    <TableCell align="right" sx={{ color: theme.palette.success.main }}>Expense VAT Reclaim</TableCell>
                    <TableCell align="right" sx={{ color: theme.palette.info.main }}>Purchase VAT Reclaim</TableCell>
                    <TableCell align="right" sx={{ color: theme.palette.error.main }}>Total VAT Payable</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {summaryLoading ? (
                    [...Array(4)].map((_, i) => (
                      <TableRow key={i}>
                        {[...Array(5)].map((__, j) => <TableCell key={j}><Skeleton width={100} height={20} /></TableCell>)}
                      </TableRow>
                    ))
                  ) : summaryError ? (
                    <TableRow>
                      <TableCell colSpan={5} align="center" sx={{ py: 4 }}>
                        <Typography variant="body2" color="error">Error: {summaryError?.message}</Typography>
                      </TableCell>
                    </TableRow>
                  ) : (
                    summaryData?.map((item, idx) => {
                      const isSelected = selectedSummaryRow === idx;
                      const netPayable = (item?.summary?.by_transaction?.sales?.vat_amount || 0) - ((item?.summary?.by_transaction?.overhead?.vat_amount || 0) + (item?.summary?.by_transaction?.stock?.vat_amount || 0));
                      return (
                        <TableRow
                          key={idx}
                          onClick={() => setSelectedSummaryRow(idx)}
                          sx={{
                            ...bodyRowSx(theme),
                            cursor: 'pointer',
                            bgcolor: isSelected ? alpha(theme.palette.primary.main, 0.06) : undefined,
                            borderLeft: isSelected ? `3px solid ${theme.palette.primary.main}` : '3px solid transparent'
                          }}
                        >
                          <TableCell align="center" sx={{ fontWeight: 700, color: theme.palette.primary.main }}>{item?.label}</TableCell>
                          <TableCell align="right" sx={{ fontWeight: 600 }}>{formatUKCurrency(item.summary?.by_transaction?.sales?.vat_amount || 0)}</TableCell>
                          <TableCell align="right" sx={{ fontWeight: 600, color: theme.palette.success.main }}>{formatUKCurrency(item.summary?.by_transaction?.overhead?.vat_amount || 0)}</TableCell>
                          <TableCell align="right" sx={{ fontWeight: 600, color: theme.palette.info.main }}>{formatUKCurrency(item.summary?.by_transaction?.stock?.vat_amount || 0)}</TableCell>
                          <TableCell align="right" sx={{ fontWeight: 700, color: netPayable >= 0 ? theme.palette.error.main : theme.palette.success.main }}>{formatUKCurrency(netPayable)}</TableCell>
                        </TableRow>
                      );
                    })
                  )}
                </TableBody>
              </>
            )}

            {/* ── VAT Return tab ── */}
            {mainTab === 1 && (
              <>
                <TableHead>
                  <TableRow sx={theadRowSx(theme)}>
                    <TableCell sx={{ width: '8%' }}>BOX</TableCell>
                    <TableCell>VAT Return Categories</TableCell>
                    <TableCell align="right" sx={{ color: theme.palette.primary.main }}>{vatPeriod}</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {loading ? (
                    [...Array(9)].map((_, i) => (
                      <TableRow key={i}>
                        <TableCell><Skeleton width={28} /></TableCell>
                        <TableCell><Skeleton width="90%" /></TableCell>
                        <TableCell align="right"><Skeleton width={100} sx={{ ml: 'auto' }} /></TableCell>
                      </TableRow>
                    ))
                  ) : error ? (
                    <TableRow>
                      <TableCell colSpan={3} align="center" sx={{ py: 4 }}>
                        <Typography variant="body2" color="error">Error: {error?.message || 'Failed to load VAT data'}</Typography>
                      </TableCell>
                    </TableRow>
                  ) : (
                    vatCalculationsData.map((row, idx) => {
                      const isTotal = row.box === '3' || row.box === '5';
                      return (
                        <TableRow
                          key={idx}
                          sx={{
                            ...bodyRowSx(theme),
                            ...(isTotal && { bgcolor: alpha(theme.palette.warning.main, 0.04) })
                          }}
                        >
                          <TableCell sx={{ fontWeight: 700, color: theme.palette.primary.main }}>{row.box}</TableCell>
                          <TableCell sx={{ fontWeight: isTotal ? 700 : 400 }}>{row.category}</TableCell>
                          <TableCell align="right" sx={{ fontWeight: isTotal ? 800 : 500, color: theme.palette.warning.main }}>{row.q2_2025}</TableCell>
                        </TableRow>
                      );
                    })
                  )}
                </TableBody>
              </>
            )}

            {/* ── Audit VAT tab ── */}
            {mainTab === 2 && (
              <>
                <TableHead>
                  <TableRow sx={theadRowSx(theme)}>
                    <TableCell sx={{ width: '6%' }}>BOX</TableCell>
                    <TableCell sx={{ width: '10%' }}>Source</TableCell>
                    <TableCell sx={{ width: '10%' }}>Date</TableCell>
                    <TableCell sx={{ width: '16%' }}>Name</TableCell>
                    <TableCell sx={{ width: '16%' }}>Details</TableCell>
                    <TableCell sx={{ width: '8%' }}>VAT Scheme</TableCell>
                    <TableCell align="right" sx={{ width: '12%', color: theme.palette.primary.main }}>Amount</TableCell>
                    <TableCell align="right" sx={{ width: '12%', color: theme.palette.primary.main }}>Net Amount</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {transactionsLoading ? (
                    [...Array(10)].map((_, i) => (
                      <TableRow key={i}>
                        {[...Array(8)].map((__, j) => <TableCell key={j}><Skeleton width={j < 2 ? 30 : 80} /></TableCell>)}
                      </TableRow>
                    ))
                  ) : transactionsError ? (
                    <TableRow>
                      <TableCell colSpan={8} align="center" sx={{ py: 4 }}>
                        <Typography variant="body2" color="error">Error: {transactionsError?.message || 'Failed to load transactions'}</Typography>
                      </TableCell>
                    </TableRow>
                  ) : transactionsData.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={8} align="center" sx={{ py: 4 }}>
                        <Empty description="No Data Found" />
                      </TableCell>
                    </TableRow>
                  ) : (
                    (() => {
                      const grouped = {};
                      transactionsData.forEach((row) => {
                        const box = row.box || '\u2014';
                        const source = row.source || 'sales';
                        const key = `${vatPeriod || 'N/A'}|${box}|${source}`;
                        if (!grouped[key]) grouped[key] = [];
                        grouped[key].push(row);
                      });

                      return Object.entries(grouped).map(([key, rows]) => {
                        const [, box] = key.split('|');
                        const groupVat = rows.reduce((s, r) => s + (r.vat_amount || 0), 0);
                        const groupNet = rows.reduce((s, r) => s + (r.net_amount || 0), 0);

                        return (
                          <React.Fragment key={key}>
                            {rows.map((row, i) => (
                              <TableRow key={`${key}-${i}`} sx={{ ...bodyRowSx(theme), borderBottom: i === rows.length - 1 ? `2px solid ${theme.palette.divider}` : undefined }}>
                                <TableCell sx={{ fontWeight: 700, color: theme.palette.primary.main }}>{i === 0 ? box : ''}</TableCell>
                                <TableCell sx={{ color: 'text.secondary' }}>{row.transaction || '\u2014'}</TableCell>
                                <TableCell sx={{ color: 'text.secondary' }}>{row.date ? new Date(row.date).toLocaleDateString('en-GB') : '\u2014'}</TableCell>
                                <TableCell>{row.name || '\u2014'}</TableCell>
                                <TableCell sx={{ color: 'text.secondary' }}>{row.details || '\u2014'}</TableCell>
                                <TableCell sx={{ color: 'text.secondary' }}>{row.vat_scheme || row.vat_type || 'Margin'}</TableCell>
                                <TableCell align="right" sx={{ fontWeight: 600, color: theme.palette.primary.main, bgcolor: alpha(theme.palette.primary.main, 0.03) }}>{formatUKCurrency(row.vat_amount || 0)}</TableCell>
                                <TableCell align="right" sx={{ fontWeight: 600, color: theme.palette.primary.main, bgcolor: alpha(theme.palette.primary.main, 0.03) }}>{formatUKCurrency(row.net_amount || 0)}</TableCell>
                              </TableRow>
                            ))}
                            {/* Subtotal */}
                            <TableRow sx={{ bgcolor: alpha(theme.palette.primary.main, 0.04), '& .MuiTableCell-root': { py: 1.25, fontSize: '0.84rem', fontWeight: 700, borderBottom: `2px solid ${theme.palette.divider}` } }}>
                              <TableCell colSpan={6} align="right">Subtotal:</TableCell>
                              <TableCell align="right" sx={{ color: theme.palette.primary.main, bgcolor: alpha(theme.palette.primary.main, 0.06) }}>{formatUKCurrency(groupVat)}</TableCell>
                              <TableCell align="right" sx={{ color: theme.palette.primary.main, bgcolor: alpha(theme.palette.primary.main, 0.06) }}>{formatUKCurrency(groupNet)}</TableCell>
                            </TableRow>
                          </React.Fragment>
                        );
                      });
                    })()
                  )}
                </TableBody>
              </>
            )}
          </Table>
        </TableContainer>
      </Card>
    </Box>
  );
}

export default Vat_Page;
