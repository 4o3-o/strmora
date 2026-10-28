# Third-party notices

Strmora source is licensed under AGPL-3.0; see LICENSE. Original project: Strmora contributors, 2026.

Go module dependencies are pinned in go.mod/go.sum. Their original licenses apply:

- github.com/mattn/go-sqlite3: MIT; wraps SQLite (public domain). Preserve upstream notices when redistributing binaries/source.
- golang.org/x/crypto and golang.org/x/sys: Go BSD-style license; preserve upstream LICENSE/PATENTS notices in distributions.

No Emby implementation code, TMDB datasets/artwork, private media files, or real directory-tree export are bundled. Emby, Infuse, VidHub and other third-party names identify compatibility targets, not affiliation or certification. P0 has no TMDB calls; TMDB attribution and current usage requirements must be verified when P2 adds TMDB.

### 架构与协议设计参考声明

本项目在独立实现 Emby 协议网关层时，研读并参考了以下开源项目的架构思路与真机协议调试记录：

- OpenList (https://github.com/OpenListTeam/OpenList) - GNU AGPL v3.0
- Emby-go (https://github.com/holll/Emby-go) - MIT License
- MediaBridge (https://github.com/dlushu/MediaBridge) - GNU AGPL v3.0

本项目未直接打包或捆绑上述项目的核心业务代码。协议契约设计中的相关模式（如大小写兼容提取、302播放重定向、辅助留白端点等）遵循公有协议约定并由 Strmora 独立实现。
