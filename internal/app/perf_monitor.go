package app

import (
	"os"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v4/process"
	"github.com/shahriarhaqueabir/Winops/internal/common"
)

// startPerfMonitor watches the app's own CPU usage and auto-throttles
// collection when it exceeds 30%, restoring standard mode when it drops
// below 20%. This prevents the dashboard from causing the spikes it
// reports on.
func (a *App) startPerfMonitor() {
	a.perfMonitorQuit = make(chan struct{})
	go func() {
		defer common.RecoverPanic()

		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		pid := int32(os.Getpid())
		proc, err := process.NewProcess(pid)
		if err != nil {
			common.LogWarn("PerfMonitor: cannot open self process: %v", err)
			return
		}

		throttled := false
		for {
			select {
			case <-ticker.C:
				cpuPct, err := proc.CPUPercent()
				if err != nil {
					continue
				}
				// cpuPct is cumulative-ish per gopsutil semantics; treat >30 as high
				if cpuPct > 30 && !throttled {
					common.LogInfo("PerfMonitor: self CPU %.1f%% > 30%%, applying eco profile", cpuPct)
					a.ApplyOperationalProfile("eco")
					throttled = true
				} else if cpuPct < 20 && throttled {
					common.LogInfo("PerfMonitor: self CPU %.1f%% < 20%%, restoring standard profile", cpuPct)
					a.ApplyOperationalProfile("standard")
					throttled = false
				}
			case <-a.perfMonitorQuit:
				return
			}
		}
	}()
}

// AppResourceUsage reports the application's own resource consumption.
type AppResourceUsage struct {
	CPUPercent float64 `json:"cpu_percent"`
	MemoryMB   float64 `json:"memory_mb"`
	Goroutines int     `json:"goroutines"`
}

// GetAppResourceUsage returns the app's own CPU/RAM usage for the
// performance indicator in the Dashboard.
func (a *App) GetAppResourceUsage() AppResourceUsage {
	out := AppResourceUsage{Goroutines: runtime.NumGoroutine()}
	pid := int32(os.Getpid())
	proc, err := process.NewProcess(pid)
	if err != nil {
		return out
	}
	if cpu, err := proc.CPUPercent(); err == nil {
		out.CPUPercent = cpu
	}
	if mem, err := proc.MemoryInfo(); err == nil && mem != nil {
		out.MemoryMB = float64(mem.RSS) / 1024 / 1024
	}
	return out
}
