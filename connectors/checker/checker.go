package checker

import (
	"fmt"

	"github.com/gwos/tcg/connectors/nsca/parser"
)

// ScheduleTask defines command
type ScheduleTask struct {
	CombinedOutput bool              `json:"combinedOutput,omitempty"`
	Command        []string          `json:"command"`
	Cron           string            `json:"cron"`
	DataFormat     parser.DataFormat `json:"dataFormat"`
	Environment    []string          `json:"environment,omitempty"`
}

func (t ScheduleTask) String() string {
	return fmt.Sprintf(
		"%s [%s] %v %v",
		t.DataFormat,
		t.Cron,
		t.Command,
		t.Environment,
	)
}

// ExtConfig defines the MonitorConnection extensions configuration
type ExtConfig struct {
	Schedule []ScheduleTask `json:"schedule"`
}

// Validate validates value
func (cfg ExtConfig) Validate() error {
	for _, task := range cfg.Schedule {
		if len(task.Command) == 0 {
			return fmt.Errorf("ExtConfig Schedule item error: Command is empty")
		}
	}
	return nil
}
