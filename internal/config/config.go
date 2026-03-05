package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type CommandItem struct {
	Command    string `yaml:"command"`
	Background bool   `yaml:"background"`
}

func (c *CommandItem) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		c.Command = value.Value
		return nil
	}
	type raw CommandItem
	var r raw
	if err := value.Decode(&r); err != nil {
		return err
	}
	*c = CommandItem(r)
	return nil
}

type Commands struct {
	Up   []CommandItem `yaml:"up"`
	Down []CommandItem `yaml:"down"`
}

type Project struct {
	Name     string   `yaml:"name"`
	Path     string   `yaml:"path"`
	Commands Commands `yaml:"commands"`
}

type Config struct {
	ExecutionMode string    `yaml:"execution_mode"`
	Projects      []Project `yaml:"projects"`
}

func ExpandHome(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, path[1:]), nil
}

func ContractHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == home {
		return "~"
	}
	prefix := home + string(os.PathSeparator)
	if strings.HasPrefix(path, prefix) {
		return "~" + string(os.PathSeparator) + path[len(prefix):]
	}
	return path
}

func BaseMDCDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".config", "mdc"), nil
}

func DefaultConfigDir() (string, error) {
	return BaseMDCDir()
}

func ListConfigs() ([]string, error) {
	configDir, err := DefaultConfigDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read config directory %s: %w", configDir, err)
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext == ".yml" || ext == ".yaml" {
			files = append(files, e.Name())
		}
	}
	return files, nil
}

func Load(name string) (*Config, error) {
	configDir, err := DefaultConfigDir()
	if err != nil {
		return nil, err
	}
	return LoadFromDir(configDir, name)
}

func resolveConfigPath(configDir, name string) (string, error) {
	path := filepath.Join(configDir, name)
	if filepath.Ext(path) != "" {
		return path, nil
	}
	for _, ext := range []string{".yml", ".yaml"} {
		candidate := path + ext
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("config file not found: tried %s.yml and %s.yaml", path, path)
}

func LoadFromDir(configDir, name string) (*Config, error) {
	path, err := resolveConfigPath(configDir, name)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %q: %w", path, err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config %q: %w", name, err)
	}

	for i := range cfg.Projects {
		expanded, err := ExpandHome(cfg.Projects[i].Path)
		if err != nil {
			return nil, fmt.Errorf("project %q: %w", cfg.Projects[i].Name, err)
		}
		cfg.Projects[i].Path = expanded
	}

	return &cfg, nil
}

const configTemplate = `# mdc 設定ファイル
# コメントを外して、プロジェクトの情報を記入してください。
#
# execution_mode: プロジェクト間の実行モード
#   "parallel"    - 全プロジェクトを同時に実行
#   "sequential"  - プロジェクトを定義順に1つずつ処理
#
# projects[].name: プロジェクト名 (ログ出力のプレフィックスに使用)
# projects[].path: プロジェクトのディレクトリパス (~展開対応)
# projects[].commands.up: 起動時に実行するコマンドのリスト
# projects[].commands.down: 停止時に実行するコマンドのリスト
# commands[][].command: 実行するコマンド文字列
# commands[][].background: true でバックグラウンド実行 (デフォルト: false)

# execution_mode: "parallel"
# projects:
#   - name: "Frontend"
#     path: "/path/to/frontend-repo"
#     commands:
#       up:
#         - command: "docker compose up -d"
#         - command: "npm run dev"
#           background: true
#       down:
#         - command: "docker compose down"
#
#   - name: "Backend-API"
#     path: "/path/to/backend-api-repo"
#     commands:
#       up:
#         - command: "docker compose up -d"
#       down:
#         - command: "docker compose down"
`

func normalizeConfigName(name string) string {
	ext := filepath.Ext(name)
	if ext == ".yml" || ext == ".yaml" {
		return strings.TrimSuffix(name, ext)
	}
	return name
}

func CreateConfig(configDir, name string) (string, error) {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}

	baseName := normalizeConfigName(name)
	targetPath := filepath.Join(configDir, baseName+".yml")

	for _, ext := range []string{".yml", ".yaml"} {
		candidate := filepath.Join(configDir, baseName+ext)
		if _, err := os.Stat(candidate); err == nil {
			return "", fmt.Errorf("config file already exists: %s", candidate)
		}
	}

	if err := os.WriteFile(targetPath, []byte(configTemplate), 0644); err != nil {
		return "", fmt.Errorf("failed to write config file %s: %w", targetPath, err)
	}

	return targetPath, nil
}

func ResolveConfigPath(name string) (string, error) {
	configDir, err := DefaultConfigDir()
	if err != nil {
		return "", err
	}
	return resolveConfigPath(configDir, name)
}

func CreateDefaultConfig(name string) (string, error) {
	configDir, err := DefaultConfigDir()
	if err != nil {
		return "", err
	}
	return CreateConfig(configDir, name)
}

func RemoveConfig(configDir, name string) (string, error) {
	path, err := resolveConfigPath(configDir, name)
	if err != nil {
		return "", err
	}
	if err := os.Remove(path); err != nil {
		return "", fmt.Errorf("failed to remove config file %s: %w", path, err)
	}
	return path, nil
}

func RemoveDefaultConfig(name string) (string, error) {
	configDir, err := DefaultConfigDir()
	if err != nil {
		return "", err
	}
	return RemoveConfig(configDir, name)
}

func (c *Config) validate() error {
	switch c.ExecutionMode {
	case "parallel", "sequential":
	default:
		return fmt.Errorf("execution_mode must be \"parallel\" or \"sequential\", got %q", c.ExecutionMode)
	}

	if len(c.Projects) == 0 {
		return fmt.Errorf("at least one project must be defined")
	}

	for i, p := range c.Projects {
		if p.Name == "" {
			return fmt.Errorf("project[%d]: name is required", i)
		}
		if p.Path == "" {
			return fmt.Errorf("project %q: path is required", p.Name)
		}
	}

	return nil
}
