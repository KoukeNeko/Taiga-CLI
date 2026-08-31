# Taiga CLI

以 Go 實作的 [Taiga 6](https://taiga.io/) 命令列工具。它提供適合人類閱讀的終端介面，以及給 Shell、CI 與 LLM／Agent 使用的穩定 JSON contract。

目前狀態：**Phase 1 與 User Story 垂直切片已實作，尚未發布正式版本。**

## 功能

- 自動從 Taiga frontend `conf.json` 發現 API，包括 `/taiga/` 子路徑部署。
- 互動式帳密登入、stdin token 登入、OS keyring 與 `TAIGA_TOKEN`。
- 多 profile、profile 預設 project 與 Git-local project mapping。
- Project list、view、use。
- Issue list、view、create、edit、close、assign、comment。
- User Story list、view、create、edit、close、move、assign、comment。
- Task list、view、create、edit、done、assign、comment。
- Taiga optimistic concurrency control（OCC）與 `--base-version`。
- Human output、versioned JSON、`--fields`、structured error 與固定 exit code。
- `--dry-run`、`--no-input`、redacted verbose logging。
- JSON Schema command descriptors 與四種 shell completion。
- `httptest` 單元測試和隔離的真實 Taiga Docker E2E。
- `taiga version` 顯示版本、commit、建置時間與平台。

## 建置

需要 Go 1.25 或更新版本。

```sh
make build
./bin/taiga version
```

安裝到 `~/.local/bin`：

```sh
make install
```

自訂安裝位置：

```sh
make install PREFIX=/usr/local
```

## 快速開始

互動登入會要求 username 與 no-echo password，並將 token 存入 OS keyring：

```sh
taiga auth login \
  --host https://taiga.example.com/taiga/ \
  --profile company
```

從 stdin 匯入既有 token：

```sh
printf '%s\n' "$TOKEN" |
  taiga auth login \
    --api-url https://taiga.example.com/taiga/api/v1/ \
    --profile company \
    --with-token
```

選擇專案並操作 Issue：

```sh
taiga project list
taiga project use example-project

taiga issue list
taiga issue view 42
taiga issue create --subject "Fix token refresh" --type Bug
taiga issue edit 42 --status "In progress"
taiga issue assign 42 --to alice
taiga issue comment 42 --body "Ready for verification"
taiga issue close 42 --status Closed
```

操作 User Story：

```sh
taiga story list --sprint backlog
taiga story view 51
taiga story create --subject "Add refresh-token rotation"
taiga story edit 51 --status "In progress"
taiga story assign 51 --to alice --to bob
taiga story move 51 --sprint sprint-27
taiga story comment 51 --body "Ready for review"
taiga story close 51 --status Closed
```

`story` 也可寫成 `userstory` 或 `us`。`story move --sprint backlog` 會把 Story 移回 backlog。

操作 Task：

```sh
taiga task list --story 51
taiga task view 72
taiga task create --story 51 --subject "Add API tests"
taiga task edit 72 --status "In progress"
taiga task assign 72 --to alice
taiga task comment 72 --body "Tests added"
taiga task done 72 --status Closed
```

Task 可以不屬於 Story；若指定 `--story`，Task 會自動繼承該 Story 的 Sprint。

Issue identifier 可以是裸 ref、`project#ref` 或 Taiga URL：

```text
42
example-project#42
https://taiga.example.com/taiga/project/example-project/issue/42
```

## Profile 與設定

一般設定使用作業系統的使用者設定目錄；token 不會寫入設定檔。

```toml
current_profile = "company"

[profiles.company]
api_url = "https://taiga.example.com/taiga/api/v1/"
project = "example-project"
```

設定優先序：

```text
command flag
→ TAIGA_PROFILE / TAIGA_API_URL / TAIGA_PROJECT / TAIGA_TOKEN
→ Git-local taiga.profile / taiga.project
→ current profile
→ safe defaults
```

將 profile／project mapping 限定在目前 Git repository：

```sh
taiga project use example-project --local
taiga config list --local
```

這些值保存在 `.git/config`，不會被 commit。

## Machine contract

Agent 必須明確要求 JSON；pipe 不會偷偷改變輸出格式：

```sh
taiga issue view example-project#42 \
  --json \
  --fields id,ref,subject,status,version \
  --no-input
```

```json
{
  "data": {
    "id": 123,
    "ref": 42,
    "subject": "Fix token refresh",
    "status": "In progress",
    "version": 7
  },
  "meta": {
    "contract": 1
  }
}
```

List 使用 `items` 與 `page`；錯誤只寫 stderr，成功資料只寫 stdout。可用下列命令取得 input/output schema：

```sh
taiga schema issue view --json
```

Exit codes：

| Code | Meaning |
| ---: | --- |
| 0 | success |
| 2 | usage / schema |
| 3 | authentication |
| 4 | forbidden |
| 5 | not found |
| 6 | OCC conflict |
| 7 | validation / ambiguity |
| 8 | throttled |
| 9 | transport / upstream |
| 10 | confirmation required |
| 11 | ambiguous commit |

Mutation 支援 dry run；它可以查詢與解析資料，但保證不送出 POST、PATCH 或 DELETE：

```sh
taiga issue edit 42 \
  --subject "New subject" \
  --dry-run \
  --json \
  --no-input
```

## 診斷

```sh
taiga doctor --host http://localhost:9000/taiga/
taiga doctor --api-url http://localhost:9000/taiga/api/v1/ --json
```

`doctor` 分別檢查 frontend discovery、API、authentication 與預設 project，不會輸出 token。

## 測試

快速測試不啟動 Docker：

```sh
make test
make test-race
make lint
```

真實 Taiga integration test：

```sh
make test-integration
```

Integration harness 使用獨立的 `taiga-cli-e2e` Compose project 與 `localhost:19000`，建立臨時帳號、Project 與 Issue，測完後刪除自己的 containers 和 volumes。它不會存取或清除日常使用的 `localhost:9000` Taiga。

## Shell completion

```sh
taiga completion bash
taiga completion zsh
taiga completion fish
taiga completion powershell
```

## 安全原則

- 不接受 command-line password。
- Authorization、password、token 不會出現在 verbose log。
- GET 才會進行 bounded automatic retry；POST／PATCH 不會盲目重送。
- Mutation 連線中斷且結果不明時回 `ambiguous_commit`。
- OCC conflict 不會自動 merge 或 overwrite。
- TLS verification 預設永遠開啟。
