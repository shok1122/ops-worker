# ops-worker 設計書

**リポジトリ名:** `ops-worker`  
**言語:** Go  
**対象OS:** Ubuntu (systemdサービスとして動作)

---

## 1. 概要

`ops-worker` は、マシンの各種状態をチェックし、その結果をJSON形式でリモートサーバに送信するエージェントプログラムです。cronスケジュール形式で各チェックの実行間隔を指定でき、Ubuntu上でsystemdサービスとして常駐します。

---

## 2. アーキテクチャ概要

```
ops-worker
├── main.go                  # エントリポイント、サービス起動
├── config/
│   └── config.go            # 設定ファイル読み込み
├── scheduler/
│   └── scheduler.go         # cronスケジューラ管理
├── checker/
│   ├── checker.go           # チェッカーインターフェース定義
│   ├── cpu.go               # CPU使用率チェック
│   ├── disk.go              # ディスク使用率チェック
│   ├── memory.go            # メモリ使用率チェック
│   ├── process.go           # プロセス起動状態チェック
│   ├── docker.go            # Dockerコンテナ起動状態チェック
│   └── external.go          # 外部プログラム呼び出しチェック
├── sender/
│   └── sender.go            # HTTP送信
├── healthcheck/
│   └── healthcheck.go       # ヘルスチェック送信
└── version/
    └── version.go           # バージョン情報
```

---

## 3. 設定ファイル仕様

設定ファイルは **2ファイル構成** とします。

| ファイル | デフォルトパス | 役割 |
|---|---|---|
| メイン設定 | `/etc/ops-worker/config.yaml` | サーバ接続・ヘルスチェックなど一般設定 |
| チェック定義 | `/etc/ops-worker/checks.yaml` | 状態チェックの定義一覧 |

起動オプション `-config <path>` でメイン設定ファイルのパスを変更可能。  
チェック定義ファイルのパスはメイン設定の `checks_file` キーで変更可能。

---

### 3.1 メイン設定ファイル (`config.yaml`)

```yaml
# 送信先サーバ設定
server:
  host: "example.com"       # FQDNまたはIPアドレス
  port: 8443                 # ポート番号
  tls: true                  # true=https / false=http
  password: "secret-password"
  timeout: 10                # 送信タイムアウト秒数 (デフォルト: 10)

# ヘルスチェック設定
healthcheck:
  schedule: "*/5 * * * *"   # cronスケジュール (5分ごと)

# チェック定義ファイルのパス
checks_file: "/etc/ops-worker/checks.yaml"
```

#### メイン設定項目詳細

| セクション | キー | 型 | 説明 | デフォルト |
|---|---|---|---|---|
| server | host | string | 送信先FQDNまたはIPアドレス | (必須) |
| server | port | int | 送信先ポート番号 | (必須) |
| server | tls | bool | TLS(HTTPS)使用フラグ | `true` |
| server | password | string | 認証パスワード | (必須) |
| server | timeout | int | タイムアウト秒数 | `10` |
| healthcheck | schedule | string | cronスケジュール | (必須) |
| checks_file | - | string | チェック定義ファイルパス | `/etc/ops-worker/checks.yaml` |

**エンドポイントURL自動生成ルール:**

| 種別 | 生成されるURL |
|---|---|
| 状態チェック結果 | `{scheme}://{host}:{port}/api/v1/report` |
| ヘルスチェック | `{scheme}://{host}:{port}/api/v1/health` |

`scheme` は `tls: true` のとき `https`、`false` のとき `http`。

---

### 3.2 チェック定義ファイル (`checks.yaml`)

```yaml
checks:
  - name: "cpu"
    type: "cpu"
    schedule: "*/1 * * * *"

  - name: "disk_root"
    type: "disk"
    schedule: "*/5 * * * *"
    options:
      path: "/"

  - name: "memory"
    type: "memory"
    schedule: "*/1 * * * *"

  - name: "nginx_process"
    type: "process"
    schedule: "*/1 * * * *"
    options:
      process_name: "nginx"

  - name: "web_container"
    type: "docker"
    schedule: "*/1 * * * *"
    options:
      container_name: "my-web-app"

  - name: "custom_check"
    type: "external"
    schedule: "*/2 * * * *"
    options:
      command: "/usr/local/bin/my-check.sh"
      args: ["--verbose"]
      timeout: 5  # 秒
```

