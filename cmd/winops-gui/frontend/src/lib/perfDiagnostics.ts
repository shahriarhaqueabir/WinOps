const IPC_WINDOW_MS = 60_000
const MAX_RENDER_SAMPLES = 120

const ipcCallTimes: number[] = []
const renderDurations: number[] = []

export function recordIpcCall(now = Date.now()): void {
  ipcCallTimes.push(now)
  pruneIpcCalls(now)
}

export function getIpcCallsPerMinute(now = Date.now()): number {
  pruneIpcCalls(now)
  return ipcCallTimes.length
}

function pruneIpcCalls(now: number): void {
  const cutoff = now - IPC_WINDOW_MS
  while (ipcCallTimes.length > 0 && ipcCallTimes[0] <= cutoff) ipcCallTimes.shift()
}

export function recordDashboardRender(durationMs: number): void {
  if (!Number.isFinite(durationMs) || durationMs < 0) return
  renderDurations.push(durationMs)
  if (renderDurations.length > MAX_RENDER_SAMPLES) renderDurations.shift()
}

export function getDashboardRenderStats(): { lastMs: number; averageMs: number; maxMs: number; samples: number } {
  const samples = renderDurations.length
  if (samples === 0) return { lastMs: 0, averageMs: 0, maxMs: 0, samples: 0 }
  const total = renderDurations.reduce((sum, value) => sum + value, 0)
  return {
    lastMs: renderDurations[samples - 1],
    averageMs: total / samples,
    maxMs: Math.max(...renderDurations),
    samples,
  }
}
