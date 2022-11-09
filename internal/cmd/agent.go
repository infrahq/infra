package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/repeat"
)

func newAgentCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "agent",
		Short:  "Start the Infra agent",
		Long:   "Start the Infra agent that runs to sync Infra state",
		Args:   NoArgs,
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			infraDir, err := initInfraHomeDir()
			if err != nil {
				return fmt.Errorf("infra home directory: %w", err)
			}
			logging.UseFileLogger(filepath.Join(infraDir, "agent.log"))

			group, ctx := errgroup.WithContext(context.Background())
			group.Go(func() error {
				backOff := &backoff.ExponentialBackOff{
					InitialInterval:     time.Minute,
					MaxInterval:         2 * time.Minute,
					RandomizationFactor: 0.05,
					Multiplier:          1.5,
					// MaxElapsedTime is the duration the infra agent should
					// continue to run without a successful sync. After this time
					// the agent will exit.
					MaxElapsedTime: 5 * time.Minute,
				}
				waiter := repeat.NewWaiter(backOff)
				for {
					if err := syncKubeConfig(ctx); err != nil {
						logging.L.Warn().Err(err).Msg("failed to sync kubeconfig")
					} else {
						waiter.Reset()
					}
					if err := waiter.Wait(ctx); err != nil {
						return err
					}
				}
			})
			// add the next agent task here

			logging.Infof("starting infra agent (%s)", internal.FullVersion())
			err = group.Wait()
			logging.L.Error().Err(err).Msg("infra agent exit")
			return err
		},
	}
}

// configAgentRunning checks if the agent process stored in config is still running
func configAgentRunning() (bool, error) {
	pid, err := readStoredAgentProcessID()
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// this is the first time the agent is running, suppress the error and continue
			logging.Debugf("%s", err.Error())
			return false, nil
		}
		return false, err
	}

	return processRunning(int32(pid))
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
	// find the location of the infra executable running
	// this ensures we run the agent for the correct version
	infraExe, err := os.Executable()
	if err != nil {
		return err
	}

	// use the current infra executable to start the agent
	cmd := exec.Command(infraExe, "agent")
	if err := cmd.Start(); err != nil {
		return err
	}

	logging.Debugf("agent started, pid: %d", cmd.Process.Pid)

	return writeAgentConfig(cmd.Process.Pid)
}

func readStoredAgentProcessID() (int, error) {
	infraDir, err := initInfraHomeDir()
	if err != nil {
		return 0, err
	}

	agentConfig, err := os.Open(filepath.Join(infraDir, "agent.pid"))
	if err != nil {
		return 0, err
	}
	defer agentConfig.Close()

	var pid int

	_, err = fmt.Fscanf(agentConfig, "%d\n", &pid)
	if err != nil {
		return 0, err
	}

	return pid, nil
}

// writeAgentProcessConfig saves details about the agent to config
func writeAgentConfig(pid int) error {
	infraDir, err := initInfraHomeDir()
	if err != nil {
		return err
	}

	agentConfig, err := os.Create(filepath.Join(infraDir, "agent.pid"))
	if err != nil {
		return err
	}
	defer agentConfig.Close()

	_, err = agentConfig.WriteString(fmt.Sprintf("%d\n", pid))
	if err != nil {
		return err
	}

	return nil
}

// syncKubeConfig updates the local kubernetes configuration from Infra grants
func syncKubeConfig(_ context.Context) error {
	client, err := defaultAPIClient()
	if err != nil {
		return fmt.Errorf("get api client: %w", err)
	}
	client.Name = "agent"

	user, destinations, grants, err := getUserDestinationGrants(client, "kubernetes")
	if err != nil {
		return fmt.Errorf("list grants: %w", err)
	}

	if err := writeKubeconfig(user, destinations, grants); err != nil {
		return fmt.Errorf("update kubeconfig file: %w", err)
	}

	logging.L.Info().Msg("finished kubeconfig sync")
	return nil
}