#### チェック定義項目詳細

| キー | 型 | 説明 |
|---|---|---|
| checks[].name | string | チェック識別名 (ファイル内で一意) |
| checks[].type | string | チェック種別: `cpu` / `disk` / `memory` / `process` / `docker` / `external` |
| checks[].schedule | string | cronスケジュール (5フィールド形式) |
| checks[].options | map | チェック種別ごとのオプション (省略可) |

---

## 4. チェッカーインターフェースと統一出力フォーマット

### 4.1 統一出力フォーマット

すべての状態チェック機能は、以下の **統一フォーマット** で結果を返します。チェック種別による差異は `metrics` フィールド内にのみ収めます。

```go
// checker/checker.go

type CheckResult struct {
    Name      string            `json:"name"`       // 設定ファイルで定義したチェック識別名
    Type      string            `json:"type"`       // チェック種別 (cpu/disk/memory/process/docker/external)
    Timestamp time.Time         `json:"timestamp"`  // チェック実行時刻 (RFC3339)
    Status    string            `json:"status"`     // "ok" | "error"
    Message   string            `json:"message"`    // 状態の概要説明 (ok時は空文字可)
    Metrics   []Metric          `json:"metrics"`    // 数値メトリクス (0件以上)
    Labels    map[string]string `json:"labels"`     // チェック種別固有の識別情報
    Error     string            `json:"error,omitempty"` // エラー詳細 (status="error"時のみ)
}

type Metric struct {
    Name  string  `json:"name"`  // メトリクス名 (例: "usage_percent")
    Value float64 `json:"value"` // メトリクス値
    Unit  string  `json:"unit"`  // 単位 (例: "percent", "bytes", "count")
}

type Checker interface {
    Name() string
    Type() string
    Check(ctx context.Context) CheckResult
}
```

#### フィールド定義

| フィールド | 必須 | 説明 |
|---|---|---|
| `name` | ○ | 設定ファイルの `checks[].name` の値 |
| `type` | ○ | チェック種別識別子 |
| `timestamp` | ○ | ISO 8601 / RFC3339形式 (例: `2025-05-21T10:00:00Z`) |
| `status` | ○ | `"ok"` または `"error"` の固定値 |
| `message` | ○ | 人間可読な状態サマリ。ok時は `""` 可 |
| `metrics` | ○ | 数値指標の配列。該当なし時は `[]` |
| `labels` | ○ | チェック対象を識別するキー・バリュー。該当なし時は `{}` |
| `error` | △ | `status="error"` の場合のみ出力 |

#### `Metric.unit` 値一覧

| 値 | 意味 |
|---|---|
| `"percent"` | パーセント (0.0〜100.0) |
| `"bytes"` | バイト数 |
| `"count"` | 個数 |
| `"seconds"` | 秒数 |
| `"bool"` | 真偽値 (1.0=true, 0.0=false) |

---

### 4.2 Checkerインターフェース

```go
type Checker interface {
    Name() string
    Type() string
    Check(ctx context.Context) CheckResult
}
```

---

## 5. 各チェッカー仕様

### 5.1 CPU使用率 (`cpu.go`)

**使用ライブラリ:** `github.com/shirou/gopsutil/v3/cpu`

**出力例:**

```json
{
  "name": "cpu",
  "type": "cpu",
  "timestamp": "2025-05-21T10:00:00Z",
  "status": "ok",
  "message": "",
  "metrics": [
    { "name": "usage_percent", "value": 23.5, "unit": "percent" },
    { "name": "core_count",    "value": 8,    "unit": "count"   }
  ],
  "labels": {}
}
```

---

