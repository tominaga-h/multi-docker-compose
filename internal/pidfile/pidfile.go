package pidfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mdc/internal/config"
)

// BaseDir overrides the default PID directory for testing.
var BaseDir string

type Entry struct {
	PID     int    `json:"pid"`
	Command string `json:"command"`
	Dir     string `json:"dir"`
}

func baseDir() (string, error) {
	if BaseDir != "" {
		return BaseDir, nil
	}
	base, err := config.BaseMDCDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "pids"), nil
}

func procBaseDir() (string, error) {
	if BaseDir != "" {
		return filepath.Join(filepath.Dir(BaseDir), "proc"), nil
	}
	base, err := config.BaseMDCDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "proc"), nil
}

// ProcLogDir returns the log directory for a given config: ~/.config/mdc/proc/<config-name>.
func ProcLogDir(configName string) (string, error) {
	base, err := procBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, configName), nil
}

// ProcLogFilePath returns the log file path for a specific process.
// Path: ~/.config/mdc/proc/<config-name>/<project-name>/<pid>.log
func ProcLogFilePath(configName, projectName string, pid int) (string, error) {
	dir, err := ProcLogDir(configName)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, projectName, fmt.Sprintf("%d.log", pid)), nil
}

// ProcLogTmpPath returns a temporary log file path used before the PID is known.
// Path: ~/.config/mdc/proc/<config-name>/<project-name>/_pending.log
func ProcLogTmpPath(configName, projectName string) (string, error) {
	dir, err := ProcLogDir(configName)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, projectName, "_pending.log"), nil
}

// RenameProcLog renames the temporary log file to the final PID-based name.
func RenameProcLog(tmpPath string, pid int) (string, error) {
	if tmpPath == "" {
		return "", nil
	}
	finalPath := filepath.Join(filepath.Dir(tmpPath), fmt.Sprintf("%d.log", pid))
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return "", fmt.Errorf("failed to rename log %s -> %s: %w", tmpPath, finalPath, err)
	}
	return finalPath, nil
}

func Dir(configName string) (string, error) {
	base, err := baseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, configName), nil
}

func filePath(configName, projectName string) (string, error) {
	dir, err := Dir(configName)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, projectName+".json"), nil
}

func Append(configName, projectName string, entry Entry) error {
	entries, err := Load(configName, projectName)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	entries = append(entries, entry)
	return Save(configName, projectName, entries)
}

func Save(configName, projectName string, entries []Entry) error {
	path, err := filePath(configName, projectName)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func Load(configName, projectName string) ([]Entry, error) {
	path, err := filePath(configName, projectName)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func LoadAll(configName string) (map[string][]Entry, error) {
	dir, err := Dir(configName)
	if err != nil {
		return nil, err
	}
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	result := make(map[string][]Entry)
	for _, de := range dirEntries {
		if de.IsDir() || filepath.Ext(de.Name()) != ".json" {
			continue
		}
		name := strings.TrimSuffix(de.Name(), ".json")
		data, err := os.ReadFile(filepath.Join(dir, de.Name()))
		if err != nil {
			return nil, err
		}
		var items []Entry
		if err := json.Unmarshal(data, &items); err != nil {
			return nil, err
		}
		result[name] = items
	}
	return result, nil
}

func LoadAllConfigs() (map[string]map[string][]Entry, error) {
	base, err := baseDir()
	if err != nil {
		return nil, err
	}
	dirs, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	result := make(map[string]map[string][]Entry)
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		projects, err := LoadAll(d.Name())
		if err != nil {
			return nil, err
		}
		if len(projects) > 0 {
			result[d.Name()] = projects
		}
	}
	return result, nil
}

// StopFunc is called for each tracked process before it is killed.
type StopFunc func(projectName, command string, pid int)

const defaultGracefulTimeout = 10 * time.Second

func KillAll(configName string) error {
	return KillAllWithCallback(configName, nil)
}

// KillAllConfigs kills tracked processes for all configs.
func KillAllConfigs() error {
	return KillAllConfigsWithCallback(nil)
}

// KillAllConfigsWithCallback kills tracked processes for all configs and calls
// onStop before each process is terminated.
func KillAllConfigsWithCallback(onStop StopFunc) error {
	allConfigs, err := LoadAllConfigs()
	if err != nil {
		return err
	}
	var failures []string
	for configName := range allConfigs {
		if err := KillAllWithCallback(configName, onStop); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", configName, err))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("failed to kill some configs:\n  %s", strings.Join(failures, "\n  "))
	}
	return nil
}

func KillAllWithCallback(configName string, onStop StopFunc) error {
	projects, err := LoadAll(configName)
	if err != nil {
		return err
	}
	for projectName, entries := range projects {
		for _, e := range entries {
			if onStop != nil {
				onStop(projectName, e.Command, e.PID)
			}
			_ = GracefulKill(e.PID, defaultGracefulTimeout)
		}
	}
	dir, err := Dir(configName)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	logDir, err := ProcLogDir(configName)
	if err != nil {
		return err
	}
	_ = os.RemoveAll(logDir)
	return nil
}

// FindByPID searches all PID files and returns the config name, project name,
// and entry for the given PID. Returns an error if the PID is not found.
func FindByPID(pid int) (configName, projectName string, entry Entry, err error) {
	allConfigs, err := LoadAllConfigs()
	if err != nil {
		return "", "", Entry{}, err
	}
	for cn, projects := range allConfigs {
		for pn, entries := range projects {
			for _, e := range entries {
				if e.PID == pid {
					return cn, pn, e, nil
				}
			}
		}
	}
	return "", "", Entry{}, fmt.Errorf("no tracked process with PID %d", pid)
}

// RemoveEntry removes a single PID entry from the project's PID file.
// If no entries remain, the file is deleted. If no files remain in the
// config directory, the directory is also removed.
func RemoveEntry(configName, projectName string, pid int) error {
	entries, err := Load(configName, projectName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	filtered := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if e.PID != pid {
			filtered = append(filtered, e)
		}
	}

	if len(filtered) == 0 {
		path, err := filePath(configName, projectName)
		if err != nil {
			return err
		}
		_ = os.Remove(path)
		return removeEmptyConfigDir(configName)
	}

	return Save(configName, projectName, filtered)
}

func removeEmptyConfigDir(configName string) error {
	dir, err := Dir(configName)
	if err != nil {
		return err
	}
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	if len(dirEntries) == 0 {
		return os.Remove(dir)
	}
	return nil
}
