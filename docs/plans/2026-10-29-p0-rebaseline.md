# P0 协议原型重新基线与分步开发计划

> 执行者按任务逐项检查；实施时使用 executing-plans 或 subagent-driven-development 流程。未验收项目不得勾选为完成。本次授权是归档、撤回、计划与审计，不包含下列缺口的功能开发。

**目标：** 用可复现的本地证据和真实客户端结果，证明 Strmora 能独立提供最小 Emby 兼容媒体库、STRM 鉴权 302、外部字幕和持久观看状态。

**架构：** Go 单进程、SQLite 本地 WAL、固定合成作品与真实 STRM 文件。播放器自行访问 STRM 中的媒体 URL；Strmora 不请求、不代理视频。P0 是协议验证原型，不是完整管理 WebApp。

**技术：** Go 1.22+、CGO/C 编译器、go-sqlite3、Argon2id、标准库 HTTP、Python 3 HTTP 验证脚本。

**依据：** `docs/plans/2026-10-28-strmora-v1.0.md` 第 1～5、10～12 节；`docs/contracts/p0-emby.md`；用户 2026-10-29 最新指令“只保留 P0”“根据 P0 需求制定分步计划并重新审计”。审计结果独立写入 `docs/verification/2026-10-29-p0-audit.md`。

## 1. 阶段边界与证据规则

- 代码基线固定为 `d2b8887d6caf51eed895bbb336e84ad8630a5f9f`（P0 最后提交）；P1～P6 全部归档撤回，包括主程序接线、数据库扩表、演示后台和发布配置。
- 保留需求基线、P0 协议与测试、原有开源参考和 AGPL-3.0。用户“仅支持第三方播放器，不做官方 Emby 客户端兼容”的边界继续有效。
- 单测试用户、固定媒体 ID、合成库、占位图片和内置 SRT 允许存在，但必须在界面说明、README 和验收记录中明确其性质。
- 不在 P0 引入 TMDB、真实媒体索引、重命名/移动、远端同步、管理工作台、MFA、30 设备服务承诺或生产发布。
- 原始媒体、目录树、播放链接、密码、token、运行数据库仅本地保存；不得因为归档而上传 GitHub。
- 后续开发、提交、推送遵循用户指令。本次 GitHub 撤回为用户已明确授权的一次操作；不等于授权此后自动推送。
- 自动化通过、用户确认、尚待真机验证分别记录；无充分证据不写百分比、不写“完全兼容”。
- 完成度用“已实现且本次通过 / 已实现但覆盖不足 / 明确缺陷 / 用户历史确认 / 待真机验证 / 阶段外”六种状态表达。

## 2. 交付文件和职责

| 文件 | 职责 |
|---|---|
| `cmd/strmora/main.go` | 启动、参数、目录树 CLI、退出与关闭数据库 |
| `cmd/strmora/accesslog.go` | 脱敏日志，仅记录方法、路由、状态、耗时、客户端与认证类别 |
| `internal/database/database.go` | P0 表、WAL、外键、busy timeout、迁移 |
| `internal/inventory/tree.go` | 纯本地目录树解码、层级、路径与歧义报告 |
| `internal/emby/catalog.go` | 固定作品、季集、多版本对应的原型身份 |
| `internal/emby/server.go` | 认证、初始化、浏览、播放、字幕图片与个人状态 |
| 同目录 `*_test.go` | 可重复行为测试，不访问真实媒体源 |
| `scripts/p0-smoke.py` | 独立 HTTP 冒烟；访问媒体源必须显式传 `--probe-media` |
| `docs/contracts/p0-emby.md` | 接口契约与逐客户端结果矩阵 |
| `docs/verification/p0-progress.md` | 历史进度，最新更正优先 |

## 3. 优先审查的五类风险

