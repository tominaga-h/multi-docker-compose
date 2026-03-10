package runner

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"mdc/internal/config"
	"mdc/internal/logger"
	"mdc/internal/pidfile"
)

type projectCommands struct {
	Project  config.Project
	Commands []config.CommandItem
}

func DryRun(cfg *config.Config, action string) error {
	pcs, err := commandsForAction(cfg, action)
	if err != nil {
		return err
	}

	logger.DryRunHeader(action, cfg.ExecutionMode)

	var invalidPaths []string
	for _, pc := range pcs {
		var warning string
		if err := validateProjectPath(pc.Project); err != nil {
			warning = "⚠️ Not Found"
			invalidPaths = append(invalidPaths, fmt.Sprintf("project %q: %s", pc.Project.Name, pc.Project.Path))
		}
		logger.DryRunProject(pc.Project.Name, pc.Project.Path, pc.Commands, warning)
	}

	if len(invalidPaths) > 0 {
		return fmt.Errorf("dry-run detected invalid paths:\n  %s", strings.Join(invalidPaths, "\n  "))
	}
	return nil
}

func Run(cfg *config.Config, action string, configName string) error {
	pcs, err := commandsForAction(cfg, action)
	if err != nil {
		return err
	}

	switch cfg.ExecutionMode {
	case "sequential":
		return runSequential(pcs, configName)
	case "parallel":
		return runParallel(pcs, configName)
	default:
		return fmt.Errorf("unknown execution_mode: %q", cfg.ExecutionMode)
	}
}

func commandsForAction(cfg *config.Config, action string) ([]projectCommands, error) {
	result := make([]projectCommands, len(cfg.Projects))
	for i, p := range cfg.Projects {
		var cmds []config.CommandItem
		switch action {
		case "up":
			cmds = p.Commands.Up
		case "down":
			cmds = p.Commands.Down
		default:
			return nil, fmt.Errorf("unknown action: %q", action)
		}
		if len(cmds) == 0 {
			return nil, fmt.Errorf("project %q: no commands defined for %q", p.Name, action)
		}
		result[i] = projectCommands{Project: p, Commands: cmds}
	}
	return result, nil
}

func runSequential(pcs []projectCommands, configName string) error {
	for _, pc := range pcs {
		if err := validateProjectPath(pc.Project); err != nil {
			return err
		}
		for _, item := range pc.Commands {
			if err := execCommand(pc.Project, item, configName, false); err != nil {
				return err
			}
		}
		logger.ProjectDone(pc.Project.Name)
	}
	return nil
}

