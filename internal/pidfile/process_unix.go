//go:build !windows

package pidfile

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

func IsRunning(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return p.Signal(syscall.Signal(0)) == nil
}

func getChildPIDs(pid int) []int {
	out, err := exec.Command("pgrep", "-P", fmt.Sprintf("%d", pid)).Output()
	if err != nil {
		return nil
	}
	var pids []int
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		var p int
		if _, err := fmt.Sscanf(line, "%d", &p); err == nil {
			pids = append(pids, p)
		}
	}
	return pids
}

func getAllDescendants(pid int) []int {
	children := getChildPIDs(pid)
	all := append([]int{}, children...)
	for _, c := range children {
		all = append(all, getAllDescendants(c)...)
	}
	return all
}

// GracefulKill sends SIGTERM to the process and its entire process tree, polls
// at 100ms intervals until the leader exits or the timeout elapses, then falls
// back to SIGKILL.
func GracefulKill(pid int, timeout time.Duration) error {
	if !IsRunning(pid) {
		return nil
	}

	p, err := os.FindProcess(pid)
	if err != nil {
		return nil
	}

	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		pgid = pid
	}

	descendants := getAllDescendants(pid)

	_ = p.Signal(syscall.SIGTERM)
	_ = syscall.Kill(-pgid, syscall.SIGTERM)
	for _, d := range descendants {
		_ = syscall.Kill(d, syscall.SIGTERM)
	}

	deadline := time.After(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !IsRunning(pid) {
				killDescendants(pgid, descendants)
				return nil
			}
		case <-deadline:
			_ = syscall.Kill(-pgid, syscall.SIGKILL)
			for _, d := range descendants {
				_ = syscall.Kill(d, syscall.SIGKILL)
			}
			_, _ = p.Wait()
			return nil
		}
	}
}

func killDescendants(pgid int, descendants []int) {
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
	for _, d := range descendants {
		_ = syscall.Kill(d, syscall.SIGKILL)
	}
}