### 5.2 ディスク使用率 (`disk.go`)

**使用ライブラリ:** `github.com/shirou/gopsutil/v3/disk`

**オプション:**

| キー | 型 | 説明 | デフォルト |
|---|---|---|---|
| path | string | チェック対象パス | `/` |

**出力例:**

```json
{
  "name": "disk_root",
  "type": "disk",
  "timestamp": "2025-05-21T10:00:00Z",
  "status": "ok",
  "message": "",
  "metrics": [
    { "name": "total_bytes",    "value": 107374182400, "unit": "bytes"   },
    { "name": "used_bytes",     "value": 53687091200,  "unit": "bytes"   },
    { "name": "free_bytes",     "value": 53687091200,  "unit": "bytes"   },
    { "name": "usage_percent",  "value": 50.0,         "unit": "percent" }
  ],
  "labels": {
    "path": "/"
  }
}
```

---

### 5.3 メモリ使用率 (`memory.go`)

**使用ライブラリ:** `github.com/shirou/gopsutil/v3/mem`

**出力例:**

```json
{
  "name": "memory",
  "type": "memory",
  "timestamp": "2025-05-21T10:00:00Z",
  "status": "ok",
  "message": "",
  "metrics": [
    { "name": "total_bytes",     "value": 17179869184, "unit": "bytes"   },
    { "name": "used_bytes",      "value": 8589934592,  "unit": "bytes"   },
    { "name": "available_bytes", "value": 8589934592,  "unit": "bytes"   },
    { "name": "usage_percent",   "value": 50.0,        "unit": "percent" }
  ],
  "labels": {}
}
```

---

### 5.4 プロセス起動状態 (`process.go`)

**使用ライブラリ:** `github.com/shirou/gopsutil/v3/process`

**オプション:**

| キー | 型 | 説明 |
|---|---|---|
| process_name | string | チェック対象プロセス名 |

**出力例:**

```json
{
  "name": "nginx_process",
  "type": "process",
  "timestamp": "2025-05-21T10:00:00Z",
  "status": "ok",
  "message": "",
  "metrics": [
    { "name": "running",   "value": 1, "unit": "bool"  },
    { "name": "pid_count", "value": 2, "unit": "count" }
  ],
  "labels": {
    "process_name": "nginx"
  }
}
```

> `running` メトリクスは `1.0` = 起動中、`0.0` = 停止中。プロセスが停止している場合は `status: "error"` とします。

---

### 5.5 Dockerコンテナ起動状態 (`docker.go`)

**使用ライブラリ:** `github.com/docker/docker/client`

**オプション:**

| キー | 型 | 説明 |
|---|---|---|
| container_name | string | チェック対象コンテナ名 |

**出力例:**

```json
{
  "name": "web_container",
  "type": "docker",
  "timestamp": "2025-05-21T10:00:00Z",
  "status": "ok",
  "message": "",
  "metrics": [
    { "name": "running", "value": 1, "unit": "bool" }
  ],
  "labels": {
    "container_name": "my-web-app",
    "container_id":   "abc123def456",
    "container_status": "running"
  }
}
```

> コンテナが停止または存在しない場合は `status: "error"` とします。

---

### 5.6 外部プログラム実行 (`external.go`)

利用者が作成した任意のプログラムを実行し、その標準出力を `metrics` および `labels` にマージします。

**オプション:**

| キー | 型 | 説明 | デフォルト |
|---|---|---|---|
| command | string | 実行コマンド (絶対パス推奨) | (必須) |
| args | []string | コマンド引数 | [] |
| timeout | int | タイムアウト秒数 | 5 |

**外部プログラム規約:**

外部プログラムは標準出力に以下の統一フォーマットのJSONを出力すること。

```json
{
  "status": "ok",
  "message": "任意のメッセージ",
  "metrics": [
    { "name": "response_time_ms", "value": 42.5, "unit": "seconds" }
  ],
  "labels": {
    "endpoint": "https://example.com"
  }
}
```

