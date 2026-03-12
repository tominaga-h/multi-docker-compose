# multi-docker-commander (mdc)

[![build](https://img.shields.io/github/actions/workflow/status/tominaga-h/multi-docker-commander/ci.yml?branch=develop)](https://github.com/tominaga-h/multi-docker-commander/actions/workflows/ci.yml)
[![version](https://img.shields.io/badge/version-2.0.0-blue)](https://github.com/tominaga-h/multi-docker-commander/releases/tag/v2.0.0)

[日本語版のREAMDEはこちら](../README.md)

A CLI tool for **managing and running** the start/stop of Docker environments **across multiple repositories** with a **single command**.

While `docker-compose` has the `-d` option for background execution, mdc can also **daemonize foreground commands** like `npm run dev` and provides the flexibility to **manage processes (stop, restart, view logs)**.

## Features

- Batch operation of Docker Compose across multiple repositories with `mdc up` / `mdc down`
- Selectable `parallel` / `sequential` execution modes between projects
- Background process management and status monitoring (`mdc proc`)
- Project name prefix in log output for better visibility
- Configuration file management (`mdc init` / `mdc edit` / `mdc rm`)
- Simple YAML-based configuration files

[![asciicast](images/demo.gif)](https://asciinema.org/a/803734)

## Installation

### Homebrew

```bash
brew tap tominaga-h/tap
brew install tominaga-h/tap/mdc
```

### Download from GitHub Releases

You can download pre-built binaries from the [latest release](https://github.com/tominaga-h/multi-docker-commander/releases/latest).

```bash
curl -L -o mdc https://github.com/tominaga-h/multi-docker-commander/releases/download/v2.0.0/mdc
chmod +x mdc
sudo mv mdc /usr/local/bin/
```

### Build from Source

```bash
git clone https://github.com/tominaga-h/multi-docker-commander.git
cd multi-docker-commander
make build
```

This generates the `./mdc` binary. Copy it to a directory in your PATH.

### Version-embedded Build

To embed version information from Git tags:

```bash
make build-v
```

## Quick Start

### 1. Create a Configuration File

```bash
mdc init myproject
```

This creates a template at `~/.config/mdc/myproject.yml`. To open it in your editor immediately:

```bash
mdc init myproject --edit
```

### 2. Edit the Configuration File

Edit the generated template to match your project setup:

```yaml
execution_mode: "parallel"
projects:
  - name: "Frontend"
    path: "/path/to/frontend-repo"
    commands:
      up:
        - command: "docker compose up -d"
        - command: "npm run dev"
          background: true
      down:
        - command: "docker compose down"

  - name: "Backend-API"
    path: "/path/to/backend-api-repo"
    commands:
      up:
        - command: "docker compose up -d"
      down:
        - command: "docker compose down"
```

You can also open the file later with `mdc edit myproject`.

### 3. Start and Stop

```bash
mdc up myproject      # Start all projects
mdc down myproject    # Stop all projects
```

The `.yml` extension can be omitted.

### 4. Check Background Processes

```bash
mdc proc list
mdc procs
```

Displays a table of background processes:

```txt
+--------------+------------+-------------+------------------------+-------+---------+
| CONFIG       | PROJECT    | COMMAND     | DIR                    |   PID | STATUS  |
+--------------+------------+-------------+------------------------+-------+---------+
| myproject    | Frontend   | npm run dev | /path/to/frontend-repo | 88888 | Running |
+--------------+------------+-------------+------------------------+-------+---------+
```

### 5. Stop / Restart Background Processes

```bash
mdc proc stop <PID>
mdc proc restart <PID>
```

### 6. View Background Process Logs

```bash
mdc proc attach <PID>
```

## Configuration

Configuration files are placed in `~/.config/mdc/` in YAML format.

### Field Reference

| Field | Required | Description |
|---|---|---|
| `execution_mode` | Yes | `"parallel"` or `"sequential"` |
| `projects` | Yes | List of project definitions (one or more) |
| `projects[].name` | Yes | Project name (used as log output prefix) |
| `projects[].path` | Yes | Project directory path (`~` expansion supported) |
| `projects[].commands.up` | No | List of command objects to run on start |
| `projects[].commands.down` | No | List of command objects to run on stop |
| `commands[][].command` | Yes | Command string to execute |
| `commands[][].background` | No | Set to `true` for background execution (default: `false`) |

### Command Format

Commands are written as objects with `command` and `background` fields:

```yaml
commands:
  up:
    - command: "docker compose up -d"
      background: true
    - command: "echo done"
```

Omitting `background` defaults to foreground execution:

```yaml
commands:
  down:
    - command: "docker compose down"
```

For backward compatibility, plain string format is also supported:

```yaml
commands:
  down:
    - "docker compose down"
```

You can use `mdc proc kill` in `commands.down` to stop all background processes managed by mdc. The runner automatically appends `-c <config-name>`, so you only need to write `mdc proc kill`:

```yaml
commands:
  down:
    - command: "docker compose down"
    - command: "mdc proc kill"
```

### Execution Modes

- **parallel**: All projects run concurrently using Goroutines. Commands within each project are still executed sequentially.
- **sequential**: Projects are processed one at a time in definition order.

## Command Reference

### `mdc up [config-name]`

Loads the specified configuration file and executes each project's `commands.up`.

```bash
mdc up myproject
```

### `mdc down [config-name]`

Loads the specified configuration file and executes each project's `commands.down`. Background processes started by `mdc up` are also automatically stopped.

```bash
mdc down myproject
```

### `mdc list`

Lists configuration files in `~/.config/mdc/`. Also available as `mdc ls`.

```bash
mdc list
mdc ls
```

### `mdc init <config-name>`

Creates a new YAML configuration template in `~/.config/mdc/`. The `.yml` extension can be omitted.

```bash
mdc init myproject           # Creates ~/.config/mdc/myproject.yml
mdc init myproject --edit    # Create and open in $EDITOR
mdc init myproject -e        # Short form
```

| Option | Description |
|---|---|
| `--edit`, `-e` | Open the created file in `$EDITOR` after creation |

### `mdc edit <config-name>`

Opens the specified configuration file in your editor. Uses the `$EDITOR` environment variable, or falls back to `vim` if not set.

```bash
mdc edit myproject
```

### `mdc rm <config-name>`

Removes the specified configuration file from `~/.config/mdc/`. Prompts for confirmation before deletion.

```bash
mdc rm myproject             # Prompts "Are you sure? [y/n]"
mdc rm myproject --force     # Skip confirmation
mdc rm myproject -f          # Short form
```

| Option | Description |
|---|---|
| `--force`, `-f` | Skip the confirmation prompt |

### `mdc proc` (alias: `mdc procs`)

Manages background processes. When called without a subcommand, it behaves as `proc list`.

#### `mdc proc list [config-name]`

Lists background processes managed by mdc. When config name is omitted, shows processes for all configurations.

```bash
mdc proc list              # Show all processes
mdc proc list myproject    # Show processes for a specific config
mdc procs                  # Alias (equivalent to proc list)
```

#### `mdc proc attach <PID>`

Streams log output from a background process. Press Ctrl-C to detach (the process continues running).

```bash
mdc proc attach 12345
mdc proc attach 12345 --tail 50       # Start from the last 50 lines
mdc proc attach 12345 --no-follow     # Print existing logs and exit
```

#### `mdc proc stop <PID>`

Stops the background process with the specified PID.

```bash
mdc proc stop 12345
```

#### `mdc proc restart <PID>`

Restarts the background process with the specified PID.

```bash
mdc proc restart 12345
```

#### `mdc proc kill`

Kills background processes by config name, PID, or all configs. Use `-c` to kill all processes belonging to a config, `-p` to kill a single process by PID, or `--all` to kill all tracked processes.

When `mdc proc kill` is used in YAML `commands.down`, the runner automatically appends `-c <config-name>`.

```bash
mdc proc kill -c myproject    # Kill all processes for a config
mdc proc kill -p 12345        # Kill a single process by PID
mdc proc kill --all           # Kill all tracked processes across all configs
```

| Option | Description |
|---|---|
| `-c`, `--config` | Config name to kill all processes for |
| `-p`, `--pid` | PID of the process to kill |
| `--all` | Kill all tracked processes across all configs |

### `mdc --version`

Displays version information.

```bash
mdc --version
mdc -v
```

## Development

### Requirements

- Go 1.25+

### Build

```bash
make build
```

### Test

```bash
make test             # Internal package tests
make test-integration # Integration tests
make test-all         # All tests
make test-cover       # Tests with coverage
make lint             # go vet + golangci-lint
make check            # lint + test-all
```

## License

TBD
