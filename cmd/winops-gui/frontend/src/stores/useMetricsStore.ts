import { create } from 'zustand'
import type { DashboardData, AlertInfo, HealthScorePoint, LiveMetricsEvent } from '@/types'

interface TimelineEvent {
  id: string
  timestamp: string
  category: string
  level: string
  title: string
  detail: string
  module: string
  related?: string[]
  metadata?: Record<string, string>
}

interface MetricsState {
  latest: DashboardData | null
  alerts: AlertInfo[]
  timeline: TimelineEvent[]
  healthTrend: HealthScorePoint[]
  lastLiveMetricsAt: string | null
  liveMetricsSequence: number
  latestLiveEvent: LiveMetricsEvent | null
  history: {
    cpu: { time: string; value: number }[]
    memory: { time: string; value: number }[]
    disk: { time: string; value: number }[]
  }
  setMetrics: (data: DashboardData) => void
  setSnapshot: (snapshot: { metrics: DashboardData; alerts: AlertInfo[]; timeline: TimelineEvent[]; timestamp?: string }) => void
  applyLiveMetrics: (event: Omit<LiveMetricsEvent, 'schema_version'> & { schema_version?: number }) => void
}

export const useMetricsStore = create<MetricsState>((set) => ({
  latest: null,
  alerts: [],
  timeline: [],
  healthTrend: [],
  lastLiveMetricsAt: null,
  liveMetricsSequence: 0,
  latestLiveEvent: null,
  history: {
    cpu: [],
    memory: [],
    disk: [],
  },
  setMetrics: (data) => {
    const time = new Date().toLocaleTimeString('en-US', { hour12: false })
    set((state) => ({
      latest: data,
      healthTrend: data.health_trend ?? [],
      history: {
        cpu: [...state.history.cpu.slice(-59), { time, value: data.cpu.value }],
        memory: [...state.history.memory.slice(-59), { time, value: data.memory.value }],
        disk: [...state.history.disk.slice(-59), { time, value: data.disk.value }],
      }
    }))
  },
  setSnapshot: (snap) => {
    if (!snap?.metrics) return
    const time = new Date().toLocaleTimeString('en-US', { hour12: false })
    set((state) => {
      const snapshotAt = snap.timestamp
      const snapshotIsOlder = !!snapshotAt && !!state.lastLiveMetricsAt && Date.parse(snapshotAt) < Date.parse(state.lastLiveMetricsAt)
      const metrics = snapshotIsOlder && state.latest
        ? { ...snap.metrics, cpu: state.latest.cpu, memory: state.latest.memory, disk: state.latest.disk, network: state.latest.network, processes: state.latest.processes, connections: state.latest.connections }
        : snap.metrics
      return {
        latest: metrics,
        alerts: snap.alerts ?? [],
        timeline: snap.timeline ?? [],
        healthTrend: metrics.health_trend ?? [],
        history: {
          cpu: [...state.history.cpu.slice(-59), { time, value: metrics.cpu.value }],
          memory: [...state.history.memory.slice(-59), { time, value: metrics.memory.value }],
          disk: [...state.history.disk.slice(-59), { time, value: metrics.disk.value }],
        },
      }
    })
  },
  applyLiveMetrics: (event) => {
    set((state) => {
      if (!state.latest || (event.schema_version != null && event.schema_version !== 1) || event.sequence <= state.liveMetricsSequence) return state
      const time = new Date(event.timestamp).toLocaleTimeString('en-US', { hour12: false })
      return {
        latest: {
          ...state.latest,
          cpu: { ...state.latest.cpu, value: event.cpu.value, unit: event.cpu.unit, trend: event.cpu.trend },
          memory: { ...state.latest.memory, value: event.memory.value, unit: event.memory.unit, trend: event.memory.trend },
          disk: { ...state.latest.disk, value: event.disk.value, unit: event.disk.unit, trend: event.disk.trend },
          network: { ...state.latest.network, ...event.network },
          processes: event.processes,
          connections: event.connections,
        },
        lastLiveMetricsAt: event.timestamp,
        liveMetricsSequence: event.sequence,
        latestLiveEvent: event as LiveMetricsEvent,
        history: {
          cpu: [...state.history.cpu.slice(-59), { time, value: event.cpu.value }],
          memory: [...state.history.memory.slice(-59), { time, value: event.memory.value }],
          disk: [...state.history.disk.slice(-59), { time, value: event.disk.value }],
        },
      }
    })
  }
}))
