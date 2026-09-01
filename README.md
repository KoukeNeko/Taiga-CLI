# Taiga CLI

以 Go 實作的 [Taiga 6](https://taiga.io/) 命令列工具。它提供適合人類閱讀的終端介面，以及給 Shell、CI 與 LLM／Agent 使用的穩定 JSON contract。

目前狀態：**Phase 1–3 operator workflows 已實作，尚未發布正式版本。**

## 功能

- 自動從 Taiga frontend `conf.json` 發現 API，包括 `/taiga/` 子路徑部署。
- 互動式帳密登入、stdin token 登入、OS keyring 與 `TAIGA_TOKEN`。
- 多 profile、profile 預設 project 與 Git-local project mapping。
- Project list、view、use、create、edit、delete。
- Project portable dump export/import，支援 plain JSON 與 gzip、串流上傳及非同步狀態。
- Project member/invitation 與 Role/permission 管理。
- Webhook list、view、create、edit、test、delete，secret 永不回顯。
- Epic、Story、Task、Issue 的 Custom field definition 與 OCC value merge。
- Epic list、view、create、edit、close、跨專案 Story link/unlink、watch 與 history。
- Issue list、view、create、edit、close、assign、comment。
- User Story list、view、create、edit、close、move、assign、comment。
- Task list、view、create、edit、done、assign、comment。
- Sprint list、view、create、edit、close、reopen。
- Issue、Story、Task、Epic 的 vote、unvote，以及 watch、unwatch、activity/comment history。
- Project timeline，以及 Project backlog/velocity、Issue 趨勢、Member 貢獻與 Sprint burndown stats。
- Issue、Story、Task、Epic、Wiki 的附件 streaming upload、list、view、edit、delete。
- Wiki list、view、create、edit、delete、watch 與 history。
- Taiga optimistic concurrency control（OCC）與 `--base-version`。
- Human output、versioned JSON、`--fields`、structured error 與固定 exit code。
- `--dry-run`、`--no-input`、redacted verbose logging。
- Epic、Story、Issue、Task 的 bounded native bulk create，支援檔案/stdin、共同 metadata 與完整 dry-run。
- JSON Schema command descriptors 與四種 shell completion。
- 依 profile、API 與 project 隔離的 completion metadata cache，支援 stale-on-error。
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

管理 Project：

```sh
taiga project create --name "Mobile App" --template scrum
taiga project edit mobile-app --description "Product delivery" --kanban=true
taiga project delete mobile-app --yes
```

Project 預設建立為 private；使用 `--public` 明確建立公開專案。永久刪除由 Taiga 非同步執行且不可還原，非互動模式必須提供 `--yes`。目前 Taiga 6 REST API 將 `archived_code` 設為唯讀且沒有 archive action，因此 `project archive|unarchive` 會清楚回報 `unsupported_capability`，需由站台管理者處理。

匯出與匯入 Project dump：

```sh
taiga project export example-project --format gzip --json
taiga project import ./example-project-export.json.gz --dry-run --json
taiga project import ./example-project-export.json.gz --yes --json
cat example-project-export.json | taiga project import - --yes --json
```

Taiga production deployment 通常以背景工作產生 export／執行 import。此時 CLI 會回報 `status: "accepted"`、task ID 與 `verified: false`；完成或失敗結果由 Taiga 寄送 email。停用 Celery 的同步部署則會直接回傳 `status: "ready"` 加下載 URL，或 `status: "created"` 加已建立的 Project。Export request 雖使用 Taiga 的 GET endpoint，但會建立背景工作，因此 CLI 刻意不自動 retry。Import 會先在本機串流驗證 JSON／gzip 格式，不會把整份 dump 載入記憶體；非互動模式必須明確提供 `--yes`。

管理成員與 Role：

```sh
taiga role list
taiga role create --name Reviewer --computable=false
taiga role edit reviewer --permission view_us --permission comment_us
taiga member add alice@example.com --role reviewer
taiga member edit alice@example.com --role ux --admin=false
taiga member remove alice@example.com --yes
taiga role delete reviewer --move-to ux --yes
```

`member add` 可加入既有 username/email，或為未知 email 建立 invitation。Taiga 會保護 owner 與最後一位 active admin；CLI 不會繞過這些限制。刪除仍有 members 的 Role 必須明確提供 `--move-to`。

管理 Webhook：

```sh
taiga webhook create --name CI --url https://ci.example.com/taiga --secret "$WEBHOOK_SECRET"
taiga webhook list
taiga webhook test CI
taiga webhook edit CI --url https://ci.example.com/hooks/taiga
taiga webhook delete CI --yes
```

Webhook signing secret 只會送往 Taiga API，不會出現在成功輸出或 dry-run。`webhook test` 要求 Taiga server 自己發送測試事件，CLI 不會直接連第三方 URL。

管理 Custom fields：

```sh
taiga custom-field create issue --name Environment --type dropdown --option staging --option production
taiga custom-field list issue
taiga custom-field set issue 42 --value Environment='"staging"' --value Attempts=3
taiga custom-field values issue 42
taiga custom-field set issue 42 --unset Environment
taiga custom-field delete issue Environment --yes
```

`--value` 的右側會先按 JSON 解析，因此字串可加 JSON 引號，boolean/number/null 會保留型別；無法解析的值視為普通字串。更新前會 GET 現值並 merge，再以 custom-values version 執行 OCC PATCH，避免覆蓋其他欄位。

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
taiga task unassign 72
taiga task comment 72 --body "Tests added"
taiga task done 72 --status Closed
taiga task reopen 72 --status New
taiga task move 72 --story 51
taiga task move 72 --sprint sprint-27
taiga task move 72 --sprint backlog
```

Task 可以不屬於 Story；若指定 `--story`，Task 會自動繼承該 Story 的 Sprint。`--sprint backlog` 會同時解除 parent Story 與 Sprint。

管理 Sprint：

```sh
taiga sprint list --state open
taiga sprint view sprint-27
taiga sprint create --name "Sprint 27" --start 2026-09-01 --finish 2026-09-14
taiga sprint edit sprint-27 --finish 2026-09-16
taiga sprint close sprint-27
taiga sprint reopen sprint-27
```

Sprint 日期使用 `YYYY-MM-DD`；`sprint` 也可寫成 `milestone`。

搜尋目前專案：

```sh
taiga search "token refresh"
taiga search "login" --type issue
taiga search "API tests" --type task --json --fields kind,ref,subject
```

Search 支援 epic、story、task、issue、wiki；Taiga server 每次最多回傳 150 筆。

查看跨資源 Timeline 與統計：

```sh
taiga timeline
taiga timeline --only-relevant=false --page 2 --limit 50

taiga stats project
taiga stats issues example-project
taiga stats members
taiga stats sprint sprint-27
taiga stats discover
taiga stats system
```

`timeline` 使用目前選取的 Project，預設排除低訊號 change/delete；輸出會將 Taiga event 正規化為 resource、action、ref/slug、subject、user、comment 與 changes。`stats project|issues|members` 可省略 project slug 並使用目前 Project；`stats sprint` 使用目前 Project 解析 Sprint。`stats discover` 可查公開可探索的 Project 數量。`stats system` 只有站台管理者啟用 Taiga `STATS_ENABLED` 時才存在，未啟用的標準部署會回報 `not_found`。

批次建立工作項目：

```sh
printf '%s\n' "First issue" "Second issue" > issues.txt
taiga batch create issue issues.txt --sprint sprint-27 --dry-run --json
taiga batch create issue issues.txt --sprint sprint-27 --yes --json

cat stories.txt | taiga batch create story - --status New --yes
taiga batch create task tasks.txt --story 51 --yes
taiga batch create epic epics.txt --yes
```

Batch input 每個非空白行是一個 subject，上限 1000 筆與 4 MiB；同一批可套用共同 `--status`。Issue 可指定共同 `--sprint`；Task 必須指定 `--sprint`，或以 `--story` 從 parent Story 推導 Sprint。Taiga 原生 bulk endpoint 不支援每筆不同 description/assignee，也不保證跨資源原子交易。CLI 在成功回應後核對回傳筆數；若不一致會回報 `ambiguous_commit`，要求先 list 確認，避免盲目重送。非互動執行必須提供 `--yes`。

管理附件：

```sh
taiga attachment list issue 42
taiga attachment add issue 42 ./error.log --description "Build failure"
cat error.log | taiga attachment add issue 42 - --name error.log
taiga attachment view issue 17
taiga attachment edit issue 17 --description "Resolved" --deprecated
taiga attachment delete issue 17 --yes
taiga attachment add epic 8 ./proposal.pdf
taiga attachment add wiki api-guide ./diagram.png
```

Attachment 支援 issue、story、task、epic、wiki。Wiki 使用 slug，其餘資源使用 ref。非互動刪除必須明確提供 `--yes`；upload 採 streaming，不會先把整個檔案載入記憶體。

關注工作項目與查看歷史：

```sh
taiga issue watch 42
taiga issue vote 42
taiga issue history 42
taiga issue history 42 --type comment
taiga issue unvote 42
taiga issue unwatch 42

taiga story history 51 --type activity
taiga task history 72 --page 2 --limit 20
```

Watch/unwatch 支援 issue、story、task、wiki、epic；vote/unvote 支援 issue、story、task、epic。命令會在 mutation 後分別回讀 `is_watcher` 或 `is_voter` 確認狀態。History 的 `--type` 可用 `all`、`activity`、`comment`。

管理 Wiki：

```sh
taiga wiki list
taiga wiki view api-guide
taiga wiki create --slug api-guide --body-file guide.md
printf '%s\n' '# Updated guide' | taiga wiki edit api-guide --body-file -
taiga wiki watch api-guide
taiga wiki history api-guide --type activity
taiga wiki unwatch api-guide
taiga wiki delete api-guide --yes
```

Wiki identifier 支援裸 slug、`project#slug` 與 Taiga Wiki URL。Edit 使用 OCC；非互動刪除必須提供 `--yes`。

管理 Epic 與跨專案 Story 關聯：

```sh
taiga epic list
taiga epic create --subject "Unify authentication"
taiga epic edit 8 --status "In progress" --assignee alice
taiga epic link 8 --story mobile-app#42
taiga epic stories 8
taiga epic unlink 8 --story mobile-app#42
taiga epic watch 8
taiga epic history 8 --type activity
taiga epic close 8
```

Epic 與 Story 是多對多關聯；`link`／`unlink` 接受其他專案的 Story ref 或 URL，不會把 Story 視為 Epic 的單一 parent。

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

Completion 會在 2 秒 timeout 內提供 Project、work item ref、status、member、Sprint 與 Issue metadata 候選。成功結果以 `0600` 權限快取 5 分鐘；網路失敗時可使用 24 小時內的 stale cache，不儲存 token，且 cache 損壞不影響 shell。

## 安全原則

- 不接受 command-line password。
- Authorization、password、token 不會出現在 verbose log。
- GET 才會進行 bounded automatic retry；POST／PATCH 不會盲目重送。
- Mutation 連線中斷且結果不明時回 `ambiguous_commit`。
- OCC conflict 不會自動 merge 或 overwrite。
- TLS verification 預設永遠開啟。