1. 一般 DTO 与 PlaybackInfo 的播放/字幕 URL 认证是否一致，尤其调用方只读取详情、不再请求 PlaybackInfo 时。
2. 匿名公开端点是否逐一列明；GET 与 HEAD 是否一致；用户状态和源链接不得被图片放行规则带出。
3. 进度写入失败、缺失位置、重启、同名播放会话对状态的影响；P0 不承诺多会话仲裁。
4. TXT 有 BOM、坏编码、重复、跳层或无末尾换行时是否丢失原名或被误判为完整快照。
5. 文档中“通过”是否有对应代码、测试或用户原话；本地 HTTP 成功不替代字幕显示、拖动和跨设备续播。

## 4. 执行顺序与独立验收任务

以下复选框表示本轮可交付验收，不能用历史摘要自动勾选；实际状态由审计报告对照填写。

### P0-01：确定保留基线并可恢复地撤回后续阶段

**文件：** `old/2026-10-29-after-p0/`、`AGENTS.md`、README、本计划与审计报告；恢复其余文件到 P0。

**输入/输出：** 输入本地 HEAD 与 GitHub main；输出独立归档、SHA-256 清单、P0 活动代码及一次正常前进的撤回提交。

- [ ] 记录本地/远端 SHA 和相对 P0 的改动清单，检查未提交文件，不能使用摘要替代 Git 事实。
- [ ] 本地完整原件和运行产物归档至 `old/` 的私有子目录并忽略 Git；GitHub 已公开的后续改动另作可发布归档。原有提交历史保留，不 force push。
- [ ] 恢复 P0 主程序和数据库迁移，移除 P1+ 活动包、后台、Compose/Dockerfile；保留本次计划和审计文档。
- [ ] 对照 `git diff d2b8887 -- cmd internal scripts go.mod go.sum .github`，期望产品代码无差异。
- [ ] 检查归档 hash 与源对象一致、私有文件不在待提交清单、`go list ./...` 不包含归档代码。
- [ ] 运行服务如仍使用旧程序，先确认 PID/可执行文件所有权，备份数据库再切换 P0；不能只换源码却报告运行服务已回退。

**验收：** 能从归档恢复旧状态；GitHub main 和本地采用同一撤回提交；P1+ 不再参与构建或运行。

### P0-02：本地基础与持久化

**文件：** `internal/database/database.go`、`database_test.go`、`cmd/strmora/main.go`。

**接口：** `database.Open(filename string) (*sql.DB, error)`；程序 `-data`、`-listen`、`-access-log`。

- [ ] 检查 P0 仅含迁移、设置、用户、设备、观看数据、播放会话表；WAL、外键开启，busy timeout=5000ms，synchronous=NORMAL。
- [ ] 执行 `go test -race ./internal/database -count=1 -v`，确认幂等打开及重开后的状态。
- [ ] 用独立临时目录启动、登录、记录 ServerId/token/进度、停止，再用同一目录启动；确认三者不变。
- [ ] 若实际关库重启证据不足，后续补 `TestProcessRestartPreservesIdentityTokenAndProgress`；不得用同一个 `sql.DB` 上重新 New 代替进程重启。
- [ ] 验证首次创建 STRM、已存在 STRM 不覆盖；缺失用户名、短密码、错误媒体 URL 启动失败。

**验收：** 新安装能启动；受控重启不丢身份、token 和状态；未声称实现生产备份/升级迁移。

### P0-03：目录树纯本地解析

**文件：** `internal/inventory/tree.go`、`tree_test.go`、CLI 分支。

**接口：** `inventory.Parse([]byte) (Tree, error)`；`strmora tree -file <local.txt> [-entries]`。

- [ ] 执行 `go test -race ./internal/inventory -count=1 -v`；核对 UTF-8/BOM、UTF-16 LE/BE、空行、中文空格标点、无末尾换行、坏编码、非法组件、层级断裂。
- [ ] 重复路径保留每一行并报告歧义；无法确定文件/目录的叶子保持 unknown，不凭扩展名强判。
- [ ] 只读本地真实样本，记录 bytes/encoding/count/duplicate 数，不输出私人目录名；CLI 代码路径无 HTTP 请求。
- [ ] 解析仅证明语法与层级有效，不证明远端导出完整；P0 不执行快照删除、整理或远端翻页。
- [ ] 对代码支持但未单测的 UTF-16 BE、UTF-8 无 BOM 等明确列出覆盖缺口；仅在后续实施授权后补测试。

