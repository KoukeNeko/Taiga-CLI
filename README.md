# Taiga CLI

一套規劃以 Go 開發的 [Taiga 6](https://taiga.io/) 命令列工具，目標是提供接近 `gh` 的操作體驗，讓使用者、Shell、CI 與 Agent 都能安全地管理 Taiga 專案。

目前狀態：**設計與 API 研究完成，CLI 尚未開始實作。**

## 目標

- 直接使用 [Taiga REST API](https://docs.taiga.io/api.html)，不依賴瀏覽器自動化。
- 用 `project`、`issue`、`story`、`task`、`sprint` 等領域命令表達完整工作流程。
- 支援官方服務與自行架設的 Taiga 6，包括部署在 `/taiga/` 等子路徑的站點。
- 同時提供適合人類閱讀的輸出與穩定的 JSON 輸出。
- 保護 token、處理分頁、限流與 Optimistic Concurrency Control（OCC）。

預計的操作形式：

```console
taiga auth login --host https://taiga.example.com
taiga project use my-project
taiga issue list
taiga issue view 128
taiga issue create --subject "Fix login failure" --type Bug
taiga issue close 128
```

## 第一階段

第一個可用版本將聚焦：

- `auth login|logout|status`
- `project list|view|use`
- `issue list|view|create|edit|close|assign|comment`
- 多站點 profile 與安全 credential storage
- project ref、`project#ref` 和 Taiga URL 解析
- human／JSON output、明確 exit code 與 `doctor` 診斷

完成這個垂直切片後，再擴充 User Story、Task、Sprint、Epic、Wiki、附件與 Webhook。

## 設計原則

- CLI 命令表達使用者意圖，不逐一複製 REST endpoint。
- primary output 寫入 stdout，診斷與錯誤寫入 stderr。
- 非 TTY 模式不詢問、不顯示 spinner，也不阻塞 CI。
- 不接受命令列密碼，不在設定或 log 保存 token。
- 修改前取得最新資源版本；發生 OCC 衝突時不私自覆寫。
- 非冪等建立操作遇到模糊網路錯誤時不自動重送。
- 不建立 `util`、`common` 或萬用 CRUD 等模糊抽象。

## 本機相容性環境

開發機已使用官方 Docker images 建立 Taiga 6 測試站，URL 結構比照 example-project 的子路徑部署：

- 網站：<http://localhost:9000/taiga/>
- API：<http://localhost:9000/taiga/api/v1/>

這個環境只供本機開發與 API 驗證，不應直接暴露到公網。
