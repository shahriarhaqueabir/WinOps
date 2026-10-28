import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Activity, Database, Gauge, Timer } from 'lucide-react'
import { useBackend } from '@/hooks/useBackend'
import { useMetricsStore } from '@/stores/useMetricsStore'
import { getDashboardRenderStats, getIpcCallsPerMinute } from '@/lib/perfDiagnostics'
import { Panel, PanelHeader } from '@/components/ui/Panel'
import type { LiveMetricEvidence } from '@/types'

interface CollectorDiagnostic {
  id: string
  name: string
  enabled: boolean
  interval_ms: number
  last_run: string
  last_duration_ms: number
  last_error?: string
  error_count: number
}

interface PerformanceDiagnosticData {
  collectors: CollectorDiagnostic[]
  metric_queue: { dropped_samples: number; queue_depth: number; queue_capacity: number }
  ai_first_token: { last_ms: number; average_ms: number; samples: number }
}

interface AppResourceUsage {
  cpu_percent: number
  memory_mb: number
  goroutines: number
}

function metricFreshness(evidence?: LiveMetricEvidence) {
  if (!evidence?.available) return 'No sample'
  return `${evidence.age_ms ?? '—'} ms · ${evidence.fresh ? 'fresh' : 'stale'}${evidence.collector_error ? ` · ${evidence.collector_error}` : ''}`
}

export function PerformanceDiagnostics({ compact = false }: { compact?: boolean }) {
  const { call } = useBackend()
  const latestLiveEvent = useMetricsStore(s => s.latestLiveEvent)
  const [sampledAt, setSampledAt] = useState(Date.now())
  const [ipcCalls, setIpcCalls] = useState(() => getIpcCallsPerMinute())
  const [renderStats, setRenderStats] = useState(() => getDashboardRenderStats())
  const { data } = useQuery<PerformanceDiagnosticData>({
    queryKey: ['performance-diagnostics'],
    queryFn: async () => await call('App.GetPerformanceDiagnostics') as PerformanceDiagnosticData,
    staleTime: 30000,
    refetchInterval: 30000,
    refetchOnWindowFocus: false,
  })

  const { data: appUsage } = useQuery<AppResourceUsage>({
    queryKey: ['app-resource-usage'],
    queryFn: async () => await call('App.GetAppResourceUsage') as AppResourceUsage,
    staleTime: 10000,
    refetchInterval: 10000,
    refetchOnWindowFocus: false,
  })

  useEffect(() => {
    const timer = window.setInterval(() => {
      setSampledAt(Date.now())
      setIpcCalls(getIpcCallsPerMinute())
      setRenderStats(getDashboardRenderStats())
    }, 5000)
    return () => window.clearInterval(timer)
  }, [])

  const maxCollectorMs = Math.max(0, ...(data?.collectors ?? []).map(c => c.last_duration_ms || 0))
  const collectorErrors = (data?.collectors ?? []).reduce((sum, collector) => sum + collector.error_count, 0)
  const dropped = data?.metric_queue?.dropped_samples ?? 0

  if (compact) {
    return (
      <div className="mt-3 flex flex-wrap items-center gap-x-4 gap-y-1 text-[10px] font-mono text-text-faint" aria-label="Performance diagnostics summary">
        <span><Activity size={11} className="mr-1 inline" />IPC {ipcCalls}/min</span>
        <span><Gauge size={11} className="mr-1 inline" />Render {renderStats.lastMs.toFixed(1)} ms</span>
        <span><Timer size={11} className="mr-1 inline" />Collector max {maxCollectorMs} ms</span>
        <span><Database size={11} className="mr-1 inline" />DB drops {dropped}</span>
        <span>AI first token {data?.ai_first_token?.last_ms ? `${data.ai_first_token.last_ms.toFixed(0)} ms` : '—'}</span>
        <span>App CPU {appUsage?.cpu_percent?.toFixed(1) ?? '—'}% · RAM {(appUsage?.memory_mb ?? 0).toFixed(0)} MB</span>
      </div>
    )
  }

  return (
    <Panel padding="md" className="space-y-4" aria-label="Performance diagnostics details">
      <PanelHeader icon={<Activity size={18} />} title="Performance Diagnostics" subtitle={`Local rolling measurements · sampled ${new Date(sampledAt).toLocaleTimeString()}`} />
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-5">
        <Metric label="UI IPC calls / min" value={ipcCalls.toString()} detail="Rolling 60-second window" />
        <Metric label="Dashboard render" value={`${renderStats.lastMs.toFixed(1)} ms`} detail={`avg ${renderStats.averageMs.toFixed(1)} · max ${renderStats.maxMs.toFixed(1)} ms · n=${renderStats.samples}`} />
        <Metric label="Collector duration" value={`${maxCollectorMs} ms`} detail={`${collectorErrors} collection errors`} />
        <Metric label="DB queue drops" value={dropped.toString()} detail={`${data?.metric_queue?.queue_depth ?? 0}/${data?.metric_queue?.queue_capacity ?? 0} queued`} />
        <Metric label="App CPU / RAM" value={`${(appUsage?.cpu_percent ?? 0).toFixed(1)}% / ${(appUsage?.memory_mb ?? 0).toFixed(0)} MB`} detail={`${appUsage?.goroutines ?? 0} goroutines`} />
        <Metric label="AI first token" value={data?.ai_first_token?.samples ? `${data.ai_first_token.last_ms.toFixed(0)} ms` : '—'} detail={data?.ai_first_token?.samples ? `avg ${data.ai_first_token.average_ms.toFixed(0)} ms · n=${data.ai_first_token.samples}` : 'No streamed sample yet'} />
      </div>
      <div className="grid grid-cols-2 gap-2 text-[10px] font-mono text-text-faint lg:grid-cols-3">
        {[
          ['cpu', latestLiveEvent?.cpu],
          ['ram', latestLiveEvent?.memory],
          ['disk', latestLiveEvent?.disk],
          ['network', latestLiveEvent?.network],
          ['processes', latestLiveEvent?.processes_evidence],
          ['connections', latestLiveEvent?.connections_evidence],
        ].map(([label, metric]) => (
          <span key={String(label)} className="rounded-lg border border-border bg-panel-2 px-2 py-1 capitalize">{String(label)}: {metricFreshness(metric as LiveMetricEvidence | undefined)}</span>
        ))}
      </div>
      <div className="space-y-1">
        <h4 className="text-[10px] font-bold uppercase tracking-wider text-text-dim">Collectors</h4>
        {(data?.collectors ?? []).map(collector => (
          <div key={collector.id} className="grid grid-cols-[1fr_auto_auto] gap-3 border-b border-border/50 py-1.5 text-[10px]">
            <span className="truncate text-text">{collector.name}</span>
            <span className="font-mono text-text-dim">{collector.last_duration_ms} ms</span>
            <span className={collector.last_error ? 'text-danger' : 'text-text-faint'}>{collector.last_error || `errors ${collector.error_count}`}</span>
          </div>
        ))}
        {!data?.collectors?.length && <p className="text-xs text-text-faint">Collector diagnostics are not available yet.</p>}
      </div>
    </Panel>
  )
}

function Metric({ label, value, detail }: { label: string; value: string; detail: string }) {
  return (
    <div className="min-w-0 rounded-xl border border-border bg-panel-2 p-3">
      <p className="truncate text-[9px] font-bold uppercase tracking-wider text-text-faint">{label}</p>
      <p className="mt-1 text-lg font-black tabular-nums text-text">{value}</p>
      <p className="truncate text-[9px] text-text-dim">{detail}</p>
    </div>
  )
}
