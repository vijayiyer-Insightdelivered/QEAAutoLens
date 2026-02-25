import React, { useState, useEffect, useMemo } from 'react';
import {
  Card,
  CardContent,
  Typography,
  Grid,
  Box,
  Stack,
  FormControl,
  Select,
  MenuItem,
  Skeleton,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tabs,
  Tab,
  ToggleButton,
  ToggleButtonGroup,
  alpha,
  useTheme
} from '@mui/material';
import Pagination from '@mui/material/Pagination';
import {
  Inventory2Outlined,
  PointOfSaleOutlined,
  MonetizationOnOutlined,
  AccountTreeOutlined,
  CurrencyPound
} from '@mui/icons-material';
import QuestionAnswerIcon from '@mui/icons-material/QuestionAnswer';
import DoneAllIcon from '@mui/icons-material/DoneAll';
import CancelIcon from '@mui/icons-material/Cancel';
import TrendingUpIcon from '@mui/icons-material/TrendingUp';
import BarChartIcon from '@mui/icons-material/BarChart';
import TableChartIcon from '@mui/icons-material/TableChart';
import MainCard from '@components/MainCard';
import { useQuery } from '@tanstack/react-query';
import axios from 'axios';
import dayjs from 'dayjs';
import { ResponsiveContainer, BarChart, Bar, XAxis, YAxis, Tooltip, CartesianGrid, Cell } from 'recharts';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import useAuth from '@hooks/useAuth';
import { useNavigate } from 'react-router-dom';

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------
const GBP = (value, opts = {}) =>
  new Intl.NumberFormat('en-GB', { style: 'currency', currency: 'GBP', ...opts }).format(value);

const GBPCompact = (value) =>
  GBP(value, { notation: 'compact', maximumFractionDigits: 1 });

const parseDateSafe = (raw) => {
  if (!raw) return null;
  let d = dayjs(raw, ['DD/MM/YYYY', 'YYYY-MM-DD', 'YYYY-MM-DDTHH:mm:ss', dayjs.ISO_8601]);
  if (d.isValid()) return d;
  const parts = String(raw).split('/').map((n) => parseInt(n, 10));
  if (parts.length === 3 && !isNaN(parts[0]) && !isNaN(parts[1]) && !isNaN(parts[2])) {
    d = dayjs(`${parts[2]}-${parts[1]}-${parts[0]}`);
    if (d.isValid()) return d;
  }
  return null;
};

// ---------------------------------------------------------------------------
// Reusable style constants
// ---------------------------------------------------------------------------
const cardBase = {
  borderRadius: 3,
  transition: 'transform 0.2s ease, box-shadow 0.2s ease',
  '&:hover': {
    transform: 'translateY(-4px)',
    boxShadow: '0 12px 28px rgba(0,0,0,0.12)'
  }
};

const sectionTitle = {
  fontWeight: 700,
  letterSpacing: '-0.02em',
  mb: 0.5
};

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

