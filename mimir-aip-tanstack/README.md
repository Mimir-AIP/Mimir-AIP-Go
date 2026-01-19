# Next.js to TanStack Router Migration - Performance Comparison

This directory contains a TanStack Router prototype implementation of the Mimir AIP frontend, built to compare performance with the existing Next.js implementation.

## Overview

The TanStack Router prototype demonstrates:
- Client-side routing with TanStack Router
- Vite-based build system
- Same UI/UX as Next.js version
- Built-in performance tracking and comparison tools

## Directory Structure

```
mimir-aip-tanstack/
├── src/
│   ├── routes/              # TanStack Router routes
│   │   ├── __root.tsx       # Root layout
│   │   ├── index.tsx        # Home redirect
│   │   ├── dashboard.tsx    # Dashboard page
│   │   └── performance.tsx  # Performance comparison page
│   ├── components/          # Reusable UI components
│   ├── lib/                 # Utilities and API client
│   │   ├── api.ts          # Backend API client
│   │   ├── performance.ts  # Performance tracking utilities
│   │   └── utils.ts        # Common utilities
│   ├── main.tsx            # Application entry point
│   └── index.css           # Global styles (Tailwind)
├── index.html              # HTML template
├── vite.config.ts          # Vite configuration
├── tsconfig.json           # TypeScript configuration
└── package.json            # Dependencies and scripts
```

## Running the Applications

### Next.js Version (Port 3000)
```bash
cd mimir-aip-frontend
npm install
npm run dev
```
Access at: http://localhost:3000

### TanStack Router Version (Port 3001)
```bash
cd mimir-aip-tanstack
npm install
npm run dev
```
Access at: http://localhost:3001

## Performance Comparison

### How to Collect Metrics

1. **Start both applications:**
   - Next.js on port 3000
   - TanStack on port 3001

2. **Navigate to dashboards:**
   - Visit http://localhost:3000/dashboard (Next.js)
   - Visit http://localhost:3001/dashboard (TanStack)
   - Performance metrics are automatically collected and stored in localStorage

3. **View comparison:**
   - Visit http://localhost:3000/performance (Next.js)
   - OR visit http://localhost:3001/performance (TanStack)
   - Both show the same comparison data from localStorage

### Metrics Collected

#### Bundle Size Metrics
- **JS Bundle Size**: Size of JavaScript files loaded
- **CSS Bundle Size**: Size of CSS files loaded
- **Total Bundle Size**: Combined size of all assets

#### Runtime Performance Metrics
- **Initial Load Time**: Time from navigation start to load complete
- **Data Fetch Time**: Time spent fetching data from API
- **First Contentful Paint (FCP)**: Time to first visual content
- **Time to Interactive (TTI)**: Time until page is fully interactive
- **Memory Usage**: JavaScript heap size used

### Expected Performance Differences

**TanStack Router Advantages:**
- ✅ Smaller bundle size (no SSR overhead)
- ✅ Faster client-side navigation
- ✅ More straightforward build output
- ✅ Lower memory footprint

**Next.js Advantages:**
- ✅ Better SEO (Server-Side Rendering)
- ✅ Built-in optimizations (Image, Font)
- ✅ API routes co-located
- ✅ More established ecosystem

## Build Comparison

### Next.js Build
```bash
cd mimir-aip-frontend
npm run build
```
- Generates `.next` directory with optimized pages
- Includes SSR and static generation
- Larger initial bundle but optimized per-page

### TanStack Build
```bash
cd mimir-aip-tanstack
npm run build
```
- Generates `dist` directory with static assets
- Pure client-side application
- Smaller total bundle size
- Current build output: ~310KB JS (gzip: ~98KB), ~15.5KB CSS (gzip: ~3.6KB)

## Key Differences

### Routing

**Next.js (File-based):**
```tsx
// app/dashboard/page.tsx
export default function DashboardPage() { ... }
```

**TanStack (File-based with explicit routes):**
```tsx
// routes/dashboard.tsx
export const Route = createFileRoute('/dashboard')({
  component: DashboardPage
})
```

### Data Fetching

**Next.js:**
- Can use Server Components for data fetching
- Client components use hooks/effects
- Built-in data caching

**TanStack:**
- Client-side data fetching only
- Can integrate TanStack Query for caching
- Simpler mental model (all client-side)

### Build Tools

**Next.js:**
- Webpack/Turbopack based
- Hot Module Replacement
- Built-in optimizations

**TanStack:**
- Vite based
- Ultra-fast HMR
- Modern ESM-first approach

## Migration Considerations

### Easy to Migrate
- ✅ Component code (mostly unchanged)
- ✅ Styling (Tailwind works the same)
- ✅ API calls (same fetch logic)
- ✅ Type definitions (TypeScript compatible)

### Requires Adaptation
- ⚠️ Server-side rendering → Client-side only
- ⚠️ Next.js Image → Standard img or library
- ⚠️ Next.js Font optimization → Manual font loading
- ⚠️ API routes → Separate backend service
- ⚠️ Middleware → Client-side or backend handling

### Migration Steps for Full App

1. **Set up TanStack Router project**
   - Initialize with Vite
   - Configure TanStack Router plugin
   - Set up Tailwind CSS

2. **Copy components**
   - Move UI components (mostly unchanged)
   - Update imports (@ alias, remove 'next/*')
   - Replace Next.js-specific components

3. **Convert pages to routes**
   - Create route files in `routes/` directory
   - Use `createFileRoute` for each page
   - Update navigation to use TanStack Link

4. **Update data fetching**
   - Remove Server Components if any
   - Use client-side data fetching
   - Consider TanStack Query for caching

5. **Configure build**
   - Set up Vite config
   - Configure proxy for API calls
   - Adjust TypeScript settings

## Performance Results

### Build Size Comparison (Production)

**TanStack Router:**
- JS: 310.17 KB (gzip: 97.87 KB)
- CSS: 15.50 KB (gzip: 3.59 KB)
- **Total: 325.67 KB (gzip: 101.46 KB)**

**Next.js:**
- (Run production build to measure)
- Expected: Larger due to SSR runtime
- Per-route code splitting may be better

### Runtime Performance
- Measured automatically when visiting dashboards
- View comparison at `/performance` route
- Export data as JSON for analysis

## Recommendations

### Use TanStack Router If:
- ✅ Building a pure SPA (Single Page Application)
- ✅ SEO is not critical
- ✅ Want smaller bundle sizes
- ✅ Prefer simpler, client-only architecture
- ✅ Need fast development iteration

### Use Next.js If:
- ✅ Need SSR or SSG for SEO
- ✅ Want built-in full-stack features
- ✅ Prefer convention over configuration
- ✅ Need API routes in same codebase
- ✅ Want broader ecosystem support

## Further Reading

- [TanStack Router Documentation](https://tanstack.com/router)
- [Next.js Documentation](https://nextjs.org/docs)
- [Vite Documentation](https://vitejs.dev)
- [Web Vitals](https://web.dev/vitals/)

## Notes

This is a prototype implementation focusing on the Dashboard page to demonstrate:
1. The feasibility of migration
2. Performance characteristics
3. Development experience differences

For a production migration, all pages would need to be converted, and additional considerations like authentication, error handling, and state management would need to be addressed.