func runParallel(pcs []projectCommands, configName string) error {
	for _, pc := range pcs {
		if err := validateProjectPath(pc.Project); err != nil {
			return err
		}
	}

	var wg sync.WaitGroup
	errs := make([]error, len(pcs))

	for i, pc := range pcs {
		wg.Add(1)
		go func(idx int, pc projectCommands) {
			defer wg.Done()
			errs[idx] = runProjectBuffered(pc, configName)
		}(i, pc)
	}

	wg.Wait()

	var failures []string
	for _, err := range errs {
		if err != nil {
			failures = append(failures, err.Error())
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("some projects failed:\n  %s", strings.Join(failures, "\n  "))
	}
	return nil
}

func runProjectBuffered(pc projectCommands, configName string) error {
	for _, item := range pc.Commands {
		if err := execCommand(pc.Project, item, configName, true); err != nil {
			return err
		}
	}
	logger.ProjectDone(pc.Project.Name)
	return nil
}

func expandProcKill(command, configName string) string {
	trimmed := strings.TrimSpace(command)
	if trimmed == "mdc proc kill" {
		return trimmed + " -c " + configName
	}
	return command
}

func execCommand(p config.Project, item config.CommandItem, configName string, buffered bool) error {
	command := expandProcKill(item.Command, configName)
	logger.Start(p.Name, command)

	if item.Background {
		expanded := item
		expanded.Command = command
		return execBackgroundCommand(p, expanded, configName)
	}

	cmd := newShellCommand(command, p.Path)

	if hasPTYSupport() && isTerminal(os.Stdout) {
		return execForegroundPTY(p, item, cmd, buffered)
	}
	return execForegroundStd(p, item, cmd, buffered)
}

func execBackgroundCommand(p config.Project, item config.CommandItem, configName string) error {
	tmpLog, _ := pidfile.ProcLogTmpPath(configName, p.Name)
	pid, err := StartBackgroundProcess(item.Command, p.Path, tmpLog)
	if err != nil {
		logger.Error(p.Name, item.Command, err)
		return fmt.Errorf("project %q: background command %q failed to start: %w", p.Name, item.Command, err)
	}

	if _, err := pidfile.RenameProcLog(tmpLog, pid); err != nil {
		logger.Warn(p.Name, fmt.Sprintf("log rename failed: %v", err))
	}

	if err := pidfile.Append(configName, p.Name, pidfile.Entry{
		PID:     pid,
		Command: item.Command,
		Dir:     p.Path,
	}); err != nil {
		return fmt.Errorf("project %q: failed to save PID: %w", p.Name, err)
	}
	logger.Background(p.Name, item.Command, pid)
	return nil
}

// StartBackgroundProcess starts a detached background process and returns its PID.
// If logFile is non-empty, the command is wrapped with the `script` utility so
// that the child runs inside a PTY. This preserves ANSI color codes in the log.
func StartBackgroundProcess(command, dir, logFile string) (int, error) {
	var cmd *exec.Cmd
	if logFile != "" {
		if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil {
			return 0, fmt.Errorf("failed to create log directory: %w", err)
		}
		cmd = newScriptCommand(command, dir, logFile)
	} else {
		cmd = newShellCommand(command, dir)
		cmd.Stdin = nil
		cmd.Stdout = nil
		cmd.Stderr = nil
	}
	setSysProcAttr(cmd)

	if err := cmd.Start(); err != nil {
		return 0, err
	}

	if logFile != "" {
		waitForFile(logFile, 2*time.Second)
	}

	return cmd.Process.Pid, nil
}

// waitForFile polls until the file at path exists or the timeout elapses.
func waitForFile(path string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func execForegroundPTY(p config.Project, item config.CommandItem, cmd *exec.Cmd, buffered bool) error {
	if !buffered {
		logger.Border()
	}
	output, err := execWithPTY(cmd, buffered)
	if !buffered {
		logger.Border()
	}
	if err != nil {
		logger.Error(p.Name, item.Command, err)
		if buffered && output != "" {
			logger.Output(p.Name, output)
		}
		return fmt.Errorf("project %q: command %q failed: %w", p.Name, item.Command, err)
	}
	logger.Success(p.Name, item.Command)
	return nil
}

func execForegroundStd(p config.Project, item config.CommandItem, cmd *exec.Cmd, buffered bool) error {
	if buffered {
		var stdoutBuf, stderrBuf bytes.Buffer
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf

		if err := cmd.Run(); err != nil {
			logger.Error(p.Name, item.Command, err)
			logger.Output(p.Name, stderrBuf.String()+stdoutBuf.String())
			return fmt.Errorf("project %q: command %q failed: %w", p.Name, item.Command, err)
		}
	} else {
		logger.Border()
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		logger.Border()

		if err != nil {
			logger.Error(p.Name, item.Command, err)
			return fmt.Errorf("project %q: command %q failed: %w", p.Name, item.Command, err)
		}
	}
	logger.Success(p.Name, item.Command)
	return nil
}

func newShellCommand(cmdStr, dir string) *exec.Cmd {
	cmd := exec.Command("sh", "-c", cmdStr)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	return cmd
}

func validateProjectPath(p config.Project) error {
	info, err := os.Stat(p.Path)
	if err != nil {
		return fmt.Errorf("project %q: path %q does not exist: %w", p.Name, p.Path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("project %q: path %q is not a directory", p.Name, p.Path)
	}
	return nil
}
