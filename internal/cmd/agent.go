package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/process"
	"github.com/spf13/cobra"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/repeat"
)

// AgentConfig stores details about the agent in config separate from the CLI config file
// this config file must be separate to avoid concurrent writes with the CLI config
type AgentConfig struct {
	ProccessID int `json:"pid"` // used to manage the agent lifecycle
}

func newAgentCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "agent",
		Short:  "Start the Infra agent",
		Long:   "Start the Infra agent that runs to sync Infra state",
		Args:   NoArgs,
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var wg sync.WaitGroup

			// start background tasks
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			repeat.InGroup(&wg, ctx, cancel, 1*time.Minute, syncKubeConfig)
			// add the next agent task here

			wg.Wait()

			return ctx.Err()
		},
	}
}

// configAgentRunning checks if the agent process stored in config is still running
func configAgentRunning() (bool, error) {
	config, err := readAgentConfig()
	if err != nil {
		if os.IsNotExist(err) {
			// this is the first time the agent is running, suppress the error and continue
			logging.S.Debug(err)
			return false, nil
		}
		return false, err
	}

	return processRunning(int32(config.ProccessID))
}

func processRunning(pid int32) (bool, error) {
	if pid == 0 { // on windows pid 0 is the system idle process, it will be running but its not the agent
		return false, nil
	}

	running, err := process.PidExists(pid)
	if err != nil {
		return false, err
	}

	return running, nil
}

// execAgent executes the agent in the background and stores its process ID in configuration
func execAgent() error {
	cmd := exec.Command("infra", "agent")
	if err := cmd.Start(); err != nil {
		return err
	}

	return writeAgentConfig(AgentConfig{ProccessID: cmd.Process.Pid})
}

func readAgentConfig() (*AgentConfig, error) {
	config := &AgentConfig{}

	infraDir, err := infraHomeDir()
	if err != nil {
		return nil, err
	}

	contents, err := ioutil.ReadFile(filepath.Join(infraDir, "agent"))
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(contents, &config); err != nil {
		return nil, err
	}

	return config, nil
}

// writeAgentProcessConfig saves details about the agent to config
func writeAgentConfig(config AgentConfig) error {
	infraDir, err := infraHomeDir()
	if err != nil {
		return err
	}

	contents, err := json.Marshal(config)
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(filepath.Join(infraDir, "agent"), []byte(contents), 0o600); err != nil {
		return err
	}

	return nil
}

// syncKubeConfig updates the local kubernetes configuration from Infra grants
func syncKubeConfig(ctx context.Context, cancel context.CancelFunc) {
	user, destinations, grants, err := getUserDestinationGrants()
	if err != nil {
		fmt.Fprintf(os.Stderr, "agent failed to get user destination grants: %v\n", err)
		cancel()
	}
	if err := writeKubeconfig(user, destinations.Items, grants.Items); err != nil {
		fmt.Fprintf(os.Stderr, "agent failed to update kube config: %v\n", err)
		cancel()
	}
}
