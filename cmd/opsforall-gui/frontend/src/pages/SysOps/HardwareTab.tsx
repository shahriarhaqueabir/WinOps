import { Monitor, Thermometer, Wind, Database, AlertTriangle } from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
import { useBackend } from '@/hooks/useBackend'
import { Panel, PanelHeader } from '@/components/ui/Panel'
import { StatCard } from '@/components/ui/StatCard'
import type { HardwareInfo, SensorData } from '@/types'
import { cn } from '@/lib/utils'

// ── Main Hardware Tab ────────────────────────────────────────────────────────

export function HardwareTab() {
  const { call } = useBackend()

  const { data: hw, isLoading, isError, error, refetch } = useQuery<HardwareInfo>({
    queryKey: ['sysops-hardware-info'],
    queryFn: async () => { const r = await call('SysOps.GetHardwareInfo'); return r as HardwareInfo },
    refetchInterval: 60000,
    retry: 2,
    retryDelay: 1000,
  })

  if (isLoading || !hw) {
    if (isError) {
      return (
        <div className="flex flex-col items-center justify-center h-64 text-[var(--color-text-faint)]">
          <AlertTriangle className="mb-4 opacity-40 text-[var(--color-danger)]" size={48} />
          <p className="text-xs font-black uppercase tracking-widest mb-2 text-[var(--color-danger)]">Telemetry Collection Failed</p>
          <p className="text-[10px] text-[var(--color-text-dim)] mb-4 max-w-sm text-center">
            {error instanceof Error ? error.message : 'Hardware telemetry query failed. This may indicate a system-level issue.'}
          </p>
          <button
            onClick={() => refetch()}
            className="px-4 py-2 text-xs font-bold uppercase tracking-wider rounded-lg border border-[var(--color-border)] hover:border-[var(--color-accent)] transition-colors"
          >
            Retry
          </button>
        </div>
      )
    }
    return (
      <div className="flex flex-col items-center justify-center h-64 text-[var(--color-text-faint)] animate-pulse">
        <Database className="mb-4 opacity-20" size={48} />
        <p className="text-xs font-black uppercase tracking-widest">Collecting Workstation Telemetry...</p>
      </div>
    )
  }

  return (
    <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500">

      <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
        {/* GPU Information */}
        <Panel variant="elevated" category="system" padding="lg">
          <PanelHeader icon={<Monitor size={20} />} title="Graphics Matrix" subtitle="GPU Hardware & Drivers" />
          {hw.gpu.detected ? (
            <div className="mt-6 space-y-4">
              <div className="flex items-center justify-between">
                <span className="text-sm font-bold text-[var(--color-text)]">{hw.gpu.name}</span>
                <span className="text-[10px] font-black text-accent border border-accent/30 rounded px-2 py-0.5 uppercase">{hw.gpu.vendor}</span>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <StatCard label="Video RAM" value={`${hw.gpu.memory_gb.toFixed(1)} GB`} />
                {hw.gpu.utilization > 0 && (
                  <StatCard label="Utilization" value={`${hw.gpu.utilization.toFixed(0)}%`} valueClassName="text-[var(--color-accent)]" />
                )}
                {hw.gpu.temperature > 0 && (
                  <StatCard label="Temperature" value={`${hw.gpu.temperature.toFixed(0)}°C`} valueClassName={cn(hw.gpu.temperature > 80 ? "text-danger" : "text-[var(--color-text)]")} />
                )}
                {hw.gpu.fan_speed > 0 && (
                  <StatCard label="Fan Speed" value={`${hw.gpu.fan_speed.toFixed(0)} RPM`} />
                )}
                {(hw.gpu.utilization === 0 && hw.gpu.temperature === 0 && hw.gpu.fan_speed === 0) && (
                  <div className="col-span-2 text-[10px] text-[var(--color-text-faint)] italic py-2">
                    Real-time metrics unavailable — install NVIDIA drivers for GPU monitoring.
                  </div>
                )}
              </div>
              <div className="pt-4 border-t border-[var(--color-border)]/50">
                <p className="text-[10px] font-black text-[var(--color-text-faint)] uppercase tracking-widest mb-1">Driver Version</p>
                <p className="text-xs font-mono text-[var(--color-text-dim)] truncate">{hw.gpu.driver}</p>
              </div>
            </div>
          ) : (
            <div className="mt-8 text-center py-6 border-2 border-dashed border-[var(--color-border)] rounded-2xl opacity-40">
              <p className="text-xs font-bold text-[var(--color-text-dim)] uppercase">No Discrete GPU Detected</p>
            </div>
          )}
        </Panel>

        {/* Thermal & Power */}
        <Panel variant="elevated" category="system" padding="lg">
          <PanelHeader icon={<Thermometer size={20} />} title="Thermal Envelope" subtitle="Sensors & Power Management" />
          <div className="mt-6 space-y-6">
            <div className="grid grid-cols-2 gap-6">
              <div className="flex flex-col gap-1">
                <span className="text-[10px] font-black text-[var(--color-text-faint)] uppercase tracking-widest">CPU Temperature</span>
                <div className="flex items-end gap-2">
                  <span className={cn(
                    "text-3xl font-black tabular-nums",
                    hw.cpu.temperature > 80 ? "text-danger" : hw.cpu.temperature > 65 ? "text-warning" : "text-accent"
                  )}>
                    {hw.cpu.temperature > 0 ? `${hw.cpu.temperature.toFixed(0)}°C` : '--'}
                  </span>
                  {hw.cpu.temperature > 0 && <span className="text-[10px] font-bold text-[var(--color-text-dim)] mb-1 uppercase">Package</span>}
                </div>
              </div>

              {hw.battery.detected && (
                <div className="flex flex-col gap-1">
                  <span className="text-[10px] font-black text-[var(--color-text-faint)] uppercase tracking-widest">Battery Status</span>
                  <div className="flex items-end gap-2">
                    <span className="text-3xl font-black text-[var(--color-success)] tabular-nums">{hw.battery.percent}%</span>
                    <span className="text-[10px] font-bold text-[var(--color-text-dim)] mb-1 uppercase">
                      {hw.battery.charging ? 'Charging' : 'Discharging'}
                    </span>
                  </div>
                </div>
              )}
            </div>

            <div className="h-px bg-[var(--color-border)]/50" />

            <div className="space-y-4">
              <p className="text-[10px] font-black text-[var(--color-text-faint)] uppercase tracking-widest">Active Hardware Sensors</p>
              {hw.sensors && hw.sensors.length > 0 ? (
                <div className="grid grid-cols-2 gap-4">
                  {hw.sensors.map((s: SensorData) => (
                    <div key={s.name} className="flex items-center justify-between p-3 rounded-xl bg-[var(--color-panel-2)] border border-[var(--color-border)]">
                      <div className="flex items-center gap-2">
                        {s.type === 'Fan' ? <Wind size={14} className="text-accent" /> : <Thermometer size={14} className="text-accent" />}
                        <span className="text-xs font-bold text-[var(--color-text)] truncate w-24">{s.name}</span>
                      </div>
                      <span className="text-xs font-black tabular-nums">{s.value > 0 ? `${s.value}${s.unit}` : 'Active'}</span>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="space-y-3">
                  <p className="text-[10px] text-[var(--color-text-dim)] italic">
                    No sensor data available. CPU temperature and fan data may require elevated privileges or vendor drivers.
                  </p>
                </div>
              )}
            </div>
          </div>
        </Panel>
      </div>

      {/* Motherboard / Chassis */}
      <Panel variant="default" category="none" padding="lg">
        <PanelHeader icon={<Database size={20} />} title="Motherboard & Baseboard" />
        <div className="mt-6 grid grid-cols-2 md:grid-cols-4 gap-8">
          <InfoItem label="Manufacturer" value={hw.baseboard.manufacturer} />
          <InfoItem label="Product" value={hw.baseboard.product} />
          <InfoItem label="Version" value={hw.baseboard.version} />
          <InfoItem label="Serial Number" value={hw.baseboard.serial_number} />
        </div>
      </Panel>
    </div>
  )
}

function InfoItem({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-[10px] font-black text-[var(--color-text-faint)] uppercase tracking-widest mb-1">{label}</p>
      <p className="text-sm font-bold text-[var(--color-text)] truncate">{value || 'Unknown'}</p>
    </div>
  )
}
