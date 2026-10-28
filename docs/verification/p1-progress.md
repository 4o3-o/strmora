# P1 执行记录：媒体索引与媒体源适配器

基线：docs/plans/2026-10-28-strmora-v1.0.md

## 1. 核心模块与实现 (2026-10-28)

- **媒体源统一适配器 (`internal/sources`)**：
  - `local.go`：支持本地挂载目录/存储卷的列目录（`List`）、打开读取（`Open`）与防路径穿越（`resolve` 校验）。
  - `openlist.go`：对接 OpenList REST API (`/api/fs/list`、`/d/...`)，支持 Token 认证，并声明具备批量移动与批量重命名能力。
  - `webdav.go`：实现标准 WebDAV PROPFIND 深度 1 递归解析与 HTTP GET 读取，声明标准协议不具备单请求多文件批量移动能力。
  - 单元测试涵盖三类适配器的连通、能力校验、列目录与文件流读取。

- **媒体源共享限流器 (`internal/jobs`)**：
  - 基于令牌桶与并发信号量控制请求速率（QPS，支持 0.5 等浮点值）、最大并发（如并发 1）、修改操作最小保护间隔（如 10 秒）。
  - 支持异常熔断冷却机制：当捕获到上游 429 或限流风控信号（`ErrRateLimited`）时，自动进入冷却保护窗口，阻断高频重试。

- **目录树快照增量比对引擎 (`internal/inventory`)**：
  - 实现 `ComputeDiff`，将已严格解码的层级树快照进行路径比对，精确输出 Added / Removed / Unchanged 差异集合。
  - 保证界面翻页与差异比对完全基于本地快照完成，无需在上游频繁扫描或翻页。

- **SQLite 数据持久化与删除保护 (`internal/database`)**：
  - 扩展迁移建表：`media_sources`（媒体源配置与只读标识）、`source_entries`（文件与目录条目及 active/missing 状态）、`source_snapshots`（快照版本记录）。
  - 落实删除保护逻辑：仅在完整有效扫描确认缺失时标记状态，网络波动与残缺快照绝不破坏已有库。

## 2. 自动化验证结果

- `go test -race ./...`：全部通过（cmd、database、emby、inventory、jobs、sources）。
- `go vet ./...`：全部检查通过。
- `go build ./cmd/strmora`：二进制构建通过。
