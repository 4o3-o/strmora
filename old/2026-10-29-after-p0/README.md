# P0 之后实现的撤回归档（2026-10-29）

用户先要求撤回 P1 之后，随后更正为“只保留 P0，撤掉往后的所有更改”。本次以最后一个 P0 提交 `d2b8887d6caf51eed895bbb336e84ad8630a5f9f` 为产品代码基线。P1～P6 不再属于活动实现，旧进度中“完成”不作为当前交付结论。

## 归档内容

| 文件 | 内容 |
|---|---|
| `state.json` | P0 基线、本地撤回前 HEAD、GitHub 撤回前 main |
| `commits.txt` | P0 之后的完整提交编号和说明 |
| `local-post-p0-files.tsv` | 本地 P1～P6 改动清单 |
| `github-post-p0-files.tsv` | GitHub P1～P3 改动清单（现场查询 main 为 7467ac4，并非此前摘要所称的 P1） |
| `github-post-p0-source.tar.gz` | GitHub 已公开的 P0 后新增文件和修改文件的完整版本，保留相对目录 |
| `source-file-sha256.json` | 本地撤回前各改动文件的 SHA-256 |
| `SHA256SUMS` | 归档产生时的文件校验值；含仅本地存在的私有归档条目 |
| `private/local-HEAD-source.tar.gz` | 本地 deed394 完整源码原件，包含旧演示后台原有私人路径，因此不发布 |
| `private/local-post-p0.patch` | 本地 P0 后完整补丁，仅本地保存 |
| `private/local-history.bundle` | 撤回前全部本地 Git refs/历史，仅本地；保留未推送的P4～P6原提交，不让私人样例随推送父提交链公开 |
| `private/strmora-before-rollback` | 切换前二进制，仅本地保存 |
| `private/runtime-files.tar.gz` | 切换前本地配置、日志、STRM、测试记录等；不含在线数据库直接拷贝 |
| `private/database-backups/` | SQLite backup API 创建的一致性备份，仅本地保存 |
| `private/p0-http-audit.py`、`.json` | 本轮隔离HTTP审计脚本与结果；合成数据，不访问真实视频源 |
| `private/runtime-rollback.py`、`.json` | 运行服务回退脚本与数据库指纹、二进制hash验证结果 |

`old/.gitignore` 忽略私有区。公开归档只含此前 GitHub 已公开的源码；本地未推送实现和运行原件均完整保存在私有区，避免通过归档泄露私人媒体列表、凭据、设备 token 或数据库。

原始 Git 提交历史保留，以正常追加提交撤回，不 reset/force push。归档包不在 Go 包扫描中，不参与测试、构建或服务路由。

## 检查与恢复

查看公开归档：`tar -tzf old/2026-10-29-after-p0/github-post-p0-source.tar.gz`。

私有归档存在时，可在归档目录执行 `sha256sum -c SHA256SUMS`。GitHub 克隆不含私有文件，对应条目缺失是预期；公开源码包的 SHA 单独核验即可。

需要查看旧代码时先解压到项目外临时目录；不要直接覆盖活动树。完整旧提交仍可用 `git show <commit>:<path>` 检查。恢复旧实现、真实源写入和重新发布需后续明确授权。
