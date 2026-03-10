# multi-docker-commander (mdc)

[![build](https://img.shields.io/github/actions/workflow/status/tominaga-h/multi-docker-commander/ci.yml?branch=develop)](https://github.com/tominaga-h/multi-docker-commander/actions/workflows/ci.yml)
[![version](https://img.shields.io/badge/version-2.0.0-blue)](https://github.com/tominaga-h/multi-docker-commander/releases/tag/v2.0.0)

**複数リポジトリにまたがる** Docker 環境の起動・停止を、**1つのコマンドで一括管理・実行** するための CLI ツール。

`docker-compose` はバックグラウンド実行できる `-d` オプションが存在するが、`npm run dev` のようなフォアグラウンドでサーバーを起動するコマンドでも **mdc上でバックグラウンド（デーモン）化できる** し、**プロセスの管理（終了・再起動・ログ出力）もできる** 柔軟さを備えている。

## 特徴

- 複数リポジトリの Docker Compose を `mdc up` / `mdc down` で一括操作
- プロジェクト間の並列 (`parallel`) / 直列 (`sequential`) 実行モードを選択可能
- バックグラウンドプロセスの管理と状態確認 (`mdc proc`)
- プロジェクト名プレフィックス付きのログ出力で視認性を確保
- 設定ファイルの管理コマンド (`mdc init` / `mdc edit` / `mdc rm`)
- YAML ベースのシンプルな設定ファイル

[![asciicast](../images/demo.gif)](https://asciinema.org/a/803734)

## インストール

### Homebrew

```bash
brew tap tominaga-h/tap
brew install tominaga-h/tap/mdc
```

### GitHub Releases からダウンロード

[最新リリース](https://github.com/tominaga-h/multi-docker-commander/releases/latest)からビルド済みバイナリをダウンロードできます。

```bash
curl -L -o mdc https://github.com/tominaga-h/multi-docker-commander/releases/download/v2.0.0/mdc
chmod +x mdc
sudo mv mdc /usr/local/bin/
```

### ソースからビルド

```bash
git clone https://github.com/tominaga-h/multi-docker-commander.git
cd multi-docker-commander
make build
```

`./mdc` バイナリが生成されます。パスの通った場所にコピーしてください。

### バージョン埋め込みビルド

Git タグからバージョン情報を埋め込む場合:

```bash
make build-v
```

## クイックスタート

### 1. 設定ファイルの作成

```bash
mdc init myproject
```

`~/.config/mdc/myproject.yml` にテンプレートが生成されます。作成後すぐにエディタで開くこともできます:

```bash
mdc init myproject --edit
```

### 2. 設定ファイルの編集

生成されたテンプレートをプロジェクト構成に合わせて編集します:

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

`mdc edit myproject` で後からエディタで開くこともできます。

### 3. 起動と停止

```bash
mdc up myproject      # 全プロジェクトを起動
mdc down myproject    # 全プロジェクトを停止
```

拡張子 `.yml` は省略可能です。

### 4. バックグラウンドプロセスの確認

```bash
mdc proc list
mdc procs
```

以下のようにテーブル形式でバックグラウンドプロセスの一覧を表示します。

```txt
+--------------+------------+-------------+------------------------+-------+---------+
| CONFIG       | PROJECT    | COMMAND     | DIR                    |   PID | STATUS  |
+--------------+------------+-------------+------------------------+-------+---------+
| myproject    | Frontend   | npm run dev | /path/to/frontend-repo | 88888 | Running |
+--------------+------------+-------------+------------------------+-------+---------+
```

### 5. バックグラウンドプロセスの終了・再起動

```bash
mdc proc stop <PID>
mdc proc restart <PID>
```

### 6. バックグラウンドプロセスのログ出力を確認

```bash
mdc proc attach <PID>
```

## 設定ファイル

設定ファイルは `~/.config/mdc/` に YAML 形式で配置します。

### フィールド一覧

| フィールド | 必須 | 説明 |
|---|---|---|
| `execution_mode` | Yes | `"parallel"` (並列) または `"sequential"` (直列) |
| `projects` | Yes | プロジェクト定義のリスト (1つ以上) |
| `projects[].name` | Yes | プロジェクト名 (ログ出力のプレフィックスに使用) |
| `projects[].path` | Yes | プロジェクトのディレクトリパス (`~` 展開対応) |
| `projects[].commands.up` | No | 起動時に実行するコマンドオブジェクトのリスト |
| `projects[].commands.down` | No | 停止時に実行するコマンドオブジェクトのリスト |
| `commands[][].command` | Yes | 実行するコマンド文字列 |
| `commands[][].background` | No | `true` でバックグラウンド実行 (デフォルト: `false`) |

### コマンドの記述形式

コマンドは `command` と `background` フィールドを持つオブジェクト形式で記述します:

```yaml
commands:
  up:
    - command: "docker compose up -d"
      background: true
    - command: "echo done"
```

`background` を省略するとフォアグラウンド実行（デフォルト）になります:

```yaml
commands:
  down:
    - command: "docker compose down"
```

後方互換として、文字列での記述も引き続きサポートされています:

```yaml
commands:
  down:
    - "docker compose down"
```

`commands.down` に `mdc proc kill` を記述すると、mdc が管理するバックグラウンドプロセスを一括停止できます。runner が自動的に `-c <設定名>` を付与するため、`mdc proc kill` とだけ記述すれば動作します:

```yaml
commands:
  down:
    - command: "docker compose down"
    - command: "mdc proc kill"
```

### 実行モード

- **parallel**: 全プロジェクトを Goroutine で同時に実行します。各プロジェクト内のコマンドは直列で実行されます。
- **sequential**: プロジェクトを定義順に1つずつ処理します。

## コマンドリファレンス

### `mdc up [config-name]`

指定した設定ファイルを読み込み、各プロジェクトの `commands.up` を実行します。

```bash
mdc up myproject
```

### `mdc down [config-name]`

指定した設定ファイルを読み込み、各プロジェクトの `commands.down` を実行します。`mdc up` で起動したバックグラウンドプロセスも自動的に停止します。

```bash
mdc down myproject
```

### `mdc list`

`~/.config/mdc/` 内の設定ファイル一覧を表示します。エイリアスとして `mdc ls` も使用できます。

```bash
mdc list
mdc ls
```

### `mdc init <config-name>`

`~/.config/mdc/` に YAML 設定ファイルのテンプレートを新規作成します。拡張子 `.yml` は省略可能です。

```bash
mdc init myproject           # ~/.config/mdc/myproject.yml を作成
mdc init myproject --edit    # 作成後に $EDITOR で開く
mdc init myproject -e        # 短縮形
```

| オプション | 説明 |
|---|---|
| `--edit`, `-e` | 作成後に `$EDITOR` でファイルを開く |

### `mdc edit <config-name>`

指定した設定ファイルをエディタで開きます。環境変数 `$EDITOR` を優先し、未設定の場合は `vim` を使用します。

```bash
mdc edit myproject
```

### `mdc rm <config-name>`

指定した設定ファイルを `~/.config/mdc/` から削除します。削除前に確認プロンプトが表示されます。

```bash
mdc rm myproject             # 確認プロンプト "[y/n]" が表示される
mdc rm myproject --force     # 確認をスキップ
mdc rm myproject -f          # 短縮形
```

| オプション | 説明 |
|---|---|
| `--force`, `-f` | 確認プロンプトをスキップする |

### `mdc proc` (エイリアス: `mdc procs`)

バックグラウンドプロセスを管理します。サブコマンドを省略すると `proc list` として動作します。

#### `mdc proc list [config-name]`

mdc が管理しているバックグラウンドプロセスの一覧を表示します。設定名を省略すると全設定のプロセスを表示します。

```bash
mdc proc list              # 全設定のプロセスを表示
mdc proc list myproject    # 特定の設定のプロセスのみ表示
mdc procs                  # エイリアス (proc list と同等)
```

#### `mdc proc attach <PID>`

バックグラウンドプロセスのログ出力をストリームします。Ctrl-C でデタッチできます（プロセスは継続）。

```bash
mdc proc attach 12345
mdc proc attach 12345 --tail 50       # 末尾50行から表示
mdc proc attach 12345 --no-follow     # 既存ログを出力して終了
```

#### `mdc proc stop <PID>`

指定した PID のバックグラウンドプロセスを停止します。

```bash
mdc proc stop 12345
```

#### `mdc proc restart <PID>`

指定した PID のバックグラウンドプロセスを再起動します。

```bash
mdc proc restart 12345
```

#### `mdc proc kill`

設定名または PID を指定してバックグラウンドプロセスを終了します。`-c` で指定した設定に属する全プロセスを一括終了、`-p` で単一プロセスを終了できます。

YAML の `commands.down` に `mdc proc kill` と記述すると、runner が自動的に `-c <設定名>` を付与して実行します。

```bash
mdc proc kill -c myproject    # 指定した設定の全プロセスを終了
mdc proc kill -p 12345        # 指定した PID のプロセスを終了
```

| オプション | 説明 |
|---|---|
| `-c`, `--config` | 全プロセスを終了する設定名 |
| `-p`, `--pid` | 終了するプロセスの PID |

### `mdc --version`

バージョン情報を表示します。

```bash
mdc --version
mdc -v
```

## 開発

### 必要な環境

- Go 1.25+

### ビルド

```bash
make build
```

### テスト

```bash
make test             # internal パッケージのテスト
make test-integration # 統合テスト
make test-all         # 全テスト
make test-cover       # カバレッジ付きテスト
make lint             # go vet + golangci-lint
make check            # lint + test-all
```

## ライセンス

TBD