- 正常終了時は終了コード `0` を返すこと
- 異常時は終了コード `0` 以外を返すこと
- `metrics` / `labels` は省略可能 (省略時は `[]` / `{}` として扱う)

**ops-workerが生成する最終出力例:**

```json
{
  "name": "custom_check",
  "type": "external",
  "timestamp": "2025-05-21T10:00:00Z",
  "status": "ok",
  "message": "任意のメッセージ",
  "metrics": [
    { "name": "response_time_ms", "value": 42.5, "unit": "seconds" }
  ],
  "labels": {
    "command":  "/usr/local/bin/my-check.sh",
    "endpoint": "https://example.com"
  }
}
```

> `labels` には外部プログラム出力の `labels` に加えて、`command` キーを自動付与します。

---

## 6. 送信仕様

### 6.1 状態チェック結果の送信

**HTTPメソッド:** POST  
**Content-Type:** `application/json`  
**認証:** `Authorization: Bearer <password>` ヘッダ

**リクエストボディ:**

```json
{
  "hostname": "my-server-01",
  "sent_at": "2025-05-21T10:00:00Z",
  "result": {
    "name": "cpu",
    "type": "cpu",
    "timestamp": "2025-05-21T10:00:00Z",
    "status": "ok",
    "message": "",
    "metrics": [
      { "name": "usage_percent", "value": 23.5, "unit": "percent" },
      { "name": "core_count",    "value": 8,    "unit": "count"   }
    ],
    "labels": {}
  }
}
```

各チェックはスケジュール実行のたびに1件ずつ即座に送信します。

---

### 6.2 ヘルスチェック送信

**HTTPメソッド:** POST  
**Content-Type:** `application/json`  
**認証:** `Authorization: Bearer <password>` ヘッダ

**リクエストボディ:**

```json
{
  "type": "healthcheck",
  "hostname": "my-server-01",
  "sent_at": "2025-05-21T10:00:00Z",
  "agent": {
    "version": "1.0.0",
    "uptime_seconds": 3600,
    "started_at": "2025-05-21T09:00:00Z",
    "go_version": "go1.22.0",
    "os": "linux",
    "arch": "amd64"
  }
}
```

---

## 7. スケジューラ仕様

**使用ライブラリ:** `github.com/robfig/cron/v3`

- 設定ファイルの各チェックと、ヘルスチェックそれぞれに対してcronジョブを登録します。
- cronフォーマット: 標準5フィールド形式 `分 時 日 月 曜日`
- サービス起動時にすべてのジョブが登録され、シグナル (`SIGTERM`, `SIGINT`) 受信時にグレースフルシャットダウンします。

---

## 8. バージョン情報

**使用方法:** ビルド時にLDフラグで埋め込み

```go
// version/version.go
var (
    Version   = "dev"
    Commit    = "unknown"
    BuildDate = "unknown"
)
```

**ビルドコマンド例:**

```bash
go build -ldflags \
  "-X ops-worker/version.Version=1.0.0 \
   -X ops-worker/version.Commit=$(git rev-parse --short HEAD) \
   -X ops-worker/version.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o ops-worker .
```

---

## 9. GitHub Actions（CI/CD）

### 9.1 ビルドワークフロー (`.github/workflows/build.yml`)

**トリガー:** `main` ブランチへのpush、またはタグ (`v*`) のpush

**処理内容:**
1. `go build` で静的リンクの単一バイナリをコンパイル
2. アーキテクチャ: `linux/amd64`（将来的に `linux/arm64` 追加可能）
3. バイナリをGitHub Releasesにアップロード

**静的リンクビルドコマンド:**

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build \
  -ldflags "-s -w \
    -X ops-worker/version.Version=${GITHUB_REF_NAME} \
    -X ops-worker/version.Commit=${GITHUB_SHA::8} \
    -X ops-worker/version.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o ops-worker .
```

### 9.2 ワークフローファイル構成

```yaml
# .github/workflows/build.yml
name: Build and Release

