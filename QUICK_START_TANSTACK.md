# TanStack Router Prototype - Quick Start Guide

This guide will help you quickly get started with the TanStack Router prototype and performance comparison.

## 🚀 Quick Start

### Option 1: Run Both Applications Simultaneously

```bash
# From the repository root
./start-comparison.sh
```

This will start:
- **Next.js** on http://localhost:3000
- **TanStack Router** on http://localhost:3001

### Option 2: Run Individually

**Next.js:**
```bash
cd mimir-aip-frontend
npm install
npm run dev
# Visit http://localhost:3000
```

**TanStack Router:**
```bash
cd mimir-aip-tanstack
npm install
npm run dev
# Visit http://localhost:3001
```

## 📊 Collecting Performance Metrics

1. **Visit the dashboard on both implementations:**
   - Next.js: http://localhost:3000/dashboard
   - TanStack: http://localhost:3001/dashboard

2. **Metrics are automatically collected** when you load the dashboard
   - Bundle sizes
   - Load times
   - Render performance
   - Memory usage

3. **View the comparison:**
   - Next.js: http://localhost:3000/performance
   - TanStack: http://localhost:3001/performance
   - Both show the same data from localStorage

## 🏗️ Build Production Versions

**Next.js:**
```bash
cd mimir-aip-frontend
npm run build
npm run start
```

**TanStack Router:**
```bash
cd mimir-aip-tanstack
npm run build
npm run preview
```

## 📦 Current Build Sizes

### TanStack Router (Vite)
```
dist/index.html                   0.47 kB │ gzip:   0.31 kB
dist/assets/index-EW3cM_v5.css   17.92 kB │ gzip:   3.93 kB
dist/assets/index-CRaSTSFr.js   321.81 kB │ gzip: 100.67 kB
────────────────────────────────────────────────────────────
Total:                          340.20 kB │ gzip: 104.91 kB
```

### Next.js
*(Run production build to measure)*

## 🎯 What's Implemented

### TanStack Router Prototype
- ✅ Root layout with navigation
- ✅ Dashboard page (identical to Next.js)
- ✅ Performance comparison page
- ✅ API client for backend communication
- ✅ Automatic performance tracking
- ✅ Tailwind CSS styling (same as Next.js)
- ✅ TypeScript with full type safety
- ✅ Production build configuration

### Performance Tracking
- ✅ Bundle size measurement (JS, CSS, Total)
- ✅ Runtime metrics (Load time, Data fetch, FCP)
- ✅ Side-by-side comparison table
- ✅ Historical metrics tracking
- ✅ JSON export functionality
- ✅ Automatic collection on page load

## 📁 Key Files

```
mimir-aip-tanstack/
├── README.md                    # Detailed TanStack documentation
├── src/
│   ├── routes/
│   │   ├── __root.tsx          # Root layout + navigation
│   │   ├── index.tsx           # Home (redirects to dashboard)
│   │   ├── dashboard.tsx       # Main dashboard page
│   │   └── performance.tsx     # Performance comparison
│   ├── lib/
│   │   ├── performance.ts      # Metrics collection utilities
│   │   ├── api.ts              # Backend API client
│   │   └── utils.ts            # Common utilities
│   └── components/
│       └── Card.tsx            # Reusable card component

mimir-aip-frontend/
├── src/
│   ├── app/
│   │   ├── dashboard/page.tsx  # Updated with performance tracking
│   │   └── performance/page.tsx # Performance comparison (same as TanStack)
│   └── lib/
│       └── performance.ts      # Metrics collection (shared logic)

# Documentation
├── MIGRATION_COMPARISON.md      # Comprehensive comparison analysis
└── start-comparison.sh          # Helper script to run both apps
```

## 🔍 Comparing the Implementations

### Routing
Both use file-based routing, but with different patterns:

**Next.js:**
```tsx
// app/dashboard/page.tsx
export default function DashboardPage() {
  // component code
}
```

**TanStack:**
```tsx
// routes/dashboard.tsx
export const Route = createFileRoute('/dashboard')({
  component: DashboardPage
})

function DashboardPage() {
  // component code
}
```

### Navigation
**Next.js:**
```tsx
import Link from 'next/link'
<Link href="/dashboard">Dashboard</Link>
```

**TanStack:**
```tsx
import { Link } from '@tanstack/react-router'
<Link to="/dashboard">Dashboard</Link>
```

### Data Fetching
Both implementations use the same pattern (client-side):
```tsx
useEffect(() => {
  const fetchData = async () => {
    const data = await getJobs()
    setJobs(data)
  }
  fetchData()
}, [])
```

## 📈 Performance Insights

### Expected Advantages - TanStack
- ✅ **Smaller bundles** (~40-50% reduction expected)
- ✅ **Faster dev server** (Vite's HMR)
- ✅ **Faster builds** (Vite vs Webpack/Turbopack)
- ✅ **Simpler architecture** (client-only)
- ✅ **Lower memory footprint**

### Expected Advantages - Next.js
- ✅ **Better SEO** (SSR/SSG capabilities)
- ✅ **Built-in optimizations** (Image, Font)
- ✅ **Full-stack features** (API routes)
- ✅ **More mature ecosystem**

## 🎓 Learning Resources

- [TanStack Router Docs](https://tanstack.com/router/latest)
- [Vite Documentation](https://vitejs.dev/)
- [Next.js Documentation](https://nextjs.org/)
- [Performance Metrics Explanation](https://web.dev/vitals/)

## ❓ FAQ

**Q: Why is the bundle smaller in TanStack?**
A: No SSR runtime overhead, simpler framework core, Vite's optimized bundling.

**Q: Can I use TanStack for production?**
A: Yes, but consider SEO needs. Best for dashboards, internal tools, authenticated apps.

**Q: What about the other pages (pipelines, models, etc.)?**
A: This is a prototype focusing on the dashboard. Full migration would include all pages.

**Q: How do I clear the performance metrics?**
A: Click the "Clear" button on the /performance page, or clear localStorage in browser dev tools.

**Q: Can I compare specific metrics over time?**
A: Yes! The performance page shows history of all collected metrics with timestamps.

## 🚨 Troubleshooting

**Port already in use:**
```bash
# Kill processes on ports 3000 or 3001
lsof -ti:3000 | xargs kill
lsof -ti:3001 | xargs kill
```

**Dependencies not installed:**
```bash
cd mimir-aip-frontend && npm install
cd ../mimir-aip-tanstack && npm install
```

**Build errors:**
```bash
# Clean and rebuild
cd mimir-aip-tanstack
rm -rf node_modules dist
npm install
npm run build
```

## 📝 Next Steps

1. Run both implementations
2. Collect performance metrics
3. Review the comparison at `/performance`
4. Read `MIGRATION_COMPARISON.md` for detailed analysis
5. Decide on migration approach

## 🤝 Contributing

This is a prototype for evaluation. Feedback welcome!

---

**Created:** 2026-01-13  
**Version:** 1.0  
**Purpose:** Next.js to TanStack Router migration evaluation