/** KPI stat card with gradient icon badge and accent stripe */
const StatCard = ({ label, count, value, color, Icon, onClick }) => {
  const theme = useTheme();
  return (
    <Card
      onClick={onClick}
      sx={{
        ...cardBase,
        cursor: onClick ? 'pointer' : 'default',
        position: 'relative',
        overflow: 'hidden',
        height: '100%',
        bgcolor: 'background.paper',
        boxShadow: `0 1px 3px ${alpha(color, 0.12)}, 0 8px 24px ${alpha(color, 0.08)}`
      }}
    >
      {/* Top accent bar */}
      <Box
        sx={{
          position: 'absolute',
          top: 0,
          left: 0,
          right: 0,
          height: 4,
          background: `linear-gradient(90deg, ${color}, ${alpha(color, 0.4)})`
        }}
      />
      <CardContent sx={{ pt: 3 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
          <Box sx={{ flex: 1 }}>
            <Typography variant="caption" sx={{ color: 'text.secondary', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.08em', fontSize: '0.7rem' }}>
              {label}
            </Typography>
            <Typography variant="h3" sx={{ fontWeight: 800, mt: 0.5, lineHeight: 1.1 }}>
              {count}
            </Typography>
          </Box>
          <Box
            sx={{
              width: 48,
              height: 48,
              borderRadius: 2.5,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              background: `linear-gradient(135deg, ${color}, ${alpha(color, 0.7)})`,
              color: '#fff',
              boxShadow: `0 4px 14px ${alpha(color, 0.4)}`
            }}
          >
            <Icon sx={{ fontSize: 24 }} />
          </Box>
        </Box>

        {value !== undefined && (
          <Box sx={{ display: 'flex', alignItems: 'center', mt: 2, pt: 1.5, borderTop: `1px solid ${alpha(color, 0.1)}` }}>
            <CurrencyPound sx={{ fontSize: 16, color: 'text.secondary', mr: 0.5 }} />
            <Typography variant="body1" sx={{ fontWeight: 700, color: 'text.primary' }}>
              {GBPCompact(value)}
            </Typography>
            <Typography variant="caption" sx={{ color: 'text.secondary', ml: 1 }}>
              {GBP(value)}
            </Typography>
          </Box>
        )}
      </CardContent>
    </Card>
  );
};

/** Enquiry status card — slim variant with count + icon */
const EnquiryCard = ({ label, count, color, Icon, isLoading, onClick }) => (
  <Card
    onClick={onClick}
    sx={{
      ...cardBase,
      cursor: 'pointer',
      position: 'relative',
      overflow: 'hidden',
      height: '100%',
      bgcolor: 'background.paper',
      boxShadow: `0 1px 3px ${alpha(color, 0.12)}, 0 8px 24px ${alpha(color, 0.08)}`
    }}
  >
    <Box
      sx={{
        position: 'absolute',
        top: 0,
        left: 0,
        right: 0,
        height: 4,
        background: `linear-gradient(90deg, ${color}, ${alpha(color, 0.4)})`
      }}
    />
    <CardContent sx={{ pt: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
        <Box>
          <Typography variant="caption" sx={{ color: 'text.secondary', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.08em', fontSize: '0.7rem' }}>
            {label}
          </Typography>
          <Typography variant="h3" sx={{ fontWeight: 800, mt: 0.5, lineHeight: 1.1 }}>
            {isLoading ? <Skeleton width={48} /> : count}
          </Typography>
        </Box>
        <Box
          sx={{
            width: 48,
            height: 48,
            borderRadius: 2.5,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            background: `linear-gradient(135deg, ${color}, ${alpha(color, 0.7)})`,
            color: '#fff',
            boxShadow: `0 4px 14px ${alpha(color, 0.4)}`
          }}
        >
          <Icon sx={{ fontSize: 24 }} />
        </Box>
      </Box>
    </CardContent>
  </Card>
);

/** Custom Recharts tooltip */
const ChartTooltip = ({ active, payload, label }) => {
  if (!active || !payload?.length) return null;
  return (
    <Box
      sx={{
        bgcolor: 'background.paper',
        border: '1px solid',
        borderColor: 'divider',
        borderRadius: 2,
        px: 2,
        py: 1.5,
        boxShadow: '0 8px 32px rgba(0,0,0,0.12)'
      }}
    >
      <Typography variant="caption" sx={{ color: 'text.secondary', fontWeight: 600 }}>
        {label}
      </Typography>
      <Typography variant="body1" sx={{ fontWeight: 700, mt: 0.25 }}>
        {GBP(payload[0].value)}
      </Typography>
    </Box>
  );
};

// ---------------------------------------------------------------------------
// Main Dashboard
// ---------------------------------------------------------------------------
const Dashboard = () => {
  const theme = useTheme();
  const { user } = useAuth();
  const BaseUrl = import.meta.env.VITE_Base_URL;
  const navigate = useNavigate();

  // --- State ---
  const [chartType, setChartType] = useState('sales');
  const [tabValue, setTabValue] = useState(0);
  const [viewType, setViewType] = useState('chart');
  const [chartYear, setChartYear] = useState(null);
  const [salesData, setSalesData] = useState([]);
  const [selectedDate, setSelectedDate] = useState(null);
  const [sum, setSum] = useState(0);
  const [totalAll, setTotalAll] = useState(0);

  // Overheads pagination
  const getInitialOverheadsRowsPerPage = () => {
    const stored = localStorage.getItem('overheadsRowsPerPage');
    return stored ? parseInt(stored, 10) : 10;
  };
  const [overheadsPage, setOverheadsPage] = useState(1);
  const [overheadsRowsPerPage, setOverheadsRowsPerPage] = useState(getInitialOverheadsRowsPerPage());

  // Sales pagination
  const getInitialSalesRowsPerPage = () => {
    const stored = localStorage.getItem('salesRowsPerPage');
    return stored ? parseInt(stored, 10) : 10;
  };
  const [salesPage, setSalesPage] = useState(1);
  const [salesRowsPerPage, setSalesRowsPerPage] = useState(getInitialSalesRowsPerPage());

  const handleTabChange = (_, newValue) => {
    setTabValue(newValue);
    setChartType(['sales', 'overheads', 'cost'][newValue]);
    setChartYear(null);
    setViewType('chart');
  };

  const handleViewChange = (_, newView) => {
    if (newView !== null) setViewType(newView);
  };

  // --- Data fetching ---
  const { data: stockData, isLoading: isStockLoading } = useQuery({
    queryKey: ['stocks'],
    queryFn: async () => {
      const response = await axios.get(`${BaseUrl}stocks/getall/tid?tenant_info_id=${user?.tenantId}`);
      return response.data.details;
    }
  });

  const { data: saleskData } = useQuery({
    queryKey: ['Sales'],
    queryFn: async () => {
      const response = await axios.get(`${BaseUrl}sales/getall/tid?tenant_info_id=${user?.tenantId}`);
      return response.data.details;
    }
  });

  const { data: costkData } = useQuery({
    queryKey: ['cost'],
    queryFn: async () => {
      const response = await axios.get(`${BaseUrl}cost/getall/tid?tenant_info_id=${user?.tenantId}`);
      return response.data.details;
    }
  });

  const { data: overheadskData } = useQuery({
    queryKey: ['overheads'],
    queryFn: async () => {
      const response = await axios.get(`${BaseUrl}overhead/getall/tid?tenant_info_id=${user?.tenantId}`);
      return response.data.details;
    }
  });

  const { data: enquiryData, isLoading: isEnquiryLoading } = useQuery({
    queryKey: ['dashboard-enquiries'],
    queryFn: async () => {
      const response = await axios.get(`${BaseUrl}vehicle/enquiry/getall/tid?tenant_info_id=${user?.tenantId}`);
      return response.data.details || [];
    }
  });

  // --- Derived data ---
  const enquiryCounts = useMemo(() => {
    if (!Array.isArray(enquiryData)) return { Pending: 0, Responded: 0, Closed: 0 };
    return {
      Pending: enquiryData.filter((e) => e.enquiry_status === 'Pending').length,
      Responded: enquiryData.filter((e) => e.enquiry_status === 'Responded').length,
      Closed: enquiryData.filter((e) => e.enquiry_status === 'Closed').length
    };
  }, [enquiryData]);

  // Sales fetch + totals
  useEffect(() => {
    fetch(`${BaseUrl}sales/getall/tid?tenant_info_id=${user?.tenantId}`)
      .then((res) => res.json())
      .then((data) => {
        const details = data.details || [];
        setSalesData(details);
        const allTotal = details.reduce((acc, sale) => acc + parseFloat(sale.sold_price || 0), 0);
        setTotalAll(allTotal);
        if (!selectedDate) setSum(allTotal);
      })
      .catch(() => {
        setSalesData([]);
        setTotalAll(0);
        if (!selectedDate) setSum(0);
      });
  }, []);

  useEffect(() => {
    if (!salesData.length) { setSum(0); return; }
    if (!selectedDate) { setSum(totalAll); return; }
    const month = selectedDate.month() + 1;
    const year = selectedDate.year();
    const filtered = salesData.filter((sale) => {
      const d = dayjs(sale.sale_date, 'DD/MM/YYYY');
      return d.month() + 1 === month && d.year() === year;
    });
    setSum(filtered.reduce((acc, sale) => acc + parseFloat(sale.sold_price || 0), 0));
  }, [selectedDate, salesData, totalAll]);

  // Available years for year selector
  const availableYears = useMemo(() => {
    const yearsSet = new Set();
    const extractYears = (items, dateField) => {
      (items || []).forEach((item) => {
        const d = parseDateSafe(item[dateField]);
        if (d) yearsSet.add(d.year());
      });
    };
    if (chartType === 'sales') {
      const src = Array.isArray(salesData) && salesData.length ? salesData : Array.isArray(saleskData) ? saleskData : [];
      extractYears(src, 'sale_date');
    } else if (chartType === 'overheads') {
      extractYears(overheadskData, 'date');
    } else if (chartType === 'cost') {
      extractYears(costkData, 'date');
    }
    return Array.from(yearsSet).sort((a, b) => b - a);
  }, [chartType, salesData, saleskData, overheadskData, costkData]);

  useEffect(() => {
    if (availableYears.length) {
      setChartYear((prev) => (availableYears.includes(prev) ? prev : availableYears[0]));
    } else {
      setChartYear(null);
    }
  }, [availableYears]);

  // Chart data — monthly totals
  const chartData = useMemo(() => {
    const getDefaultYear = () => {
      const extractMax = (items, dateField) => {
        const years = (items || []).map((i) => parseDateSafe(i[dateField])?.year()).filter(Boolean);
        return years.length ? Math.max(...years) : null;
      };
      if (chartType === 'sales') {
        const src = Array.isArray(salesData) && salesData.length ? salesData : Array.isArray(saleskData) ? saleskData : [];
        return extractMax(src, 'sale_date') || dayjs().year();
      }
      if (chartType === 'overheads') return extractMax(overheadskData, 'date') || dayjs().year();
      if (chartType === 'cost') return extractMax(costkData, 'date') || dayjs().year();
      return dayjs().year();
    };

    const year = chartYear || (selectedDate ? selectedDate.year() : getDefaultYear());
    const months = Array.from({ length: 12 }, (_, i) => ({ monthIndex: i, month: dayjs().month(i).format('MMM'), value: 0 }));

    const accumulate = (items, dateField, valueField) => {
      (items || []).forEach((item) => {
        const d = parseDateSafe(item[dateField]);
        if (d && d.year() === year) {
          months[d.month()].value += parseFloat(item[valueField] || 0);
        }
      });
    };

    if (chartType === 'sales') {
      const src = Array.isArray(salesData) && salesData.length ? salesData : Array.isArray(saleskData) ? saleskData : [];
      accumulate(src, 'sale_date', 'sold_price');
    } else if (chartType === 'overheads') {
      accumulate(overheadskData, 'date', 'overhead_value');
    } else if (chartType === 'cost') {
      accumulate(costkData, 'date', 'cost_value');
    }

    return months.map((m) => ({ month: m.month, value: Math.round((m.value + Number.EPSILON) * 100) / 100 }));
  }, [salesData, saleskData, overheadskData, costkData, selectedDate, chartType, chartYear]);

  // Month-wise overheads for table view
  const monthWiseOverheads = useMemo(() => {
    if (!Array.isArray(overheadskData)) return [];
    const grouped = {};
    overheadskData.forEach((row) => {
      const d = parseDateSafe(row.date);
      if (!d) return;
      const key = d.format('YYYY-MM');
      if (!grouped[key]) grouped[key] = { month: d.format('MMM YYYY'), overhead: 0, vat: 0, total: 0 };
      const ov = parseFloat(row.overhead_value) || 0;
      const vt = parseFloat(row.vat) || 0;
      grouped[key].overhead += ov;
      grouped[key].vat += vt;
      grouped[key].total += ov + vt;
    });
    return Object.entries(grouped).sort(([a], [b]) => a.localeCompare(b)).map(([, val]) => val);
  }, [overheadskData]);

  // Month-wise sales for table view
  const monthWiseSales = useMemo(() => {
    const src = Array.isArray(salesData) && salesData.length ? salesData : Array.isArray(saleskData) ? saleskData : [];
    if (!src.length) return [];
    const grouped = {};
    src.forEach((sale) => {
      const d = parseDateSafe(sale.sale_date);
      if (!d) return;
      const key = d.format('YYYY-MM');
      if (!grouped[key]) grouped[key] = { month: d.format('MMM YYYY'), count: 0, total: 0 };
      grouped[key].count += 1;
      grouped[key].total += parseFloat(sale.sold_price || 0);
    });
    return Object.entries(grouped).sort(([a], [b]) => a.localeCompare(b)).map(([, val]) => val);
  }, [salesData, saleskData]);

  // Sales pagination persistence
  useEffect(() => {
    try { localStorage.setItem('salesRowsPerPage', String(salesRowsPerPage)); } catch {}
  }, [salesRowsPerPage]);

  const totalSalesPages = Math.max(1, Math.ceil((monthWiseSales?.length || 0) / (salesRowsPerPage || 1)));
  const paginatedMonthSales = (monthWiseSales || []).slice(
    (salesPage - 1) * salesRowsPerPage,
    salesPage * salesRowsPerPage
  );

  // Stock Age Analysis — bucket active stock by days in inventory
  const AGE_BUCKETS = [
    { label: '0–30 days', min: 0, max: 30, color: '#10B981' },
    { label: '31–60 days', min: 31, max: 60, color: '#3B82F6' },
    { label: '61–90 days', min: 61, max: 90, color: '#F59E0B' },
    { label: '91–120 days', min: 91, max: 120, color: '#F97316' },
    { label: '120+ days', min: 121, max: Infinity, color: '#EF4444' }
  ];

  const stockAgeData = useMemo(() => {
    const activeStock = (stockData || []).filter((d) => d.status == 1);
    if (!activeStock.length) return AGE_BUCKETS.map((b) => ({ ...b, count: 0, value: 0 }));

    const today = dayjs();
    const buckets = AGE_BUCKETS.map((b) => ({ ...b, count: 0, value: 0 }));

    activeStock.forEach((item) => {
      // Try common date fields for when the vehicle entered stock
      const raw = item.purchase_date || item.stock_in_date || item.stock_date || item.date || item.created_at;
      const d = parseDateSafe(raw);
      const days = d ? today.diff(d, 'day') : null;
      // If no date available, put into 120+ bucket as unknown age
      const age = days !== null ? days : Infinity;
      const bucket = buckets.find((b) => age >= b.min && age <= b.max);
      if (bucket) {
        bucket.count += 1;
        bucket.value += parseFloat(item.purchase_price || 0);
      }
    });

    return buckets;
  }, [stockData]);

  useEffect(() => {
    try { localStorage.setItem('overheadsRowsPerPage', String(overheadsRowsPerPage)); } catch {}
  }, [overheadsRowsPerPage]);

  const totalOverheadsPages = Math.max(1, Math.ceil((monthWiseOverheads?.length || 0) / (overheadsRowsPerPage || 1)));
  const paginatedMonthOverheads = (monthWiseOverheads || []).slice(
    (overheadsPage - 1) * overheadsRowsPerPage,
    overheadsPage * overheadsRowsPerPage
  );

  // --- Card data arrays ---
  const stockStats = [
    {
      label: 'Stock',
      count: stockData?.filter((d) => d.status == 1).length || 0,
      value: stockData?.filter((d) => d.status == 1).reduce((s, d) => s + parseFloat(d.purchase_price || 0), 0) || 0,
      color: '#3B82F6',
      Icon: Inventory2Outlined
    },
    {
      label: 'Reserve',
      count: stockData?.filter((d) => d.status == 3).length || 0,
      value: stockData?.filter((d) => d.status == 3).reduce((s, d) => s + parseFloat(d.purchase_price || 0), 0) || 0,
      color: '#10B981',
      Icon: PointOfSaleOutlined
    },
    {
      label: 'On Hold',
      count: stockData?.filter((d) => d.status == 4).length || 0,
      value: stockData?.filter((d) => d.status == 4).reduce((s, d) => s + parseFloat(d.purchase_price || 0), 0) || 0,
      color: '#F59E0B',
      Icon: MonetizationOnOutlined
    }
  ];

  const enquiryStats = [
    { label: 'Pending', count: enquiryCounts.Pending, color: '#F59E0B', Icon: QuestionAnswerIcon, navState: undefined },
    { label: 'Responded', count: enquiryCounts.Responded, color: '#3B82F6', Icon: DoneAllIcon, navState: { tab: 'Responded' } },
    { label: 'Closed', count: enquiryCounts.Closed, color: '#EF4444', Icon: CancelIcon, navState: { tab: 'Closed' } }
  ];

  const chartBarColor = { sales: '#3B82F6', overheads: '#8B5CF6', cost: '#F59E0B' }[chartType];

  // --- Render ---
  return (
    <Box sx={{ pb: 4 }}>
      {/* ── Inventory KPIs ── */}
      <Grid container spacing={2.5}>
        {stockStats.map((s) => (
          <Grid item xs={12} sm={6} md={4} key={s.label}>
            <StatCard {...s} />
          </Grid>
        ))}
      </Grid>

      {/* ── Enquiry Status ── */}
      <Box sx={{ mt: 4, mb: 2 }}>
        <Typography variant="h6" sx={sectionTitle}>
          Enquiry Status
        </Typography>
        <Typography variant="body2" sx={{ color: 'text.secondary', mb: 2 }}>
          Click a card to view details
        </Typography>
      </Box>

      <Grid container spacing={2.5}>
        {enquiryStats.map((e) => (
          <Grid item xs={12} sm={4} key={e.label}>
            <EnquiryCard
              {...e}
              isLoading={isEnquiryLoading}
              onClick={() => navigate('/enquiry', e.navState ? { state: e.navState } : undefined)}
            />
          </Grid>
        ))}
      </Grid>

      {/* ── Chart / Table Section ── */}
      <Card
        sx={{
          mt: 4,
          borderRadius: 3,
          boxShadow: '0 1px 3px rgba(0,0,0,0.06), 0 12px 32px rgba(0,0,0,0.08)',
          overflow: 'hidden'
        }}
      >
        {/* Header bar */}
        <Box
          sx={{
            px: 3,
            pt: 2.5,
            pb: 0,
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            flexWrap: 'wrap',
            gap: 2
          }}
        >
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
            <Box
              sx={{
                width: 36,
                height: 36,
                borderRadius: 2,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                background: `linear-gradient(135deg, ${chartBarColor}, ${alpha(chartBarColor, 0.6)})`,
                color: '#fff'
              }}
            >
              <TrendingUpIcon sx={{ fontSize: 20 }} />
            </Box>
            <Box>
              <Typography variant="h6" sx={{ fontWeight: 700, lineHeight: 1.2 }}>
                Financial Overview
              </Typography>
              <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                Monthly breakdown by category
              </Typography>
            </Box>
          </Box>

          {/* Year selector */}
          <FormControl size="small" sx={{ minWidth: 110 }}>
            <Select
              displayEmpty
              value={chartYear ?? ''}
              onChange={(e) => setChartYear(e.target.value === '' ? null : Number(e.target.value))}
              sx={{ borderRadius: 2, fontWeight: 600, fontSize: '0.85rem' }}
              renderValue={(selected) => (selected === '' ? 'Year' : selected)}
            >
              {availableYears.length === 0 ? (
                <MenuItem value=""><em>No years</em></MenuItem>
              ) : (
                availableYears.map((y) => <MenuItem key={y} value={y}>{y}</MenuItem>)
              )}
            </Select>
          </FormControl>
        </Box>

        {/* Tabs + view toggle */}
        <Box sx={{ px: 3, mt: 1.5, display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderBottom: 1, borderColor: 'divider' }}>
          <Tabs
            value={tabValue}
            onChange={handleTabChange}
            sx={{
              '& .MuiTab-root': {
                textTransform: 'none',
                fontWeight: 600,
                fontSize: '0.875rem',
                minHeight: 44
              },
              '& .Mui-selected': { color: chartBarColor },
              '& .MuiTabs-indicator': { backgroundColor: chartBarColor, height: 3, borderRadius: '3px 3px 0 0' }
            }}
          >
            <Tab label="Sales" />
            <Tab label="Overheads" />
            <Tab label="Cost" />
          </Tabs>

          {(tabValue === 0 || tabValue === 1) && (
            <ToggleButtonGroup
              value={viewType}
              exclusive
              onChange={handleViewChange}
              size="small"
              sx={{
                '& .MuiToggleButton-root': {
                  borderRadius: 1.5,
                  px: 1.5,
                  py: 0.5,
                  textTransform: 'none',
                  fontWeight: 600,
                  fontSize: '0.8rem',
                  gap: 0.5
                },
                '& .Mui-selected': {
                  bgcolor: `${alpha(chartBarColor, 0.1)} !important`,
                  color: `${chartBarColor} !important`
                }
              }}
            >
              <ToggleButton value="chart"><BarChartIcon sx={{ fontSize: 16 }} /> Chart</ToggleButton>
              <ToggleButton value="table"><TableChartIcon sx={{ fontSize: 16 }} /> Table</ToggleButton>
            </ToggleButtonGroup>
          )}
        </Box>

        <CardContent sx={{ px: 3, pt: 3, pb: 3 }}>
          {/* Bar Chart */}
          {(tabValue === 2 || viewType === 'chart') && (
            <Box sx={{ width: '100%', height: { xs: 300, md: 400 } }}>
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={chartData} margin={{ top: 24, right: 12, left: 0, bottom: 5 }}>
                  <defs>
                    <linearGradient id="barGradient" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor={chartBarColor} stopOpacity={1} />
                      <stop offset="100%" stopColor={chartBarColor} stopOpacity={0.6} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" vertical={false} stroke={alpha('#000', 0.06)} />
                  <XAxis
                    dataKey="month"
                    axisLine={false}
                    tickLine={false}
                    tick={{ fontSize: 12, fontWeight: 500, fill: theme.palette.text.secondary }}
                  />
                  <YAxis
                    axisLine={false}
                    tickLine={false}
                    tick={{ fontSize: 12, fill: theme.palette.text.secondary }}
                    tickFormatter={(val) => GBP(val, { maximumFractionDigits: 0, notation: 'compact' })}
                  />
                  <Tooltip content={<ChartTooltip />} cursor={{ fill: alpha(chartBarColor, 0.06), radius: 6 }} />
                  <Bar
                    dataKey="value"
                    fill="url(#barGradient)"
                    barSize={36}
                    radius={[8, 8, 0, 0]}
                    label={(props) => {
                      if (!props.value) return null;
                      return (
                        <text
                          x={props.x + props.width / 2}
                          y={props.y - 8}
                          fill={chartBarColor}
                          textAnchor="middle"
                          fontSize={11}
                          fontWeight={700}
                        >
                          {GBPCompact(props.value)}
                        </text>
                      );
                    }}
                  />
                </BarChart>
              </ResponsiveContainer>
            </Box>
          )}

          {/* Sales Table */}
          {tabValue === 0 && viewType === 'table' && (
            <>
              <TableContainer sx={{ borderRadius: 2, border: '1px solid', borderColor: 'divider', overflow: 'hidden' }}>
                <Table sx={{ minWidth: 650 }}>
                  <TableHead>
                    <TableRow
                      sx={{
                        background: `linear-gradient(135deg, ${alpha(chartBarColor, 0.08)}, ${alpha(chartBarColor, 0.03)})`,
                        '& .MuiTableCell-root': { fontWeight: 700, fontSize: '0.8rem', color: 'text.primary', letterSpacing: '0.02em', py: 1.5 }
                      }}
                    >
                      <TableCell>Month</TableCell>
                      <TableCell align="right">Vehicles Sold</TableCell>
                      <TableCell align="right">Total Revenue</TableCell>
                      <TableCell align="right">Avg. Sale Price</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {monthWiseSales.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={4} align="center" sx={{ py: 6, color: 'text.secondary' }}>
                          No data available
                        </TableCell>
                      </TableRow>
                    ) : (
                      paginatedMonthSales.map((row, idx) => (
                        <TableRow
                          key={idx}
                          sx={{
                            '&:nth-of-type(even)': { bgcolor: alpha(chartBarColor, 0.02) },
                            '&:hover': { bgcolor: alpha(chartBarColor, 0.05) },
                            transition: 'background-color 0.15s ease',
                            '& .MuiTableCell-root': { py: 1.5, fontSize: '0.85rem' }
                          }}
                        >
                          <TableCell sx={{ fontWeight: 600 }}>{row.month}</TableCell>
                          <TableCell align="right">{row.count}</TableCell>
                          <TableCell align="right">{GBP(row.total)}</TableCell>
                          <TableCell align="right" sx={{ fontWeight: 700 }}>{GBP(row.count > 0 ? row.total / row.count : 0)}</TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              </TableContainer>

              {/* Pagination */}
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mt: 2, px: 0.5 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <Typography variant="body2" sx={{ color: 'text.secondary', fontWeight: 500 }}>
                    Show:
                  </Typography>
                  <FormControl size="small">
                    <Select
                      value={salesRowsPerPage}
                      onChange={(e) => { setSalesRowsPerPage(parseInt(e.target.value, 10)); setSalesPage(1); }}
                      sx={{ minWidth: 64, borderRadius: 1.5, fontSize: '0.85rem' }}
                    >
                      <MenuItem value={5}>5</MenuItem>
                      <MenuItem value={10}>10</MenuItem>
                      <MenuItem value={25}>25</MenuItem>
                    </Select>
                  </FormControl>
                </Box>
                <Pagination
                  count={totalSalesPages}
                  page={salesPage}
                  onChange={(_, p) => setSalesPage(p)}
                  color="primary"
                  size="small"
                  sx={{
                    '& .MuiPaginationItem-root': {
                      borderRadius: 1.5,
                      fontWeight: 600,
                      minWidth: 32,
                      height: 32
                    }
                  }}
                />
              </Box>
            </>
          )}

          {/* Overheads Table */}
          {tabValue === 1 && viewType === 'table' && (
            <>
              <TableContainer sx={{ borderRadius: 2, border: '1px solid', borderColor: 'divider', overflow: 'hidden' }}>
                <Table sx={{ minWidth: 650 }}>
                  <TableHead>
                    <TableRow
                      sx={{
                        background: `linear-gradient(135deg, ${alpha(chartBarColor, 0.08)}, ${alpha(chartBarColor, 0.03)})`,
                        '& .MuiTableCell-root': { fontWeight: 700, fontSize: '0.8rem', color: 'text.primary', letterSpacing: '0.02em', py: 1.5 }
                      }}
                    >
                      <TableCell>Month</TableCell>
                      <TableCell align="right">Overhead (Exc VAT)</TableCell>
                      <TableCell align="right">VAT Reclaimable</TableCell>
                      <TableCell align="right">Total Overhead</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {monthWiseOverheads.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={4} align="center" sx={{ py: 6, color: 'text.secondary' }}>
                          No data available
                        </TableCell>
                      </TableRow>
                    ) : (
                      paginatedMonthOverheads.map((row, idx) => (
                        <TableRow
                          key={idx}
                          sx={{
                            '&:nth-of-type(even)': { bgcolor: alpha(chartBarColor, 0.02) },
                            '&:hover': { bgcolor: alpha(chartBarColor, 0.05) },
                            transition: 'background-color 0.15s ease',
                            '& .MuiTableCell-root': { py: 1.5, fontSize: '0.85rem' }
                          }}
                        >
                          <TableCell sx={{ fontWeight: 600 }}>{row.month}</TableCell>
                          <TableCell align="right">{GBP(row.overhead)}</TableCell>
                          <TableCell align="right">{GBP(row.vat)}</TableCell>
                          <TableCell align="right" sx={{ fontWeight: 700 }}>{GBP(row.total)}</TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              </TableContainer>

              {/* Pagination */}
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mt: 2, px: 0.5 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <Typography variant="body2" sx={{ color: 'text.secondary', fontWeight: 500 }}>
                    Show:
                  </Typography>
                  <FormControl size="small">
                    <Select
                      value={overheadsRowsPerPage}
                      onChange={(e) => { setOverheadsRowsPerPage(parseInt(e.target.value, 10)); setOverheadsPage(1); }}
                      sx={{ minWidth: 64, borderRadius: 1.5, fontSize: '0.85rem' }}
                    >
                      <MenuItem value={5}>5</MenuItem>
                      <MenuItem value={10}>10</MenuItem>
                      <MenuItem value={25}>25</MenuItem>
                    </Select>
                  </FormControl>
                </Box>
                <Pagination
                  count={totalOverheadsPages}
                  page={overheadsPage}
                  onChange={(_, p) => setOverheadsPage(p)}
                  color="primary"
                  size="small"
                  sx={{
                    '& .MuiPaginationItem-root': {
                      borderRadius: 1.5,
                      fontWeight: 600,
                      minWidth: 32,
                      height: 32
                    }
                  }}
                />
              </Box>
            </>
          )}
        </CardContent>
      </Card>

      {/* ── Stock Age Analysis ── */}
      <Card
        sx={{
          mt: 4,
          borderRadius: 3,
          boxShadow: '0 1px 3px rgba(0,0,0,0.06), 0 12px 32px rgba(0,0,0,0.08)',
          overflow: 'hidden'
        }}
      >
        <Box sx={{ px: 3, pt: 2.5, pb: 2, display: 'flex', alignItems: 'center', gap: 1.5, borderBottom: 1, borderColor: 'divider' }}>
          <Box
            sx={{
              width: 36,
              height: 36,
              borderRadius: 2,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              background: 'linear-gradient(135deg, #8B5CF6, rgba(139,92,246,0.6))',
              color: '#fff'
            }}
          >
            <AccessTimeIcon sx={{ fontSize: 20 }} />
          </Box>
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, lineHeight: 1.2 }}>
              Stock Age Analysis
            </Typography>
            <Typography variant="caption" sx={{ color: 'text.secondary' }}>
              How long current inventory has been in stock
            </Typography>
          </Box>
        </Box>

        <CardContent sx={{ px: 3, pt: 3, pb: 3 }}>
          {stockAgeData.every((b) => b.count === 0) ? (
            <Typography variant="body2" sx={{ color: 'text.secondary', textAlign: 'center', py: 6 }}>
              No active stock data available
            </Typography>
          ) : (
            <Grid container spacing={3}>
              {/* Bar chart */}
              <Grid item xs={12} md={8}>
                <Box sx={{ width: '100%', height: 280 }}>
                  <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={stockAgeData} layout="vertical" margin={{ top: 5, right: 30, left: 20, bottom: 5 }}>
                      <CartesianGrid strokeDasharray="3 3" horizontal={false} stroke={alpha('#000', 0.06)} />
                      <XAxis
                        type="number"
                        axisLine={false}
                        tickLine={false}
                        tick={{ fontSize: 12, fill: theme.palette.text.secondary }}
                        allowDecimals={false}
                      />
                      <YAxis
                        type="category"
                        dataKey="label"
                        axisLine={false}
                        tickLine={false}
                        width={100}
                        tick={{ fontSize: 12, fontWeight: 500, fill: theme.palette.text.secondary }}
                      />
                      <Tooltip
                        content={({ active, payload, label }) => {
                          if (!active || !payload?.length) return null;
                          const d = payload[0].payload;
                          return (
                            <Box
                              sx={{
                                bgcolor: 'background.paper',
                                border: '1px solid',
                                borderColor: 'divider',
                                borderRadius: 2,
                                px: 2,
                                py: 1.5,
                                boxShadow: '0 8px 32px rgba(0,0,0,0.12)'
                              }}
                            >
                              <Typography variant="caption" sx={{ color: 'text.secondary', fontWeight: 600 }}>
                                {label}
                              </Typography>
                              <Typography variant="body1" sx={{ fontWeight: 700, mt: 0.25 }}>
                                {d.count} vehicle{d.count !== 1 ? 's' : ''}
                              </Typography>
                              <Typography variant="body2" sx={{ color: 'text.secondary' }}>
                                {GBP(d.value)} total value
                              </Typography>
                            </Box>
                          );
                        }}
                        cursor={{ fill: alpha('#8B5CF6', 0.06) }}
                      />
                      <Bar dataKey="count" barSize={28} radius={[0, 8, 8, 0]}>
                        {stockAgeData.map((entry, index) => (
                          <Cell key={index} fill={entry.color} />
                        ))}
                      </Bar>
                    </BarChart>
                  </ResponsiveContainer>
                </Box>
              </Grid>

              {/* Summary cards */}
              <Grid item xs={12} md={4}>
                <Stack spacing={1.5}>
                  {stockAgeData.map((bucket) => (
                    <Box
                      key={bucket.label}
                      sx={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: 1.5,
                        p: 1.5,
                        borderRadius: 2,
                        border: '1px solid',
                        borderColor: 'divider',
                        transition: 'background-color 0.15s ease',
                        '&:hover': { bgcolor: alpha(bucket.color, 0.04) }
                      }}
                    >
                      <Box
                        sx={{
                          width: 10,
                          height: 10,
                          borderRadius: '50%',
                          bgcolor: bucket.color,
                          flexShrink: 0
                        }}
                      />
                      <Box sx={{ flex: 1, minWidth: 0 }}>
                        <Typography variant="body2" sx={{ fontWeight: 600, lineHeight: 1.2 }}>
                          {bucket.label}
                        </Typography>
                        <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                          {GBPCompact(bucket.value)}
                        </Typography>
                      </Box>
                      <Typography variant="h6" sx={{ fontWeight: 800, color: bucket.color }}>
                        {bucket.count}
                      </Typography>
                    </Box>
                  ))}
                </Stack>
              </Grid>
            </Grid>
          )}
        </CardContent>
      </Card>
    </Box>
  );
};

export default Dashboard;
