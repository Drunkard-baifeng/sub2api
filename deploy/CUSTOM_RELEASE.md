# 定制版发布与 Linux 安装

本仓库维护分支为 `custom`，发布仓库为 `Drunkard-baifeng/sub2api`。
保留完整发布流程：先构建前端并嵌入 Go 程序，再生成多平台安装包和 Docker 镜像。

## 发布内容

| 类型 | 平台 / 地址 |
| --- | --- |
| Linux 程序 | amd64、arm64，`.tar.gz` |
| Windows 程序 | amd64，`.zip` |
| macOS 程序 | amd64、arm64，`.tar.gz` |
| 文件校验 | `checksums.txt`，SHA-256 |
| Docker 镜像 | `ghcr.io/drunkard-baifeng/sub2api`，linux/amd64 和 linux/arm64 |
| Docker Hub（可选） | 配置凭据后额外推送 `<用户名>/sub2api` |

服务器使用二进制安装包时不需要安装 Docker、Go、Node.js 或 pnpm，
但仍然需要可连接的 PostgreSQL 和 Redis。Docker 镜像发布与二进制部署方式互不影响。

## 首次准备

1. 将发布配置提交并推送到 `origin/custom`。
2. 在 GitHub 仓库的 **Actions** 页面启用工作流；Fork 仓库若出现启用提示，先确认启用。
3. 在 **Settings → Secrets and variables → Actions → Variables** 中，
   确保 `SIMPLE_RELEASE` 没有设置为 `true`（删除或设为 `false`）。
   完整发布不勾选手动运行中的 `simple_release`。
4. 工作流已声明 `contents: write` 和 `packages: write`，使用 GitHub 自动提供的 `GITHUB_TOKEN`。
   GHCR 不需要另建同名 Secret；仓库/组织策略需允许这些权限，`custom` 分支保护规则需允许版本文件回写。
5. 默认只向 GHCR 发布 Docker 镜像。需要同时推送 Docker Hub 时，再添加 Actions Secrets：
   `DOCKERHUB_USERNAME`、`DOCKERHUB_TOKEN`。不使用 Docker Hub 时两项都不配置。
6. 首次发布 GHCR 包后，检查包 **Package settings → Change visibility**，
   需要匿名拉取时将包设为 Public；已有同名包则检查是否授予本仓库 Actions 写入权限。

## 先试跑，不发布

在 **Actions → Release → Run workflow**：

- **Use workflow from**：选择 `custom`。
- **tag**：填写 `custom`。
- **simple_release**：不勾选。
- **dry_run**：勾选。

这会编译五个平台的程序、校验压缩包，并构建两种架构的 Docker 镜像，
但不会创建 Release、推送镜像或回写版本文件。
试跑版本读取当前 `backend/cmd/server/VERSION`，不需要先创建 tag。
可在该次运行的 Artifacts 下载 `release-binary-*` 和 `release-dry-run-report` 检查产物；
Docker OCI 文件仅用于 runner 内验证，不作为这些下载产物上传。

如果没有 **Run workflow** 按钮，检查仓库是否已启用 Actions，以及默认分支是否存在
带 `workflow_dispatch` 的 `release.yml`。选择 `custom` 后再运行，不要误选上游的 `main`。

## 正式发布

定制稳定版本格式为 `v<上游基础版本>-custom.<递增序号>`，例如：

- 首次：`v0.2.13-custom.1`
- 同一基础版本的下一次定制更新：`v0.2.13-custom.2`
- 合并新的基础版本后：例如 `v0.2.14-custom.1`

虽然后缀包含连字符，本项目将严格匹配 `X.Y.Z-custom.N`（N 从 1 开始）的完整发布
作为正式 Release，并设为 Latest，供安装脚本和后台更新读取。
`rc`、`beta` 等仍按预发布处理。当前镜像脚本也会更新 `latest` 等浮动标签，
因此此发布通道只用于准备上线的稳定版本，不用它试发 rc/beta；验证使用 `dry_run`。
仅镜像的 `simple_release` 模式保留上游预发布策略，不用于本定制二进制发布通道。

确认试跑通过、代码已提交推送后，在项目根目录执行以下 PowerShell 命令：

```powershell
git switch custom
git pull --ff-only origin custom
git status --short
# 确认工作区没有待提交改动，然后执行：
git tag -a v0.2.13-custom.1 -m "release: v0.2.13-custom.1" -m "定制版首次发布：品牌界面、充值与兑换优化，多平台程序及 Docker 镜像。"
git push origin v0.2.13-custom.1
```

推送 tag 会自动启动 Release 工作流，不必再手动运行一次。
后续每次修改版本号和发布说明；不要移动已发布的 tag。
要重新执行失败的构建，先检查失败步骤和是否已有部分上传的产物，再决定重跑或递增版本。

成功后，在本仓库 Releases 下载产物。普通 Intel/AMD Linux 服务器选择
`sub2api_0.2.13-custom.1_linux_amd64.tar.gz`；ARM64 服务器选择 `linux_arm64` 包。
用 `uname -m` 判断：`x86_64` 对应 amd64，`aarch64` 对应 arm64。

全部发布步骤成功后，工作流会将版本号写回 **custom** 分支的
`backend/cmd/server/VERSION`。本地继续开发前执行 `git pull --ff-only origin custom`。

## Linux 首次安装（不用 Docker）

先准备 PostgreSQL 和 Redis，以及该应用独立使用的数据库/用户。
数据库用户需要拥有应用数据库和迁移表的权限。
在正式发布成功后，下载该版本的安装脚本并运行：

```bash
curl -fsSL https://raw.githubusercontent.com/Drunkard-baifeng/sub2api/v0.2.13-custom.1/deploy/install.sh -o install.sh
sudo bash install.sh install -v v0.2.13-custom.1
```

脚本默认安装至 `/opt/sub2api`，创建 `sub2api` systemd 服务，并询问监听地址和端口。
按脚本输出的地址打开安装向导，填写服务器实际使用的 PostgreSQL、Redis 和管理员信息。
前端已经打包进程序，不需要再单独启动 Vite。

检查服务：

```bash
sudo systemctl status sub2api --no-pager
sudo journalctl -u sub2api -n 100 --no-pager
```

已有实例不要重新初始化数据库。升级前备份数据库和配置，下载目标版本的脚本后执行：

```bash
# 示例：该版本发布成功后再执行
curl -fsSL https://raw.githubusercontent.com/Drunkard-baifeng/sub2api/v0.2.13-custom.2/deploy/install.sh -o install.sh
sudo bash install.sh upgrade -v v0.2.13-custom.2
```

后台更新和回滚信息也来自 `Drunkard-baifeng/sub2api`，不会读取原作者的 Release 缓存。
`custom.1 → custom.2` 会被识别为更新；发布更高基础版本时，基础版本优先比较。
