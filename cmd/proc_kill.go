package cmd

import (
	"fmt"
	"os"
	"time"

	"mdc/internal/logger"
	"mdc/internal/pidfile"

	"github.com/spf13/cobra"
)

var (
	procKillConfigName string
	procKillPID        int
)

var procKillCmd = &cobra.Command{
	Use:   "kill",
	Short: "Kill background processes by config name or PID",
	Long: `Kill background processes tracked by mdc.

Use -c to kill all processes belonging to a config, or -p to kill a single process by PID.
This command can be used in YAML down commands; the runner automatically adds -c <config-name>.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		hasConfig := cmd.Flags().Changed("config")
		hasPID := cmd.Flags().Changed("pid")

		if !hasConfig && !hasPID {
			fmt.Fprintln(os.Stderr, "error: specify -c <config-name> or -p <PID>")
			os.Exit(1)
		}

		if hasConfig && hasPID {
			fmt.Fprintln(os.Stderr, "error: -c and -p cannot be used together")
			os.Exit(1)
		}

		if hasPID {
			killByPID(procKillPID)
		} else {
			killByConfig(procKillConfigName)
		}
	},
}

func killByPID(pid int) {
	configName, projectName, entry, err := pidfile.FindByPID(pid)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	logger.Stop(projectName, entry.Command, pid)

	_ = pidfile.GracefulKill(pid, 10*time.Second)

	if err := pidfile.RemoveEntry(configName, projectName, pid); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Warning: failed to remove PID entry: %v\n", err)
	}

	logger.Stopped(projectName)
}

func killByConfig(configName string) {
	if err := pidfile.KillAllWithCallback(configName, logger.Stop); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Warning: failed to kill processes: %v\n", err)
	}
}

func init() {
	procKillCmd.Flags().StringVarP(&procKillConfigName, "config", "c", "", "Config name to kill all processes for")
	procKillCmd.Flags().IntVarP(&procKillPID, "pid", "p", 0, "PID of the process to kill")
	procCmd.AddCommand(procKillCmd)
}
