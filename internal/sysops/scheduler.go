package sysops

import (
	"fmt"
	"strings"

	"github.com/shahriarhaqueabir/WinOps/internal/common"
)

// ScheduledTaskInfo holds info about a scheduled task.
type ScheduledTaskInfo struct {
	Name     string `json:"name"`
	Schedule string `json:"schedule"`
	Command  string `json:"command"`
	Enabled  bool   `json:"enabled"`
	NextRun  string `json:"next_run"`
}

// GetScheduledTasks returns all scheduled tasks (schtasks on Windows).
func GetScheduledTasks() ([]ScheduledTaskInfo, error) {
	return getSchtasks()
}

func getSchtasks() ([]ScheduledTaskInfo, error) {
	cmd := common.HiddenCommand("schtasks", "/query", "/fo", "LIST", "/v")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	var tasks []ScheduledTaskInfo
	blocks := strings.Split(string(output), "\n\n")
	for _, block := range blocks {
		if block == "" {
			continue
		}
		task := ScheduledTaskInfo{}
		for _, line := range strings.Split(block, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "TaskName:") {
				task.Name = strings.TrimPrefix(line, "TaskName: ")
			} else if strings.HasPrefix(line, "Next Run Time:") {
				task.NextRun = strings.TrimPrefix(line, "Next Run Time: ")
			} else if strings.HasPrefix(line, "Status:") {
				status := strings.TrimPrefix(line, "Status: ")
				task.Enabled = status == "Ready" || status == "Running"
			} else if strings.HasPrefix(line, "Task To Run:") {
				task.Command = strings.TrimPrefix(line, "Task To Run: ")
			}
		}
		if task.Name != "" {
			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

// RunScheduledTask executes a scheduled task by name.
func RunScheduledTask(name string) error {
	cmd := common.HiddenCommand("schtasks", "/run", "/tn", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run task %s: %v (output: %s)", name, err, string(output))
	}
	return nil
}
