package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name: "absolute path unchanged",
			path: "/usr/local/bin",
			want: "/usr/local/bin",
		},
		{
			name: "relative path unchanged",
			path: "relative/path",
			want: "relative/path",
		},
		{
			name: "empty string unchanged",
			path: "",
			want: "",
		},
		{
			name: "tilde expands to home",
			path: "~/projects",
			want: filepath.Join(home, "projects"),
		},
		{
			name: "tilde only expands to home",
			path: "~",
			want: home,
		},
		{
			name: "tilde with nested path",
			path: "~/a/b/c",
			want: filepath.Join(home, "a", "b", "c"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandHome(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandHome(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExpandHome(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestContractHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "home directory itself",
			path: home,
			want: "~",
		},
		{
			name: "path under home",
			path: filepath.Join(home, ".config", "mdc", "project.yml"),
			want: filepath.Join("~", ".config", "mdc", "project.yml"),
		},
		{
			name: "absolute path outside home",
			path: "/usr/local/bin",
			want: "/usr/local/bin",
		},
		{
			name: "empty string unchanged",
			path: "",
			want: "",
		},
		{
			name: "relative path unchanged",
			path: "relative/path",
			want: "relative/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContractHome(tt.path)
			if got != tt.want {
				t.Errorf("ContractHome(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name: "valid sequential config",
			cfg: Config{
				ExecutionMode: "sequential",
				Projects:      []Project{{Name: "svc", Path: "/tmp"}},
			},
		},
		{
			name: "valid parallel config",
			cfg: Config{
				ExecutionMode: "parallel",
				Projects:      []Project{{Name: "svc", Path: "/tmp"}},
			},
		},
		{
			name: "invalid execution_mode",
			cfg: Config{
				ExecutionMode: "invalid",
				Projects:      []Project{{Name: "svc", Path: "/tmp"}},
			},
			wantErr: "execution_mode must be",
		},
		{
			name: "empty execution_mode",
			cfg: Config{
				ExecutionMode: "",
				Projects:      []Project{{Name: "svc", Path: "/tmp"}},
			},
			wantErr: "execution_mode must be",
		},
		{
			name: "no projects",
			cfg: Config{
				ExecutionMode: "parallel",
				Projects:      []Project{},
			},
			wantErr: "at least one project",
		},
		{
			name: "project missing name",
			cfg: Config{
				ExecutionMode: "parallel",
				Projects:      []Project{{Name: "", Path: "/tmp"}},
			},
			wantErr: "name is required",
		},
		{
			name: "project missing path",
			cfg: Config{
				ExecutionMode: "parallel",
				Projects:      []Project{{Name: "svc", Path: ""}},
			},
			wantErr: "path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("validate() unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Errorf("validate() expected error containing %q, got nil", tt.wantErr)
				return
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("validate() error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestLoadFromDir(t *testing.T) {
	t.Run("valid yaml with extension", func(t *testing.T) {
		dir := t.TempDir()
		yaml := `execution_mode: sequential
projects:
  - name: app
    path: /tmp
    commands:
      up: ["echo up"]
      down: ["echo down"]
`
		if err := os.WriteFile(filepath.Join(dir, "test.yml"), []byte(yaml), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadFromDir(dir, "test.yml")
		if err != nil {
			t.Fatalf("LoadFromDir() error: %v", err)
		}
		if cfg.ExecutionMode != "sequential" {
			t.Errorf("ExecutionMode = %q, want %q", cfg.ExecutionMode, "sequential")
		}
		if len(cfg.Projects) != 1 {
			t.Fatalf("len(Projects) = %d, want 1", len(cfg.Projects))
		}
		if cfg.Projects[0].Name != "app" {
			t.Errorf("Projects[0].Name = %q, want %q", cfg.Projects[0].Name, "app")
		}
	})

	t.Run("auto-appends .yml extension", func(t *testing.T) {
		dir := t.TempDir()
		yaml := `execution_mode: parallel
projects:
  - name: svc
    path: /tmp
    commands:
      up: ["echo up"]
`
		if err := os.WriteFile(filepath.Join(dir, "dev.yml"), []byte(yaml), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadFromDir(dir, "dev")
		if err != nil {
			t.Fatalf("LoadFromDir() error: %v", err)
		}
		if cfg.ExecutionMode != "parallel" {
			t.Errorf("ExecutionMode = %q, want %q", cfg.ExecutionMode, "parallel")
		}
	})

	t.Run("tilde path expansion", func(t *testing.T) {
		dir := t.TempDir()
		yaml := `execution_mode: sequential
projects:
  - name: app
    path: ~/myproject
    commands:
      up: ["echo up"]
`
		if err := os.WriteFile(filepath.Join(dir, "test.yml"), []byte(yaml), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadFromDir(dir, "test")
		if err != nil {
			t.Fatalf("LoadFromDir() error: %v", err)
		}

		home, _ := os.UserHomeDir()
		want := filepath.Join(home, "myproject")
		if cfg.Projects[0].Path != want {
			t.Errorf("Projects[0].Path = %q, want %q", cfg.Projects[0].Path, want)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		dir := t.TempDir()
		_, err := LoadFromDir(dir, "nonexistent")
		if err == nil {
			t.Fatal("LoadFromDir() expected error, got nil")
		}
		if !strings.Contains(err.Error(), "config file not found") {
			t.Errorf("error = %q, want containing 'config file not found'", err.Error())
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "bad.yml"), []byte(":::invalid"), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := LoadFromDir(dir, "bad")
		if err == nil {
			t.Fatal("LoadFromDir() expected error, got nil")
		}
		if !strings.Contains(err.Error(), "failed to parse config file") {
			t.Errorf("error = %q, want containing 'failed to parse config file'", err.Error())
		}
	})

	t.Run("validation failure", func(t *testing.T) {
		dir := t.TempDir()
		yaml := `execution_mode: invalid
projects:
  - name: svc
    path: /tmp
`
		if err := os.WriteFile(filepath.Join(dir, "bad.yml"), []byte(yaml), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := LoadFromDir(dir, "bad")
		if err == nil {
			t.Fatal("LoadFromDir() expected error, got nil")
		}
		if !strings.Contains(err.Error(), "invalid config") {
			t.Errorf("error = %q, want containing 'invalid config'", err.Error())
		}
	})

	t.Run("auto-resolves .yaml extension", func(t *testing.T) {
		dir := t.TempDir()
		yaml := `execution_mode: sequential
projects:
  - name: svc
    path: /tmp
    commands:
      up: ["echo up"]
`
		if err := os.WriteFile(filepath.Join(dir, "myconfig.yaml"), []byte(yaml), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadFromDir(dir, "myconfig")
		if err != nil {
			t.Fatalf("LoadFromDir() error: %v", err)
		}
		if cfg.ExecutionMode != "sequential" {
			t.Errorf("ExecutionMode = %q, want %q", cfg.ExecutionMode, "sequential")
		}
		if cfg.Projects[0].Name != "svc" {
			t.Errorf("Projects[0].Name = %q, want %q", cfg.Projects[0].Name, "svc")
		}
	})

	t.Run("explicit .yaml extension", func(t *testing.T) {
		dir := t.TempDir()
		yaml := `execution_mode: parallel
projects:
  - name: app
    path: /tmp
    commands:
      up: ["echo up"]
`
		if err := os.WriteFile(filepath.Join(dir, "project.yaml"), []byte(yaml), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadFromDir(dir, "project.yaml")
		if err != nil {
			t.Fatalf("LoadFromDir() error: %v", err)
		}
		if cfg.ExecutionMode != "parallel" {
			t.Errorf("ExecutionMode = %q, want %q", cfg.ExecutionMode, "parallel")
		}
	})

	t.Run(".yml takes priority over .yaml", func(t *testing.T) {
		dir := t.TempDir()
		ymlContent := `execution_mode: parallel
projects:
  - name: from-yml
    path: /tmp
    commands:
      up: ["echo up"]
`
		yamlContent := `execution_mode: sequential
projects:
  - name: from-yaml
    path: /tmp
    commands:
      up: ["echo up"]
`
		if err := os.WriteFile(filepath.Join(dir, "both.yml"), []byte(ymlContent), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "both.yaml"), []byte(yamlContent), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadFromDir(dir, "both")
		if err != nil {
			t.Fatalf("LoadFromDir() error: %v", err)
		}
		if cfg.Projects[0].Name != "from-yml" {
			t.Errorf("Projects[0].Name = %q, want %q (should prefer .yml)", cfg.Projects[0].Name, "from-yml")
		}
	})

	t.Run("multiple projects", func(t *testing.T) {
		dir := t.TempDir()
		yaml := `execution_mode: parallel
projects:
  - name: api
    path: /tmp
    commands:
      up: ["echo api-up"]
      down: ["echo api-down"]
  - name: web
    path: /tmp
    commands:
      up: ["echo web-up"]
      down: ["echo web-down"]
`
		if err := os.WriteFile(filepath.Join(dir, "multi.yml"), []byte(yaml), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadFromDir(dir, "multi")
		if err != nil {
			t.Fatalf("LoadFromDir() error: %v", err)
		}
		if len(cfg.Projects) != 2 {
			t.Fatalf("len(Projects) = %d, want 2", len(cfg.Projects))
		}
		if cfg.Projects[0].Name != "api" {
			t.Errorf("Projects[0].Name = %q, want %q", cfg.Projects[0].Name, "api")
		}
		if cfg.Projects[1].Name != "web" {
			t.Errorf("Projects[1].Name = %q, want %q", cfg.Projects[1].Name, "web")
		}
	})
}

func TestCommandItemUnmarshalYAML(t *testing.T) {
	t.Run("string format", func(t *testing.T) {
		dir := t.TempDir()
		yaml := `execution_mode: sequential
projects:
  - name: app
    path: /tmp
    commands:
      up:
        - "make up"
        - "make run"
`
		if err := os.WriteFile(filepath.Join(dir, "test.yml"), []byte(yaml), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadFromDir(dir, "test")
		if err != nil {
			t.Fatalf("LoadFromDir() error: %v", err)
		}
		up := cfg.Projects[0].Commands.Up
		if len(up) != 2 {
			t.Fatalf("len(Up) = %d, want 2", len(up))
		}
		if up[0].Command != "make up" || up[0].Background {
			t.Errorf("Up[0] = %+v, want {Command:make up, Background:false}", up[0])
		}
		if up[1].Command != "make run" || up[1].Background {
			t.Errorf("Up[1] = %+v, want {Command:make run, Background:false}", up[1])
		}
	})

	t.Run("struct format with background", func(t *testing.T) {
		dir := t.TempDir()
		yaml := `execution_mode: sequential
projects:
  - name: app
    path: /tmp
    commands:
      up:
        - command: "npm run dev"
          background: true
        - command: "make build"
`
		if err := os.WriteFile(filepath.Join(dir, "test.yml"), []byte(yaml), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadFromDir(dir, "test")
		if err != nil {
			t.Fatalf("LoadFromDir() error: %v", err)
		}
		up := cfg.Projects[0].Commands.Up
		if len(up) != 2 {
			t.Fatalf("len(Up) = %d, want 2", len(up))
		}
		if up[0].Command != "npm run dev" || !up[0].Background {
			t.Errorf("Up[0] = %+v, want {Command:npm run dev, Background:true}", up[0])
		}
		if up[1].Command != "make build" || up[1].Background {
			t.Errorf("Up[1] = %+v, want {Command:make build, Background:false}", up[1])
		}
	})

	t.Run("mixed format", func(t *testing.T) {
		dir := t.TempDir()
		yaml := `execution_mode: sequential
projects:
  - name: app
    path: /tmp
    commands:
      up:
        - "docker-compose up -d"
        - command: "npm run dev"
          background: true
        - "echo done"
`
		if err := os.WriteFile(filepath.Join(dir, "test.yml"), []byte(yaml), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadFromDir(dir, "test")
		if err != nil {
			t.Fatalf("LoadFromDir() error: %v", err)
		}
		up := cfg.Projects[0].Commands.Up
		if len(up) != 3 {
			t.Fatalf("len(Up) = %d, want 3", len(up))
		}
		if up[0].Command != "docker-compose up -d" || up[0].Background {
			t.Errorf("Up[0] = %+v, want {Command:docker-compose up -d, Background:false}", up[0])
		}
		if up[1].Command != "npm run dev" || !up[1].Background {
			t.Errorf("Up[1] = %+v, want {Command:npm run dev, Background:true}", up[1])
		}
		if up[2].Command != "echo done" || up[2].Background {
			t.Errorf("Up[2] = %+v, want {Command:echo done, Background:false}", up[2])
		}
	})
}

func TestCreateConfig(t *testing.T) {
	t.Run("creates template without extension", func(t *testing.T) {
		dir := t.TempDir()
		path, err := CreateConfig(dir, "myproject")
		if err != nil {
			t.Fatalf("CreateConfig() error: %v", err)
		}
		want := filepath.Join(dir, "myproject.yml")
		if path != want {
			t.Errorf("path = %q, want %q", path, want)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile() error: %v", err)
		}
		if string(data) != configTemplate {
			t.Errorf("file content does not match template")
		}
	})

	t.Run("strips .yml extension", func(t *testing.T) {
		dir := t.TempDir()
		path, err := CreateConfig(dir, "myproject.yml")
		if err != nil {
			t.Fatalf("CreateConfig() error: %v", err)
		}
		want := filepath.Join(dir, "myproject.yml")
		if path != want {
			t.Errorf("path = %q, want %q", path, want)
		}
	})

	t.Run("strips .yaml extension and creates .yml", func(t *testing.T) {
		dir := t.TempDir()
		path, err := CreateConfig(dir, "myproject.yaml")
		if err != nil {
			t.Fatalf("CreateConfig() error: %v", err)
		}
		want := filepath.Join(dir, "myproject.yml")
		if path != want {
			t.Errorf("path = %q, want %q", path, want)
		}
	})

	t.Run("error when .yml already exists", func(t *testing.T) {
		dir := t.TempDir()
		existing := filepath.Join(dir, "existing.yml")
		if err := os.WriteFile(existing, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := CreateConfig(dir, "existing")
		if err == nil {
			t.Fatal("CreateConfig() expected error, got nil")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("error = %q, want containing 'already exists'", err.Error())
		}
	})

	t.Run("error when .yaml already exists", func(t *testing.T) {
		dir := t.TempDir()
		existing := filepath.Join(dir, "existing.yaml")
		if err := os.WriteFile(existing, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := CreateConfig(dir, "existing")
		if err == nil {
			t.Fatal("CreateConfig() expected error, got nil")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("error = %q, want containing 'already exists'", err.Error())
		}
	})

	t.Run("creates directory if not exists", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "nested", "dir")
		path, err := CreateConfig(dir, "newconfig")
		if err != nil {
			t.Fatalf("CreateConfig() error: %v", err)
		}
		want := filepath.Join(dir, "newconfig.yml")
		if path != want {
			t.Errorf("path = %q, want %q", path, want)
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("file was not created at %s", path)
		}
	})

	t.Run("template starts with comment header", func(t *testing.T) {
		dir := t.TempDir()
		path, err := CreateConfig(dir, "headercheck")
		if err != nil {
			t.Fatalf("CreateConfig() error: %v", err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile() error: %v", err)
		}
		content := string(data)
		if !strings.HasPrefix(content, "# mdc") {
			t.Errorf("template should start with comment header, got %q", content[:40])
		}
		if !strings.Contains(content, "execution_mode") {
			t.Error("template should contain execution_mode example")
		}
		if !strings.Contains(content, "projects") {
			t.Error("template should contain projects example")
		}
	})
}

func TestRemoveConfig(t *testing.T) {
	t.Run("removes .yml file", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "myproject.yml")
		if err := os.WriteFile(target, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		path, err := RemoveConfig(dir, "myproject")
		if err != nil {
			t.Fatalf("RemoveConfig() error: %v", err)
		}
		if path != target {
			t.Errorf("path = %q, want %q", path, target)
		}
		if _, err := os.Stat(target); !os.IsNotExist(err) {
			t.Errorf("file should have been removed: %s", target)
		}
	})

	t.Run("removes .yaml file", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "myproject.yaml")
		if err := os.WriteFile(target, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		path, err := RemoveConfig(dir, "myproject")
		if err != nil {
			t.Fatalf("RemoveConfig() error: %v", err)
		}
		if path != target {
			t.Errorf("path = %q, want %q", path, target)
		}
		if _, err := os.Stat(target); !os.IsNotExist(err) {
			t.Errorf("file should have been removed: %s", target)
		}
	})

	t.Run("removes with explicit extension", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "myproject.yml")
		if err := os.WriteFile(target, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		path, err := RemoveConfig(dir, "myproject.yml")
		if err != nil {
			t.Fatalf("RemoveConfig() error: %v", err)
		}
		if path != target {
			t.Errorf("path = %q, want %q", path, target)
		}
		if _, err := os.Stat(target); !os.IsNotExist(err) {
			t.Errorf("file should have been removed: %s", target)
		}
	})

	t.Run("error for nonexistent file", func(t *testing.T) {
		dir := t.TempDir()
		_, err := RemoveConfig(dir, "nonexistent")
		if err == nil {
			t.Fatal("RemoveConfig() expected error, got nil")
		}
		if !strings.Contains(err.Error(), "config file not found") {
			t.Errorf("error = %q, want containing 'config file not found'", err.Error())
		}
	})
}

func TestResolveConfigPath(t *testing.T) {
	t.Run("resolves existing .yml file", func(t *testing.T) {
		dir := t.TempDir()
		want := filepath.Join(dir, "myproject.yml")
		if err := os.WriteFile(want, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		got, err := resolveConfigPath(dir, "myproject")
		if err != nil {
			t.Fatalf("resolveConfigPath() error: %v", err)
		}
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("resolves existing .yaml file", func(t *testing.T) {
		dir := t.TempDir()
		want := filepath.Join(dir, "myproject.yaml")
		if err := os.WriteFile(want, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		got, err := resolveConfigPath(dir, "myproject")
		if err != nil {
			t.Fatalf("resolveConfigPath() error: %v", err)
		}
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		dir := t.TempDir()
		_, err := resolveConfigPath(dir, "nonexistent")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "config file not found") {
			t.Errorf("error = %q, want containing 'config file not found'", err.Error())
		}
	})

	t.Run("passes through explicit extension", func(t *testing.T) {
		dir := t.TempDir()
		want := filepath.Join(dir, "myproject.yml")
		if err := os.WriteFile(want, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		got, err := resolveConfigPath(dir, "myproject.yml")
		if err != nil {
			t.Fatalf("resolveConfigPath() error: %v", err)
		}
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}
