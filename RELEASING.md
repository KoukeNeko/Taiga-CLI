# 發行流程

## 前置條件

- `main` 與 `origin/main` 同步且工作樹乾淨。
- Release commit 已用 Git config 的 signing key 簽署，`git verify-commit HEAD` 與 `git log --format='%H %G? %s' -1` 都通過。
- `make verify` 與 `make test-integration` 通過。
- Version 使用 `vMAJOR.MINOR.PATCH`；pre-release 可使用 SemVer suffix。
- Repository 已有經 owner 選定的 `LICENSE`（目前為 MIT）；workflow 會在缺少此檔時主動拒絕發布。

## 本機重建套件

```sh
version=v0.1.0
commit="$(git rev-parse HEAD)"
epoch="$(git show -s --format=%ct HEAD)"
make release VERSION="$version" COMMIT="$commit" SOURCE_DATE_EPOCH="$epoch"
```

輸出位於 `dist/<version>/`：六個平台 archive、SPDX 2.3 SBOM 與 `SHA256SUMS`。Release build 固定使用 `CGO_ENABLED=0`、`-trimpath`、`-buildvcs=false`、空 Go build ID、排序後的 archive entry，以及 commit timestamp。相同 source、Go toolchain、version、commit 與 epoch 應產生相同 bytes。

Packager 只接受不存在或空的 output directory，避免先前版本或失敗建置的 stale asset 被上傳。若要重建同一版本，先將舊目錄移到備份位置，再重新執行。

## 建立 signed tag

禁止使用 lightweight 或未簽署 tag：

```sh
git tag -s "$version" -m "Taiga CLI $version"
git verify-tag "$version"
git push origin "$version"
```

Tag workflow 會透過 GitHub API 要求 annotated tag 與其 commit 的 `verification.verified` 都為 true，然後重新測試、封裝、驗證 checksums 與 embedded metadata，最後才執行 `gh release create --verify-tag --generate-notes`。推送 tag 前應先確認版本號，因發布後的 release asset 不應被覆寫。

## 發布後驗證

```sh
gh release view "$version"
gh release download "$version" --dir /tmp/taiga-release-check
cd /tmp/taiga-release-check
sha256sum --check SHA256SUMS
```

至少在一個下載 archive 執行 `taiga version --json` 與 `taiga doctor --json`。若 artifact 或簽章不正確，不要以同名檔案覆寫已發布 asset；應撤回 release、調查後使用新的版本號。
