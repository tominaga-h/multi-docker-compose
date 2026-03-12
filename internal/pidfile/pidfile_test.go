package pidfile

import (
	"os"
	"path/filepath"
	"testing"
)

func withTempBaseDir(t *testing.T) func() {
	t.Helper()
	dir := t.TempDir()
	old := BaseDir
	BaseDir = dir
	return func() { BaseDir = old }
}

func TestSaveAndLoad(t *testing.T) {
	cleanup := withTempBaseDir(t)
	defer cleanup()

	entries := []Entry{
		{PID: 1234, Command: "npm run dev"},
		{PID: 5678, Command: "make watch"},
	}

	if err := Save("myconfig", "frontend", entries); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load("myconfig", "frontend")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("len(loaded) = %d, want 2", len(loaded))
	}
	if loaded[0].PID != 1234 || loaded[0].Command != "npm run dev" {
		t.Errorf("loaded[0] = %+v, want {PID:1234, Command:npm run dev}", loaded[0])
	}
	if loaded[1].PID != 5678 || loaded[1].Command != "make watch" {
		t.Errorf("loaded[1] = %+v, want {PID:5678, Command:make watch}", loaded[1])
	}
}

func TestAppend(t *testing.T) {
	cleanup := withTempBaseDir(t)
	defer cleanup()

	if err := Append("cfg", "proj", Entry{PID: 100, Command: "cmd1"}); err != nil {
		t.Fatalf("Append() error: %v", err)
	}
	if err := Append("cfg", "proj", Entry{PID: 200, Command: "cmd2"}); err != nil {
		t.Fatalf("Append() error: %v", err)
	}

	loaded, err := Load("cfg", "proj")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("len(loaded) = %d, want 2", len(loaded))
	}
	if loaded[0].PID != 100 {
		t.Errorf("loaded[0].PID = %d, want 100", loaded[0].PID)
	}
	if loaded[1].PID != 200 {
		t.Errorf("loaded[1].PID = %d, want 200", loaded[1].PID)
	}
}

func TestLoadNonexistent(t *testing.T) {
	cleanup := withTempBaseDir(t)
	defer cleanup()

	_, err := Load("noconfig", "noproj")
	if err == nil {
		t.Fatal("Load() expected error for nonexistent file, got nil")
	}
	if !os.IsNotExist(err) {
		t.Errorf("expected os.IsNotExist error, got: %v", err)
	}
}