**验收：** 同一本地文件可离线重复解析；失败不改现有数据；统计可核查，私有样本不入库。

### P0-04：登录、认证与初始化契约

**文件：** `internal/emby/server.go` 的 `New/extractToken/ServeHTTP/login/routes`，`server_test.go`，协议文档。

**接口：** 根路径与 `/emby` 下 Public Info、AuthenticateByName、Users/Me、Users/{id}、Views、VirtualFolders、DisplayPreferences。

- [ ] 检查 Argon2id 密码和 SHA-256 token 摘要持久化；错误密码401、缺失/无效 token401、其他用户路径或查询403。
- [ ] 覆盖 X-Emby-Token、X-MediaBrowser-Token、MediaBrowser/Emby 头、query 支持形式；测试多来源冲突，明确唯一优先顺序，不能依赖 map 遍历顺序。
- [ ] 为公开端点列出准确方法和路径，包含占位图片及初始化辅助接口；按 GET/HEAD/POST 测试，不用全局空200吞掉未知请求。
- [ ] 测试根和 `/emby` 初始化序列，Views 的库 ID 与 VirtualFolders 一致；不返回物理路径/凭据。
- [ ] 发现 Token 解析歧义或公开规则不一致时，记录复现与预期；建议补 `TestTokenConflictPrecedence`、`TestPublicMethodMatrix`，修复后回归三客户端。

**验收：** 明确契约可自动验证；三款第三方播放器可以建立连接；官方 Emby 客户端不在验收范围。完整 MFA/IP 封禁属后续，不把原型认证当生产安全方案。

### P0-05：合成库、浏览和基础状态

**文件：** `catalog.go`、`server.go` 的 `dto/list/listWhere/seasons/episodes/flag`。

**接口：** 2 库、2 部电影（其中一部双版本）、1 剧 2 季各 1 集；详情、搜索、ParentId、SeasonId、分页、收藏/已看/Resume。

- [ ] 验证稳定 ID、父子与季集关系；第二季显示别名 1997，但编号2；搜索和分页结果内容/数量正确。
- [ ] 检查未实现参数（如排序）不对外承诺；Latest 只是原型条目集合时不能称为真实新增时间排序。
- [ ] 验证收藏/已看 POST 和 DELETE 及 Resume 的进度>0且未看规则；同一用户重启仍保留。
- [ ] 确认只有固定合成数据；占位图片可以解码；不伪造 TMDB 数据、编码、分辨率、时长。

**验收：** 精确验证返回内容与状态变化，不能只验证 HTTP200。真实媒体库、真实海报、TMDB、管理网页不属于 P0。

### P0-06：真实 STRM → 鉴权入口 → 302，不转发视频

**文件：** `server.go` 的 `ensureSTRM/sources/playback/sourceID/stream`，`server_test.go`。

**接口：** GET/POST PlaybackInfo、GET/HEAD stream、MediaSourceId、DirectStreamUrl、两份独立 STRM。

- [ ] 准备不同合成目标 URL 的两版本 STRM；请求对应 MediaSourceId 必须分别返回精确 Location，默认版本与未知版本行为明确。
- [ ] PlaybackInfo 产生的流 URL 在不带认证头时仍可通过 query token 得到302；无 token401，撤销 token 后旧 URL401。
- [ ] 对一般列表/详情中的 MediaSources URL 作同样检查，确定它是否只是描述字段；若向客户端声明可直接播放，则认证交接必须成立，不能只测 PlaybackInfo。
- [ ] STRM 仅允许绝对 HTTP(S)、无内嵌凭据、fragment、多行；本地 GET/HEAD 均302，Range不触发本地代理读取。
- [ ] 用受控本地假上游计数器验证 PlaybackInfo/stream 对媒体上游请求数为0；不要靠真实视频播放下载来证明“不转发”。
- [ ] 真机分别验证两版选片与最终源拖动；最终206由源提供，不把 Strmora 的302当作最终 Range 成功证据。

**验收：** 302 Location 与原始 STRM 一致、上游访问计数为0；“两版本 API 返回”与“真机实际选版”分别记录。

