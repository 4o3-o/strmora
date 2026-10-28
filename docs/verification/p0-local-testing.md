# P0 本地20000端口测试准备

用户于2026-10-28授权启动本地服务，并自行用容器中的Cloudflare Tunnel转发为HTTPS访问。具体域名、网关、密码和媒体映射仅保存于被Git忽略的 data/p0-local/session.md。

- 固定服务端口20000；监听用户指定Docker网关，使Tunnel容器可以访问。
- 使用用户指定的本机STRM只读复制，合成库不修改真实媒体源。
- 新增可选 `-access-log`，以JSONL记录UTC时间、请求方法/路径/状态/耗时，不记录查询参数、认证头、正文或跳转地址。
- scripts/p0-smoke.py支持本地和HTTPS基地址，输出脱敏报告；`--probe-media`额外做HEAD及两次1KiB Range，间隔3秒，不下载完整视频。
- 协议模拟写入12秒测试进度，保留供继续观看验证。
- 当前本地25项检查通过。默认Python UA访问实际源403，curl UA对照206；媒体探测显式配置UA，真实播放器行为单独验收。
- 当前无管理前端；浏览器查看 /healthz 或 /System/Info/Public，播放器使用Emby连接。
- 仍在P0，不推进P1；真实三客户端及公网验收另行记录。

## 公网复测

用户自行放行防火墙后，公网域名25项模拟全部通过（2026-10-28，Asia/Shanghai），包括登录、库/季度、多来源、302、字幕及Range、状态上报和实际视频HEAD及两次范围读取。原始脱敏JSON在被忽略的本地测试目录。没有宣称真实播放器兼容。

公网API入口也存在User-Agent差异：Python默认UA403，curl UA200；脚本支持 `--api-user-agent`、`--media-user-agent`，默认curl/8.5.0。它是模拟器行为，不会修改播放器请求或Tunnel配置。真实播放器是否被入口规则拦截需实际记录。
