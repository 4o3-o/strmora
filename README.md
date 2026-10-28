# Strmora · 流幕

自部署媒体索引、TMDB 刮削、STRM 管理和独立 Emby API 兼容服务。

**当前仅保留 P0 协议原型。用户历史确认 Infuse、VidHub、网易爆米花基本连接与播放可用；不代表所有协议能力已验收。P1～P6 实现已撤回归档，当前没有管理后台。**

2026-10-29 重新基线：见 [P0 分步计划](docs/plans/2026-10-29-p0-rebaseline.md)、[P0 重新审计](docs/verification/2026-10-29-p0-audit.md) 和 [撤回归档说明](old/2026-10-29-after-p0/README.md)。本轮计划与审计不等于缺口已修复，不自动推进下一阶段。

需求边界和阶段验收以 [开发计划](docs/plans/2026-10-28-strmora-v1.0.md) 为准；实际结果见 [P0 执行记录](docs/verification/p0-progress.md)，接口和手工验收见 [P0 契约](docs/contracts/p0-emby.md)。

## P0 本地运行

需要 Go 1.22+、CGO 和 C 编译器。SQLite 驱动当前采用 go-sqlite3。

```bash
go test ./...
go build -o bin/strmora ./cmd/strmora
./bin/strmora tree -file /path/to/directory-tree.txt
```

`tree` 只读本地文件，默认输出统计，不输出私人目录名。`-entries` 可输出完整解析结果。编码支持 UTF-8、带 BOM 的 UTF-16 LE/BE；解析成功不代表远端导出完整。重复路径原样保留并报告，歧义未解决不得用于删除或整理。

运行协议原型前，通过环境配置：

| 变量 | 要求 |
|---|---|
| STRMORA_P0_USER | 测试用户名 |
| STRMORA_P0_PASSWORD | 至少 12 字节的测试密码，无默认密码 |
| STRMORA_P0_MEDIA_URL | 自己可访问的 HTTP(S) 视频完整 URL，无 URL 内嵌凭据 |

```bash
./bin/strmora -data ./data/p0 -listen 127.0.0.1:8096
```

不会访问视频源或 TMDB，不会修改网盘。启动时创建合成电影、双版本电影及两季电视剧，并为它们生成真实 STRM；首次默认指向同一个测试 URL。可直接修改 `data/p0/strm/*.strm` 为不同测试资源，已有 STRM 重启不覆盖。字幕是本地内置合成 SRT，图片为 1×1 占位图；可用于协议检查，不能代表真实海报墙体验。

协议同时支持根路径和 `/emby`；目标为上述第三方播放器，不提供官方 Emby 客户端兼容承诺。健康检查 `/healthz`，真实播放器接入前阅读契约。P0 不提供 `/admin` 或首页，浏览器直接访问这些路径可能收到认证错误 JSON，这不是管理页面。默认仅回环监听；可通过 SSH 隧道测试。P0 尚无完整限速、MFA 或多用户管理，不用于生产公网部署。受控 Tunnel 测试见 [20000端口测试记录](docs/verification/p0-local-testing.md)。

数据库、凭据、令牌、真实目录树和私人媒体链接不要提交仓库。已有 P0 用户的启动凭据必须一致；修改用户名/密码的管理功能在后续阶段实现。

可选 `-access-log /path/to/access.jsonl` 记录脱敏请求日志，省略查询参数、认证头和正文。

## v1.0 边界

- 输入 Local、OpenList API、WebDAV；目录树完整下载后在本地解析。
- 远端同步 STRM、字幕和 Markdown；NFO、图片独立保存在本地。
- 播放器访问媒体源，Strmora 不代理或转码视频。
- 计划、限流、取消、恢复和人工确认后续按阶段交付。

## 开源参考与致谢

Strmora 的 Emby 兼容网关与协议契约设计参考了以下优秀开源项目的实践经验与思路：

- [OpenList](https://github.com/OpenListTeam/OpenList) (AGPL-3.0)：参考其内置 Emby 兼容服务的请求路由分发、多源 Token 容错提取、会话鉴权交接（Token Handoff）以及辅助握手契约设计。
- [Emby-go](https://github.com/holll/Emby-go) (MIT)：参考其基于 Go 语言的轻量级 Emby 协议契约实现、STRM 302 重定向播放策略与会话播放状态设计。
- [MediaBridge](https://github.com/dlushu/MediaBridge) (AGPL-3.0)：参考其实机协议测试对照记录（真机 Emby 客户端对 DTO 字段完整性、相对路径处理及错误码的响应细节）。

## 许可

AGPL-3.0，见 LICENSE。用户自行配置 TMDB Key；TMDB 内容不随代码许可重新授权。当前无第三方元数据或私人媒体清单打包。