on:
  push:
    tags:
      - 'v*'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: Build static binary
        run: |
          CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
          go build -ldflags "-s -w \
            -X ops-worker/version.Version=${{ github.ref_name }} \
            -X ops-worker/version.Commit=${{ github.sha }} \
            -X ops-worker/version.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
          -o ops-worker .
      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          files: ops-worker
```

---

## 10. インストール・サービス登録

### 10.1 GitHubから直接インストールするスクリプト

リポジトリに `install.sh` を同梱し、以下のコマンドでインストール可能にします。

```bash
curl -fsSL https://raw.githubusercontent.com/<owner>/ops-worker/main/install.sh | sudo bash
```

**`install.sh` の処理内容:**

1. 最新リリースのバイナリURLを GitHub Releases API から取得
2. `/usr/local/bin/ops-worker` にダウンロード・配置
3. 実行権限を付与
4. サンプル設定ファイルを `/etc/ops-worker/config.yaml` に配置（未存在時のみ）
5. サンプルチェック定義ファイルを `/etc/ops-worker/checks.yaml` に配置（未存在時のみ）
6. systemdユニットファイルを `/etc/systemd/system/ops-worker.service` に配置
7. `systemctl daemon-reload` → `systemctl enable ops-worker` → `systemctl start ops-worker`

### 10.2 systemdユニットファイル

```ini
# ops-worker.service
[Unit]
Description=ops-worker machine state monitor
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/ops-worker -config /etc/ops-worker/config.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

---

## 11. 依存ライブラリ一覧

| ライブラリ | 用途 |
|---|---|
| `github.com/shirou/gopsutil/v3` | CPU/メモリ/ディスク/プロセス情報取得 |
| `github.com/docker/docker/client` | Docker APIアクセス |
| `github.com/robfig/cron/v3` | cronスケジューラ |
| `gopkg.in/yaml.v3` | YAML設定ファイルのパース |

標準ライブラリ (`net/http`, `os/exec`, `encoding/json` 等) を最大限活用し、外部依存を最小限にします。

---

## 12. ディレクトリ・ファイル構成（最終）

```
ops-worker/
├── .github/
│   └── workflows/
│       └── build.yml
├── checker/
│   ├── checker.go       # Checkerインターフェース、CheckResult構造体
│   ├── cpu.go
│   ├── disk.go
│   ├── memory.go
│   ├── process.go
│   ├── docker.go
│   └── external.go
├── config/
│   ├── config.go        # Config構造体、メイン設定YAMLパース
│   └── checks.go        # ChecksConfig構造体、チェック定義YAMLパース
├── scheduler/
│   └── scheduler.go     # cronジョブ登録・管理
├── sender/
│   └── sender.go        # HTTP POST送信
├── healthcheck/
│   └── healthcheck.go   # ヘルスチェック構造体・送信
├── version/
│   └── version.go       # バージョン変数
├── install.sh            # 自動インストールスクリプト
├── ops-worker.service    # systemdユニットファイル
├── config.example.yaml   # メイン設定ファイルサンプル
├── checks.example.yaml   # チェック定義ファイルサンプル
├── go.mod
├── go.sum
├── main.go
└── README.md
```

---

## 13. 起動オプション

| フラグ | 説明 | デフォルト |
|---|---|---|
| `-config` | メイン設定ファイルパス | `/etc/ops-worker/config.yaml` |
| `-checks` | チェック定義ファイルパス (指定時はconfig.yamlの`checks_file`より優先) | - |
| `-version` | バージョン情報を表示して終了 | - |

---

## 14. エラーハンドリング方針

- 各チェッカーは実行時エラーが発生しても `CheckResult.Status = "error"` として結果を返し、プロセスを停止しない。
- 送信失敗時はログに記録し、次のスケジュールで再試行（バッファリングなし）。
- 設定ファイルの読み込みエラーはプロセスを終了する。
- ログは `systemd journal` 経由（標準エラー出力）に出力する。

---

*設計書バージョン: 1.0.0 / 作成日: 2026-05-21*