func TestLoadAll(t *testing.T) {
	cleanup := withTempBaseDir(t)
	defer cleanup()

	if err := Save("cfg", "projA", []Entry{{PID: 10, Command: "a"}}); err != nil {
		t.Fatal(err)
	}
	if err := Save("cfg", "projB", []Entry{{PID: 20, Command: "b"}}); err != nil {
		t.Fatal(err)
	}

	result, err := LoadAll("cfg")
	if err != nil {
		t.Fatalf("LoadAll() error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}
	if len(result["projA"]) != 1 || result["projA"][0].PID != 10 {
		t.Errorf("projA entries = %+v, want [{PID:10}]", result["projA"])
	}
	if len(result["projB"]) != 1 || result["projB"][0].PID != 20 {
		t.Errorf("projB entries = %+v, want [{PID:20}]", result["projB"])
	}
}

func TestLoadAllNonexistentDir(t *testing.T) {
	cleanup := withTempBaseDir(t)
	defer cleanup()

	result, err := LoadAll("nonexistent")
	if err != nil {
		t.Fatalf("LoadAll() error: %v", err)
	}
	if result != nil {
		t.Errorf("LoadAll() = %v, want nil for nonexistent dir", result)
	}
}

func TestLoadAllConfigs(t *testing.T) {
	cleanup := withTempBaseDir(t)
	defer cleanup()

	if err := Save("cfg1", "proj", []Entry{{PID: 1, Command: "x"}}); err != nil {
		t.Fatal(err)
	}
	if err := Save("cfg2", "proj", []Entry{{PID: 2, Command: "y"}}); err != nil {
		t.Fatal(err)
	}

	result, err := LoadAllConfigs()
	if err != nil {
		t.Fatalf("LoadAllConfigs() error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}
	if _, ok := result["cfg1"]; !ok {
		t.Error("expected cfg1 in result")
	}
	if _, ok := result["cfg2"]; !ok {
		t.Error("expected cfg2 in result")
	}
}

func TestKillAll(t *testing.T) {
	cleanup := withTempBaseDir(t)
	defer cleanup()

	// Save some PID entries (use PID 0 which won't match any real process)
	if err := Save("cfg", "proj", []Entry{{PID: 999999999, Command: "fake"}}); err != nil {
		t.Fatal(err)
	}

	if err := KillAll("cfg"); err != nil {
		t.Fatalf("KillAll() error: %v", err)
	}

	dir, err := Dir("cfg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("PID directory should be removed after KillAll, err = %v", err)
	}
}

func TestKillAllConfigs(t *testing.T) {
	cleanup := withTempBaseDir(t)
	defer cleanup()

	if err := Save("cfg1", "proj1", []Entry{{PID: 999999991, Command: "fake1"}}); err != nil {
		t.Fatal(err)
	}
	if err := Save("cfg2", "proj2", []Entry{{PID: 999999992, Command: "fake2"}}); err != nil {
		t.Fatal(err)
	}

	logDir1, err := ProcLogDir("cfg1")
	if err != nil {
		t.Fatal(err)
	}
	logDir2, err := ProcLogDir("cfg2")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(logDir1, "proj1"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(logDir2, "proj2"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logDir1, "proj1", "999999991.log"), []byte("log1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logDir2, "proj2", "999999992.log"), []byte("log2"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := KillAllConfigs(); err != nil {
		t.Fatalf("KillAllConfigs() error: %v", err)
	}

	pidDir1, err := Dir("cfg1")
	if err != nil {
		t.Fatal(err)
	}
	pidDir2, err := Dir("cfg2")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(pidDir1); !os.IsNotExist(err) {
		t.Errorf("cfg1 PID directory should be removed after KillAllConfigs, err = %v", err)
	}
	if _, err := os.Stat(pidDir2); !os.IsNotExist(err) {
		t.Errorf("cfg2 PID directory should be removed after KillAllConfigs, err = %v", err)
	}
	if _, err := os.Stat(logDir1); !os.IsNotExist(err) {
		t.Errorf("cfg1 log directory should be removed after KillAllConfigs, err = %v", err)
	}
	if _, err := os.Stat(logDir2); !os.IsNotExist(err) {
		t.Errorf("cfg2 log directory should be removed after KillAllConfigs, err = %v", err)
	}
}

func TestKillAllNonexistentConfig(t *testing.T) {
	cleanup := withTempBaseDir(t)
	defer cleanup()

	if err := KillAll("nonexistent"); err != nil {
		t.Fatalf("KillAll() should not error for nonexistent config, got: %v", err)
	}
}

func TestIsRunning(t *testing.T) {
	// Current process should be running
	if !IsRunning(os.Getpid()) {
		t.Error("IsRunning(os.Getpid()) = false, want true")
	}

	// A very large PID should not be running
	if IsRunning(999999999) {
		t.Error("IsRunning(999999999) = true, want false")
	}
}

func TestDir(t *testing.T) {
	cleanup := withTempBaseDir(t)
	defer cleanup()

	dir, err := Dir("myconfig")
	if err != nil {
		t.Fatalf("Dir() error: %v", err)
	}
	if filepath.Base(dir) != "myconfig" {
		t.Errorf("Dir() base = %q, want %q", filepath.Base(dir), "myconfig")
	}
}

func TestProcLogDir(t *testing.T) {
	cleanup := withTempBaseDir(t)
	defer cleanup()

	dir, err := ProcLogDir("myconfig")
	if err != nil {
		t.Fatalf("ProcLogDir() error: %v", err)
	}
	if filepath.Base(dir) != "myconfig" {
		t.Errorf("ProcLogDir() base = %q, want %q", filepath.Base(dir), "myconfig")
	}
	parent := filepath.Base(filepath.Dir(dir))
	if parent != "proc" {
		t.Errorf("ProcLogDir() parent = %q, want %q", parent, "proc")
	}
}

func TestProcLogFilePath(t *testing.T) {
	cleanup := withTempBaseDir(t)
	defer cleanup()

	path, err := ProcLogFilePath("cfg", "proj", 12345)
	if err != nil {
		t.Fatalf("ProcLogFilePath() error: %v", err)
	}
	if filepath.Base(path) != "12345.log" {
		t.Errorf("ProcLogFilePath() base = %q, want %q", filepath.Base(path), "12345.log")
	}
	if filepath.Base(filepath.Dir(path)) != "proj" {
		t.Errorf("ProcLogFilePath() project dir = %q, want %q", filepath.Base(filepath.Dir(path)), "proj")
	}
}

func TestProcLogTmpPath(t *testing.T) {
	cleanup := withTempBaseDir(t)
	defer cleanup()

	path, err := ProcLogTmpPath("cfg", "proj")
	if err != nil {
		t.Fatalf("ProcLogTmpPath() error: %v", err)
	}
	if filepath.Base(path) != "_pending.log" {
		t.Errorf("ProcLogTmpPath() base = %q, want %q", filepath.Base(path), "_pending.log")
	}
}

func TestRenameProcLog(t *testing.T) {
	dir := t.TempDir()
	tmpPath := filepath.Join(dir, "_pending.log")
	if err := os.WriteFile(tmpPath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := RenameProcLog(tmpPath, 42)
	if err != nil {
		t.Fatalf("RenameProcLog() unexpected error: %v", err)
	}
	expected := filepath.Join(dir, "42.log")
	if result != expected {
		t.Errorf("RenameProcLog() = %q, want %q", result, expected)
	}
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("renamed file should exist: %v", err)
	}
}

func TestRenameProcLogEmpty(t *testing.T) {
	result, err := RenameProcLog("", 42)
	if err != nil {
		t.Fatalf("RenameProcLog(\"\") unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("RenameProcLog(\"\") = %q, want empty", result)
	}
}

func TestRenameProcLogNonexistent(t *testing.T) {
	dir := t.TempDir()
	tmpPath := filepath.Join(dir, "_pending.log")

	result, err := RenameProcLog(tmpPath, 42)
	if err == nil {
		t.Fatal("RenameProcLog() expected error for nonexistent file, got nil")
	}
	if result != "" {
		t.Errorf("RenameProcLog() = %q, want empty on error", result)
	}
}

func TestKillAllCleansUpLogs(t *testing.T) {
	cleanup := withTempBaseDir(t)
	defer cleanup()

	if err := Save("cfg", "proj", []Entry{{PID: 999999999, Command: "fake"}}); err != nil {
		t.Fatal(err)
	}

	logDir, err := ProcLogDir("cfg")
	if err != nil {
		t.Fatal(err)
	}
	projLogDir := filepath.Join(logDir, "proj")
	if err := os.MkdirAll(projLogDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projLogDir, "999999999.log"), []byte("log"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := KillAll("cfg"); err != nil {
		t.Fatalf("KillAll() error: %v", err)
	}

	if _, err := os.Stat(logDir); !os.IsNotExist(err) {
		t.Errorf("log directory should be removed after KillAll, err = %v", err)
	}
}
