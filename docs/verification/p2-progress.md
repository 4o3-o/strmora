# P2 执行记录：TMDB 刮削与整理规划引擎

基线：docs/plans/2026-10-28-strmora-v1.0.md

## 1. 核心模块与实现 (2026-10-28)

- **文件名与目录启发式解析器 (`internal/metadata/parser.go`)**：
  - 提取文件名中的明确 `tmdb-ID`、年份（如 `(1995)`）、季度与集数（支持 `S01E01-E02` 合并集）、分段标签（`part1` / `cd1`）及清洗标题。
  - 针对真实 115 目录树特征验证通过，精准切分中英文标题与方括号压制说明。

- **TMDB API 轻量客户端 (`internal/metadata/tmdb.go`)**：
  - 针对影视主数据及复杂分集组设计：
    - `SearchMovie`：支持名称 + 年份模糊搜索；
    - `GetTVEpisodeGroups`：获取电视剧关联的所有分集组（如按年份排序、按剧情篇章排序）；
    - `GetTVEpisodeGroupDetail`：获取特定分集组有序的 groups 及 episodes 详情列表。

- **自定义命名模板引擎 (`internal/organizer/template.go`)**：
  - 支持 `{title}`、`{year}`、`{season}`、`{episode}`、`{part}`、`{version}`、`{ext}` 自由组合与字段缺失平滑退避。
  - 预设标准化：电影文件夹 `{title} ({year})`，分集文件 `{title} - S{season}E{episode}.{ext}`，分片保留 `{title} - {part}.{ext}`。

- **整理规划器与人工目录覆盖 (`internal/organizer/planner.go`)**：
  - `BuildPlan`：生成 `current_path` 到标准化 `default_target_path` 的完整移动改名路径。
  - `SetOverride`：支持管理员人工指定单独目标目录（`override_target_path`），执行时直接一步归类到位，不绕默认目录。

## 2. 自动化验证结果

- `go test -race ./...`：全部通过（覆盖 cmd、database、emby、inventory、jobs、metadata、organizer、sources）。
- `go vet ./...`：全部通过。
- `go build -o bin/strmora ./cmd/strmora`：二进制构建通过。
