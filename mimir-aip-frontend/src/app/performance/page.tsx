"use client";
import { useState, useEffect } from 'react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { BarChart3, Download, Trash2, RefreshCw } from 'lucide-react'
import { loadAllMetrics, clearAllMetrics, type PerformanceMetrics } from '@/lib/performance'

export default function PerformanceComparison() {
  const [metrics, setMetrics] = useState<Partial<PerformanceMetrics>[]>([])
  const [nextjsMetrics, setNextjsMetrics] = useState<Partial<PerformanceMetrics> | null>(null)
  const [tanstackMetrics, setTanstackMetrics] = useState<Partial<PerformanceMetrics> | null>(null)

  const loadMetrics = () => {
    const allMetrics = loadAllMetrics()
    setMetrics(allMetrics)
    
    // Get latest metrics for each framework
    const nextjs = allMetrics.filter(m => m.framework === 'nextjs').sort((a, b) => 
      new Date(b.timestamp!).getTime() - new Date(a.timestamp!).getTime()
    )[0]
    
    const tanstack = allMetrics.filter(m => m.framework === 'tanstack').sort((a, b) => 
      new Date(b.timestamp!).getTime() - new Date(a.timestamp!).getTime()
    )[0]
    
    setNextjsMetrics(nextjs || null)
    setTanstackMetrics(tanstack || null)
  }

  useEffect(() => {
    loadMetrics()
  }, [])

  const handleClearMetrics = () => {
    clearAllMetrics()
    loadMetrics()
  }

  const handleExport = () => {
    const data = JSON.stringify(metrics, null, 2)
    const blob = new Blob([data], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `performance-comparison-${Date.now()}.json`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }

  const calculateImprovement = (nextjsValue: number, tanstackValue: number): string => {
    if (!nextjsValue || !tanstackValue) return 'N/A'
    const improvement = ((nextjsValue - tanstackValue) / nextjsValue) * 100
    return `${improvement > 0 ? '+' : ''}${improvement.toFixed(1)}%`
  }

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold text-orange mb-2">Performance Comparison</h1>
          <p className="text-white/60">Next.js vs TanStack Router</p>
        </div>
        <div className="flex gap-2">
          <Button onClick={loadMetrics} variant="outline" className="flex items-center gap-2">
            <RefreshCw className="w-4 h-4" />
            Refresh
          </Button>
          <Button onClick={handleExport} className="flex items-center gap-2 bg-orange text-navy hover:bg-orange/90">
            <Download className="w-4 h-4" />
            Export
          </Button>
          <Button onClick={handleClearMetrics} variant="destructive" className="flex items-center gap-2">
            <Trash2 className="w-4 h-4" />
            Clear
          </Button>
        </div>
      </div>

      {/* Summary Stats */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <Card className="bg-navy border-blue p-6">
          <div className="flex items-center gap-3 mb-4">
            <BarChart3 className="w-6 h-6 text-blue" />
            <h2 className="text-xl font-bold text-white">Next.js</h2>
          </div>
          {nextjsMetrics ? (
            <div className="space-y-3">
              <MetricRow label="Bundle Size (JS)" value={formatBytes(nextjsMetrics.bundleSize?.js || 0)} />
              <MetricRow label="Bundle Size (CSS)" value={formatBytes(nextjsMetrics.bundleSize?.css || 0)} />
              <MetricRow label="Total Bundle" value={formatBytes(nextjsMetrics.bundleSize?.total || 0)} />
              <MetricRow label="Initial Load Time" value={`${nextjsMetrics.runtimeMetrics?.initialLoadTime?.toFixed(2) || 0}ms`} />
              <MetricRow label="Data Fetch Time" value={`${nextjsMetrics.runtimeMetrics?.dataFetchTime?.toFixed(2) || 0}ms`} />
              <MetricRow label="FCP" value={`${nextjsMetrics.renderMetrics?.firstContentfulPaint?.toFixed(2) || 0}ms`} />
            </div>
          ) : (
            <p className="text-white/40 text-center py-4">No metrics collected yet</p>
          )}
        </Card>

        <Card className="bg-navy border-orange p-6">
          <div className="flex items-center gap-3 mb-4">
            <BarChart3 className="w-6 h-6 text-orange" />
            <h2 className="text-xl font-bold text-white">TanStack Router</h2>
          </div>
          {tanstackMetrics ? (
            <div className="space-y-3">
              <MetricRow label="Bundle Size (JS)" value={formatBytes(tanstackMetrics.bundleSize?.js || 0)} />
              <MetricRow label="Bundle Size (CSS)" value={formatBytes(tanstackMetrics.bundleSize?.css || 0)} />
              <MetricRow label="Total Bundle" value={formatBytes(tanstackMetrics.bundleSize?.total || 0)} />
              <MetricRow label="Initial Load Time" value={`${tanstackMetrics.runtimeMetrics?.initialLoadTime?.toFixed(2) || 0}ms`} />
              <MetricRow label="Data Fetch Time" value={`${tanstackMetrics.runtimeMetrics?.dataFetchTime?.toFixed(2) || 0}ms`} />
              <MetricRow label="FCP" value={`${tanstackMetrics.renderMetrics?.firstContentfulPaint?.toFixed(2) || 0}ms`} />
            </div>
          ) : (
            <p className="text-white/40 text-center py-4">No metrics collected yet</p>
          )}
        </Card>
      </div>

      {/* Comparison Table */}
      {nextjsMetrics && tanstackMetrics && (
        <Card className="bg-navy border-orange/30 p-6">
          <h2 className="text-xl font-bold text-orange mb-4">Performance Comparison</h2>
          <div className="overflow-x-auto">
            <table className="w-full text-left">
              <thead>
                <tr className="border-b border-white/10">
                  <th className="pb-3 text-white/60">Metric</th>
                  <th className="pb-3 text-white/60">Next.js</th>
                  <th className="pb-3 text-white/60">TanStack</th>
                  <th className="pb-3 text-white/60">Improvement</th>
                </tr>
              </thead>
              <tbody className="text-white">
                <tr className="border-b border-white/5">
                  <td className="py-3">JS Bundle Size</td>
                  <td>{formatBytes(nextjsMetrics.bundleSize?.js || 0)}</td>
                  <td>{formatBytes(tanstackMetrics.bundleSize?.js || 0)}</td>
                  <td className={
                    (tanstackMetrics.bundleSize?.js || 0) < (nextjsMetrics.bundleSize?.js || 0) 
                      ? 'text-green-400' 
                      : 'text-red-400'
                  }>
                    {calculateImprovement(nextjsMetrics.bundleSize?.js || 0, tanstackMetrics.bundleSize?.js || 0)}
                  </td>
                </tr>
                <tr className="border-b border-white/5">
                  <td className="py-3">Total Bundle Size</td>
                  <td>{formatBytes(nextjsMetrics.bundleSize?.total || 0)}</td>
                  <td>{formatBytes(tanstackMetrics.bundleSize?.total || 0)}</td>
                  <td className={
                    (tanstackMetrics.bundleSize?.total || 0) < (nextjsMetrics.bundleSize?.total || 0) 
                      ? 'text-green-400' 
                      : 'text-red-400'
                  }>
                    {calculateImprovement(nextjsMetrics.bundleSize?.total || 0, tanstackMetrics.bundleSize?.total || 0)}
                  </td>
                </tr>
                <tr className="border-b border-white/5">
                  <td className="py-3">Initial Load Time</td>
                  <td>{(nextjsMetrics.runtimeMetrics?.initialLoadTime || 0).toFixed(2)}ms</td>
                  <td>{(tanstackMetrics.runtimeMetrics?.initialLoadTime || 0).toFixed(2)}ms</td>
                  <td className={
                    (tanstackMetrics.runtimeMetrics?.initialLoadTime || 0) < (nextjsMetrics.runtimeMetrics?.initialLoadTime || 0) 
                      ? 'text-green-400' 
                      : 'text-red-400'
                  }>
                    {calculateImprovement(nextjsMetrics.runtimeMetrics?.initialLoadTime || 0, tanstackMetrics.runtimeMetrics?.initialLoadTime || 0)}
                  </td>
                </tr>
                <tr className="border-b border-white/5">
                  <td className="py-3">Data Fetch Time</td>
                  <td>{(nextjsMetrics.runtimeMetrics?.dataFetchTime || 0).toFixed(2)}ms</td>
                  <td>{(tanstackMetrics.runtimeMetrics?.dataFetchTime || 0).toFixed(2)}ms</td>
                  <td className={
                    (tanstackMetrics.runtimeMetrics?.dataFetchTime || 0) < (nextjsMetrics.runtimeMetrics?.dataFetchTime || 0) 
                      ? 'text-green-400' 
                      : 'text-red-400'
                  }>
                    {calculateImprovement(nextjsMetrics.runtimeMetrics?.dataFetchTime || 0, tanstackMetrics.runtimeMetrics?.dataFetchTime || 0)}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </Card>
      )}

      {/* Instructions */}
      <Card className="bg-blue/10 border-blue p-6">
        <h3 className="text-lg font-bold text-orange mb-3">How to Collect Metrics</h3>
        <ol className="space-y-2 text-white/80 list-decimal list-inside">
          <li>Navigate to the dashboard page in both Next.js (port 3000) and TanStack (port 3001) implementations</li>
          <li>Performance metrics will be automatically collected and stored in localStorage</li>
          <li>Return to this page to view the comparison</li>
          <li>Click &quot;Refresh&quot; to reload metrics or &quot;Export&quot; to download as JSON</li>
        </ol>
      </Card>

      {/* All Metrics History */}
      {metrics.length > 0 && (
        <Card className="bg-navy border-blue/30 p-6">
          <h2 className="text-xl font-bold text-white mb-4">Metrics History ({metrics.length} entries)</h2>
          <div className="space-y-2 max-h-96 overflow-y-auto">
            {metrics.map((metric, index) => (
              <div key={index} className="p-3 bg-blue/10 rounded border border-white/5">
                <div className="flex items-center justify-between">
                  <span className={`font-semibold ${metric.framework === 'nextjs' ? 'text-blue' : 'text-orange'}`}>
                    {metric.framework?.toUpperCase()}
                  </span>
                  <span className="text-xs text-white/40">
                    {metric.timestamp ? new Date(metric.timestamp).toLocaleString() : 'N/A'}
                  </span>
                </div>
                <div className="mt-2 text-sm text-white/60">
                  Bundle: {formatBytes(metric.bundleSize?.total || 0)} | 
                  Load: {(metric.runtimeMetrics?.initialLoadTime || 0).toFixed(2)}ms
                </div>
              </div>
            ))}
          </div>
        </Card>
      )}
    </div>
  )
}

function MetricRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between p-2 bg-white/5 rounded">
      <span className="text-white/60 text-sm">{label}</span>
      <span className="text-white font-mono">{value}</span>
    </div>
  )
}
