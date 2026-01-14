import { createFileRoute } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { Card } from '@/components/Card'
import { 
  GitBranch, 
  Network, 
  Brain, 
  Copy, 
  Play, 
  CheckCircle, 
  Clock,
  AlertCircle,
  TrendingUp,
  Activity
} from 'lucide-react'
import { getJobs, getRunningJobs, getRecentJobs, getPerformanceMetrics, type Job } from '@/lib/api'
import { getPerformanceTracker, saveMetricsToStorage } from '@/lib/performance'

export const Route = createFileRoute('/dashboard')({
  component: DashboardPage,
})

function DashboardPage() {
  const [jobs, setJobs] = useState<Job[]>([])
  const [runningJobs, setRunningJobs] = useState<Job[]>([])
  const [recentJobs, setRecentJobs] = useState<Job[]>([])
  const [metrics, setMetrics] = useState<Record<string, unknown>>({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let mounted = true
    const tracker = getPerformanceTracker('tanstack')
    
    async function fetchData() {
      try {
        console.log('[Dashboard] Fetching data...')
        setLoading(true)
        setError(null)
        
        tracker.mark('component-mount')
        tracker.mark('data-fetch-start')
        
        const [jobsData, runningData, recentData, metricsData] = await Promise.all([
          getJobs().catch(err => { console.error('[Dashboard] getJobs failed:', err); return []; }),
          getRunningJobs().catch(err => { console.error('[Dashboard] getRunningJobs failed:', err); return []; }),
          getRecentJobs().catch(err => { console.error('[Dashboard] getRecentJobs failed:', err); return []; }),
          getPerformanceMetrics().catch(err => { console.error('[Dashboard] getPerformanceMetrics failed:', err); return {}; }),
        ])
        
        tracker.mark('data-fetch-end')
        const fetchTime = tracker.measure('data-fetch-start', 'data-fetch-end')
        console.log(`[Dashboard] Data fetched in ${fetchTime.toFixed(2)}ms`)
        
        if (!mounted) return
        
        setJobs(jobsData)
        setRunningJobs(runningData)
        setRecentJobs(recentData)
        setMetrics(metricsData)
        
        // Collect and save performance metrics
        tracker.mark('render-complete')
        const perfMetrics = await tracker.collectMetrics()
        const bundleSize = await tracker.getBundleSize()
        
        saveMetricsToStorage({
          ...perfMetrics,
          bundleSize,
        })
        
        console.log('[TanStack] Performance metrics collected:', perfMetrics)
      } catch (err) {
        console.error('[Dashboard] Fetch error:', err)
        if (mounted) {
          setError(err instanceof Error ? err.message : 'Unknown error')
        }
      } finally {
        if (mounted) {
          setLoading(false)
        }
      }
    }
    
    fetchData()
    
    return () => {
      mounted = false
    }
  }, [])

  if (loading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
        <Card className="col-span-full bg-blue text-white border-orange p-8 text-center">
          <h2 className="text-2xl font-bold mb-2">Loading Dashboard...</h2>
          <p className="text-sm opacity-75">Fetching data from API</p>
        </Card>
      </div>
    )
  }
  
  if (error) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
        <Card className="col-span-full bg-red-900 text-white border-orange p-8">
          <h2 className="text-2xl font-bold mb-2">Error Loading Dashboard</h2>
          <p className="font-mono text-sm">{error}</p>
          <button 
            onClick={() => window.location.reload()} 
            className="mt-4 px-4 py-2 bg-orange text-navy rounded hover:bg-orange/90"
          >
            Retry
          </button>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-orange mb-2">Dashboard</h1>
        <p className="text-white/60">Welcome to Mimir AIP - Your Autonomous Intelligence Platform</p>
      </div>

      {/* Quick Actions */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
        <a href="/pipelines" data-test="dashboard-pipelines-link">
          <Card className="bg-gradient-to-br from-blue to-blue/80 border-blue hover:border-orange transition-all p-4 cursor-pointer group">
            <GitBranch className="w-8 h-8 text-orange mb-2 group-hover:scale-110 transition-transform" />
            <h3 className="font-semibold text-white">Pipelines</h3>
            <p className="text-xs text-white/60">Data ingestion</p>
          </Card>
        </a>
        <a href="/ontologies" data-test="dashboard-ontologies-link">
          <Card className="bg-gradient-to-br from-blue to-blue/80 border-blue hover:border-orange transition-all p-4 cursor-pointer group">
            <Network className="w-8 h-8 text-orange mb-2 group-hover:scale-110 transition-transform" />
            <h3 className="font-semibold text-white">Ontologies</h3>
            <p className="text-xs text-white/60">Knowledge schemas</p>
          </Card>
        </a>
        <a href="/models" data-test="dashboard-models-link">
          <Card className="bg-gradient-to-br from-blue to-blue/80 border-blue hover:border-orange transition-all p-4 cursor-pointer group">
            <Brain className="w-8 h-8 text-orange mb-2 group-hover:scale-110 transition-transform" />
            <h3 className="font-semibold text-white">ML Models</h3>
            <p className="text-xs text-white/60">Predictions</p>
          </Card>
        </a>
        <a href="/digital-twins" data-test="dashboard-digital-twins-link">
          <Card className="bg-gradient-to-br from-blue to-blue/80 border-blue hover:border-orange transition-all p-4 cursor-pointer group">
            <Copy className="w-8 h-8 text-orange mb-2 group-hover:scale-110 transition-transform" />
            <h3 className="font-semibold text-white">Digital Twins</h3>
            <p className="text-xs text-white/60">Simulations</p>
          </Card>
        </a>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <Card className="bg-navy border-blue p-6 hover:border-orange/50 transition-colors">
          <div className="flex items-center justify-between mb-4">
            <Activity className="w-10 h-10 text-blue-400 p-2 bg-blue-400/10 rounded-lg" />
            <span className="text-xs text-white/40 uppercase tracking-wide">Total</span>
          </div>
          <p className="text-4xl font-bold text-white mb-1">{jobs.length}</p>
          <p className="text-sm text-white/60">Total Jobs</p>
        </Card>
        
        <Card className="bg-navy border-blue p-6 hover:border-orange/50 transition-colors">
          <div className="flex items-center justify-between mb-4">
            <Play className="w-10 h-10 text-green-400 p-2 bg-green-400/10 rounded-lg" />
            <span className="text-xs text-green-400 uppercase tracking-wide">Active</span>
          </div>
          <p className="text-4xl font-bold text-white mb-1">{runningJobs.length}</p>
          <p className="text-sm text-white/60">Running Jobs</p>
        </Card>
        
        <Card className="bg-navy border-blue p-6 hover:border-orange/50 transition-colors">
          <div className="flex items-center justify-between mb-4">
            <Clock className="w-10 h-10 text-orange p-2 bg-orange/10 rounded-lg" />
            <span className="text-xs text-white/40 uppercase tracking-wide">Recent</span>
          </div>
          <p className="text-4xl font-bold text-white mb-1">{recentJobs.length}</p>
          <p className="text-sm text-white/60">Recent Jobs</p>
        </Card>
      </div>

      {/* Recent Activity & Metrics */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Recent Jobs List */}
        <Card className="bg-navy border-blue p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-xl font-bold text-orange">Recent Activity</h2>
          </div>
          {recentJobs.length > 0 ? (
            <div className="space-y-3">
              {recentJobs.slice(0, 5).map((job, i) => (
                <div key={job.id || i} className="flex items-center justify-between p-3 bg-blue/20 rounded-lg">
                  <div className="flex items-center gap-3">
                    {job.status === 'completed' ? (
                      <CheckCircle className="w-5 h-5 text-green-400" />
                    ) : job.status === 'failed' ? (
                      <AlertCircle className="w-5 h-5 text-red-400" />
                    ) : (
                      <Clock className="w-5 h-5 text-orange" />
                    )}
                    <span className="text-white truncate max-w-[200px]">{job.name || job.id}</span>
                  </div>
                  <span className="text-xs text-white/40">{job.status}</span>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-white/40 text-center py-8">No recent activity</p>
          )}
        </Card>

        {/* Performance Metrics */}
        <Card className="bg-navy border-blue p-6">
          <div className="flex items-center gap-2 mb-4">
            <TrendingUp className="w-5 h-5 text-orange" />
            <h2 className="text-xl font-bold text-orange">Performance Metrics</h2>
          </div>
          {Object.keys(metrics).length > 0 ? (
            <div className="space-y-3">
              {Object.entries(metrics).slice(0, 6).map(([key, value]) => (
                <div key={key} className="flex items-center justify-between p-3 bg-blue/20 rounded-lg">
                  <span className="text-white/60 text-sm capitalize">{key.replace(/_/g, ' ')}</span>
                  <span className="text-white font-mono">{String(value)}</span>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-white/40 text-center py-8">No metrics available</p>
          )}
        </Card>
      </div>

      {/* Autonomous Flow CTA */}
      <Card className="bg-gradient-to-r from-blue via-navy to-blue border-orange/30 p-8">
        <div className="flex flex-col md:flex-row items-center justify-between gap-4">
          <div>
            <h2 className="text-2xl font-bold text-orange mb-2">🚀 Start Autonomous Workflow</h2>
            <p className="text-white/60">Create a pipeline, then let Mimir automatically extract entities, train ML models, and create digital twins.</p>
          </div>
          <div className="flex gap-3">
            <a href="/pipelines" className="inline-flex items-center justify-center px-4 py-2 border border-orange text-orange hover:bg-orange hover:text-navy rounded transition-colors">
              Create Pipeline
            </a>
            <a href="/ontologies" className="inline-flex items-center justify-center px-4 py-2 bg-orange text-navy hover:bg-orange/90 rounded transition-colors">
              Create Ontology
            </a>
          </div>
        </div>
      </Card>
    </div>
  )
}
