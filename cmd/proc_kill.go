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
	procKillAll        bool
)

var procKillCmd = &cobra.Command{
	Use:   "kill",
	Short: "Kill background processes by config name, PID, or all configs",
	Long: `Kill background processes tracked by mdc.

Use -c to kill all processes belonging to a config, -p to kill a single process by PID,
or --all to kill all tracked processes across all configs.
This command can be used in YAML down commands; the runner automatically adds -c <config-name>.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		hasConfig := cmd.Flags().Changed("config")
		hasPID := cmd.Flags().Changed("pid")
		hasAll := cmd.Flags().Changed("all")

		selected := 0
		if hasConfig {
			selected++
		}
		if hasPID {
			selected++
		}
		if hasAll {
			selected++
		}

		if selected == 0 {
			fmt.Fprintln(os.Stderr, "error: specify one of -c <config-name>, -p <PID>, or --all")
			os.Exit(1)
		}

		if selected > 1 {
			fmt.Fprintln(os.Stderr, "error: -c, -p, and --all cannot be used together")
			os.Exit(1)
		}

		if hasPID {
			killByPID(procKillPID)
		} else if hasAll {
			killAllConfigs()
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

func killAllConfigs() {
	if err := pidfile.KillAllConfigsWithCallback(logger.Stop); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Warning: failed to kill processes: %v\n", err)
	}
}

func init() {
	procKillCmd.Flags().StringVarP(&procKillConfigName, "config", "c", "", "Config name to kill all processes for")
	procKillCmd.Flags().IntVarP(&procKillPID, "pid", "p", 0, "PID of the process to kill")
	procKillCmd.Flags().BoolVar(&procKillAll, "all", false, "Kill all tracked processes across all configs")
	procCmd.AddCommand(procKillCmd)
}