### P0-07：外部字幕与进度闭环

**文件：** `server.go` 的 `subtitle/progress/data/flag/logout`；数据库表；HTTP 验证脚本。

**接口：** MediaStreams 外部 SRT、字幕 GET/HEAD/Range、Playing/Progress/Stopped、PositionTicks（100ns/tick）。

- [ ] 字幕 URL 无认证头但有交接 token 时可读；有效 source/index200，未知source/index404；Range=0-15 返回206和16字节，越界416。
- [ ] 明确 P0 字幕为内存合成字符串，不是实际字幕文件下载/缓存；ASS/SUP 与字幕转换不纳入通过。
- [ ] 三种播放事件可保存非负位置；缺失/null/负数拒绝且原进度不变；显式0合法；未知作品404。
- [ ] 校验会话与个人状态事务一致；停止后重开可续播；注销后旧 token401（P0 注销撤销测试用户所有设备，不能称设备级撤销）。
- [ ] 记录 P0 最后到达写入、无自动已看阈值、无跨设备仲裁等限制，真机没上报时不推算进度。

**验收：** 自动化验证语义；每款客户端的字幕显示、拖动后进度、退出重进续播须独立实测。

### P0-08：运行和脱敏证据

**文件：** `accesslog.go`、`accesslog_test.go`、`scripts/p0-smoke.py`、本地忽略的测试配置。

- [ ] 执行 `go test -race ./cmd/strmora -count=1 -v`，确认 token、查询、请求头、正文、签名 Location 和设备信息不进日志。
- [ ] 固定20000端口只保留原有测试监听与 Tunnel 路由；不修改防火墙/Tunnel，不新增外网服务。
- [ ] HTTP 验证使用独立临时数据库/合成 URL，禁止默认跟随302到真实网盘；线上只作轻量只读健康和连接检查。
- [ ] 运行 `go test -race ./... -count=1`、`go vet ./...`、`go build -o bin/strmora ./cmd/strmora`；记录工具链版本、package数、实际结果。
- [ ] 样本、原始日志、token、数据库和旧二进制的备份只在 `old/` 私有区；公开报告只写统计与可复现步骤。

**验收：** 测试失败如实记录；修复必须另行按计划执行，不能在审计中悄悄修改产品行为。

### P0-09：真机矩阵、审计闭环与阶段出口

**文件：** `docs/contracts/p0-emby.md`、`docs/verification/p0-progress.md`、本轮审计、README、AGENTS。

- [ ] 保留用户历史原话“三款播放器均正常播放”作为基本连接/播放证据，不能推导每款字幕/选版/拖动/重启均通过。
- [ ] 为 Infuse、VidHub、网易爆米花逐项记录：准确版本/build、OS、测试日期、根或 `/emby`、登录、浏览、单片、两版本、两季、字幕、拖动、进度、重启续播、注销。
- [ ] 缺失证据标为待测；用户未给具体版本时明确“当时称最新版，具体版本未知”。
- [ ] 审计按本计划任务号建立需求→代码→测试→真机→结论追踪表；每项缺陷含复现命令/响应、影响、最小修复建议。
- [ ] 更正文档中互相冲突的“尚未实测”和“全部完成”，README 明确只有 P0，无后台；新指令收紧阶段写入 AGENTS。

**出口条件：** P0 自动化重新通过、关键契约缺陷关闭、真实目录树解析可复现、三款基本连接播放有证据，且字幕/选版/进度等承诺与实测范围一致。完成审计不等于补齐缺口，更不自动进入 P1。

## 5. 后续执行规则

1. 本次先完成归档与重新审计，报告事实和待办，不开发 P1。
2. 用户审阅本计划和审计后，再决定执行哪些 P0 修复任务；每个修复先复现，再最小修改、对应回归，必要时请用户进行真机复测。
3. 未关闭缺陷与未测能力持续列出，不通过修改验收描述把缺口变成“已完成”。
4. 未来 P1 必须另有可操作管理流程和端到端验收计划；本次撤回不保留其已完成结论。
