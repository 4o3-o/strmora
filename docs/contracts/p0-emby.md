# P0 Emby 最小协议契约与手工验收

这是原型接口契约，客户端实测前不声称兼容完整 Emby。根路径和 `/emby` 路径同时可用。所有业务接口都需设备 token，只有 health、Public Info 和密码登录公开。

## 合成媒体

movies 库：movie-p0（单版本），movie-versions（双版本）。tv 库：series-p0 → season-1/season-2 → episode-1/episode-2。第二季显示名 1997，IndexNumber=2。无真实 TMDB 信息、时长、视频/音轨探测结果；技术参数未知，不冒充 4K 或具体编码。图片为占位 PNG，SRT 为原型字幕。

## 接口

| 方法和路径 | 行为 |
|---|---|
| GET /healthz | 数据库 ping，不代表客户端兼容 |
| GET /System/Info/Public | 稳定 server ID、名称、原型版本 |
| POST /Users/AuthenticateByName | JSON Username/Pw，返回 User/AccessToken/SessionInfo |
| GET /Users/Me、GET /Users/{id} | 当前已认证测试用户；其他用户403，无令牌401 |
| GET /Users/{user}/Views | movies、tv 库 |
| GET /Library/VirtualFolders | 授权逻辑库数组，ItemId对应库ID，不返回物理路径 |
| GET /DisplayPreferences/{id} | P0 默认显示设置；支持UserId/userId边界，尚不提供设置写入 |
| GET /Users/{user}/Items，GET /Items | ParentId、Recursive、IncludeItemTypes、SearchTerm、StartIndex、Limit |
| GET /Users/{user}/Items/Latest，…/Resume | 合成库最近条目/有进度未看条目；尚无真实时间排序 |
| GET /Users/{user}/Items/{id}，GET /Items/{id} | 条目、元数据、当前用户状态 |
| GET /Shows/{id}/Seasons，…/Episodes | 季度/单集；Episodes 支持 SeasonId |
| GET /Items/{id}/Images/{kind} | 本地占位 PNG |
| GET/POST /Items/{id}/PlaybackInfo | MediaSources、DirectStreamUrl、外挂字幕、PlaySessionId |
| GET/HEAD /Videos/{id}/stream（或 stream.mkv） | 从本地 STRM 读取 URL，鉴权后仅 302 |
| GET/HEAD /Videos/{id}/{source}/Subtitles/0/Stream.srt | 原样合成 SRT，支持本地文件内容 Range |
| POST /Sessions/Playing、…/Progress、…/Stopped | ItemId、PositionTicks、PlaySessionId 持久化 |
| POST/DELETE /Users/{user}/PlayedItems/{id} | 已看 |
| POST/DELETE /Users/{user}/FavoriteItems/{id} | 收藏 |
| POST /Sessions/Logout | P0 单测试用户所有设备注销（后续替换为设备级） |

令牌接受 X-Emby-Token、X-MediaBrowser-Token、api_key 或 MediaBrowser Token 请求头。密码 Argon2id，设备 token 只持久化 SHA-256 摘要；响应中的 token 不写日志。错误通过 HTTP 400/401/403/404/500 表示。

播放事件 PositionTicks 必填且非 null、非负；缺失请求返回400且不修改已有进度，显式0合法。

P0 测试用户固定内部 ID user-p0。API 拒绝其他 user 路径和 UserId 查询。暂不做用户管理、自动已看阈值、过期会话清理、实时事件和多会话迟到进度仲裁；进度当前按最后到达写入，后续必须补足后才承诺多设备体验。

播放信息不发起视频请求。STRM 仅允许 HTTP(S) URL，禁止多行、URL内嵌密码、非HTTP协议。双版本可分别编辑对应 STRM；原型不会下载或读取视频长度。需独立验证最终地址 HEAD/Range/206/拖动。

## 三客户端验收（待执行）

每个目标分别记录平台、系统、客户端版本、接入 URL 形式、测试时间（Asia/Shanghai），不得保存密码/token/带签名URL。

1. 按 README 配置可播放的测试资源；使用回环或隧道，确认客户端能访问最终视频地址。
2. 密码登录、看到两个库和合成条目；错误密码拒绝。
3. 浏览电影、两季电视剧；确认 season-2 别名与序号；搜索与分页。
4. 单电影播放，观测 302 和最终 Range/206，拖动和恢复；服务无视频带宽转发。
5. 双版本选择，分别访问对应 STRM；两季单集正确访问。
6. SRT 被发现、下载并显示；原始 ASS/SUP 不在此合成用例中，不据此声称已兼容。
7. 开始/进度/停止上报，退出重进继续观看；重启服务身份/令牌/进度仍在。
8. 收藏、已看；注销后旧 token 失败。
9. 记录未实现端点、字段差异与失败，按标准 Emby 契约补齐，不写播放器私有进度逻辑。

| 客户端 | 版本 | 登录/库 | 302/拖动 | 字幕 | 进度 | 结果 |
|---|---|---|---|---|---|---|
| Infuse | iOS，用户称最新版，build待提供 | 补齐库和显示设置后用户确认连接成功 | 入口302已观测，实际播放/拖动待确认 | 待测 | 待测 | 连接通过，整体待验收 |
| VidHub | iOS，用户称最新版，build待提供 | 用户确认可连接 | 初次失败；用户随后复测确认可播放 | 待测 | 待测 | 基本播放已确认，拖动/字幕/进度仍待验收 |
| 网易爆米花 | 版本/平台待提供 | 用户已登录浏览；修复后用户确认恢复连接 | 用户已播放，入口日志302；拖动待测 | 待测 | 播放事件/进度已收到 | 已发现并修复用户查询404，尚未整体通过 |

P0 退出条件：本地自动化通过 + 原始目录树解析通过 + 三客户端实际结果记录及关键缺失修复。当前不能因本地测试通过把整个 P0 标记完成。

## 网易爆米花连接检查修复（2026-10-28）

实际客户端反复 GET /emby/Users/user-p0，旧实现只有 /Users/Me，返回404。补齐按ID读取当前用户接口，同时保留401/403边界；测试先复现404再通过。该路由对应 [Emby官方getUsersById](https://dev.emby.media/reference/RestAPI/UserService/getUsersById.html)。

公网根路径和/emby前缀均已验证200。服务重启前签发的token、ServerId、观看状态均保留。用户刷新后明确反馈“已恢复连接”，本项已有真实客户端确认。图片401和 HideFromResume/NextUp缺失另行跟踪，不在此次用户查询修复中放宽鉴权或伪造成功。

## Infuse 初始化接口修复与 VidHub 排查（2026-10-28）

Infuse实际请求依次在 /Library/VirtualFolders 和 /DisplayPreferences/usersettings 遇到404。分别补齐逻辑库数组（含Name/ItemId/CollectionType/Locations空数组）和默认显示设置（Id/Client/SortBy/SortOrder/CustomPrefs），保留认证和用户隔离；并非所有未知接口统一返回空200。用户复测确认完成连接。显示设置当前无用户自定义覆盖，不提供写入。

参考模型：[VirtualFolderInfo](https://dev.emby.media/reference/RestAPI/LibraryStructureService/getLibraryVirtualfoldersQuery.html)、[DisplayPreferences](https://dev.emby.media/reference/RestAPI/DisplayPreferencesService/getDisplaypreferencesById.html)。VirtualFolders旧路由按客户端实际请求提供数组，不混用Query路由的Items封装。

访问日志增加固定客户端类别和凭据传输类别，只记录类别、不记录UA原文、设备ID、查询参数/头/令牌值。VidHub先前失败原因尚未定论；用户随后复测明确“vidhub可以播放视频了”，基本播放记为通过，但不将其归因于Infuse初始化接口修复。
