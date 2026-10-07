# 定制版更新操作手册：合并 → 推送 → 编译发布 → 1Panel 更新

适用于本仓库的 `custom` 分支，以及已通过 1Panel Docker 编排安装的应用。PostgreSQL、Redis 沿用现有部署。

流程：**检查工作区 → 同步自己的远端 → 合并上游 → 处理冲突 → 验证 → 提交并推送分支 → 检查 CI → 推送新标签 → 等待 Release 成功 → 备份并更新服务器 → 验证业务。**

本文命令分为本地 Windows PowerShell 和服务器 Linux Shell，按小节执行，每步成功再继续。编辑文档本身、推送分支或更新 VERSION 文件，都不会自动更新服务器。

## 1. 仓库和版本规则

| 项目 | 固定值或规则 |
| --- | --- |
| 本地项目 | `E:\Project\Go\server\sub2api` |
| 定制分支 | `custom` |
| origin（自己的仓库） | `https://github.com/Drunkard-baifeng/sub2api.git` |
| upstream（上游） | `https://github.com/Wei-Shaw/sub2api.git` |
| 合并来源 | `upstream/main` |
| Git 标签 | `v<上游基础版本>-custom.<序号>` |
| VERSION 文件和镜像标签 | 去掉 Git 标签开头的 `v` |
| Docker 镜像 | `ghcr.io/drunkard-baifeng/sub2api:<镜像标签>` |

`v0.2.14-custom.1` 是本次已使用的版本，后续不要重复创建。同一基础版本的下次发布可以使用 `v0.2.14-custom.2`；合并新基础版本后，例如使用 `v0.2.15-custom.1`。先查上游实际版本和已有标签，再确定编号。

## 2. 同步和备份本地分支

本地 PowerShell：

```powershell
Set-Location E:\Project\Go\server\sub2api
git status --short --branch
```

如有未提交改动，先检查并提交，或用 `git stash push -u -m "before upstream update"` 暂存。后续恢复用 `git stash apply`，核对完成再删除对应 stash。不要用强制重置覆盖自己的改动。

工作区干净后：

```powershell
git switch custom
git pull --ff-only origin custom
git fetch upstream --prune
git fetch origin --tags
git remote -v
git log --oneline HEAD..upstream/main
git diff --stat HEAD...upstream/main
git show upstream/main:backend/cmd/server/VERSION
git tag --list 'v*-custom.*' --sort=-version:refname
```

`pull --ff-only` 如果提示分叉，先检查双方提交，不要直接强推。Release 可能回写版本号，所以合并上游前要先同步 `origin/custom`。

保存合并前的本地指针：

```powershell
$backupBranch = 'codex/backup-custom-' + (Get-Date -Format 'yyyyMMdd-HHmmss')
git branch $backupBranch
```

## 3. 合并上游并处理冲突

```powershell
git merge --no-ff --no-commit upstream/main
git status
git diff --name-only --diff-filter=U
```

即使自动合并成功，也应检查改动。不要对全部文件一律选择 ours/theirs：上游修复和定制功能都需要保留。

重点核对：

- 首页、登录注册、青绿色主题、品牌 Logo、右上角钱包。
- 充值、兑换、支付方式布局和后台购买 CDK 链接配置。
- `.github/workflows/release.yml`、`.goreleaser.yaml`、更新服务及 `deploy/install.sh` 中自己的仓库来源。
- `backend/cmd/server/VERSION` 和 `frontend/pnpm-lock.yaml`。

### 版本号冲突

选择本次准备发布的**新版本**。以下 `0.2.14-custom.2` 仅作下次同基础版本发布的示例：

```powershell
$releaseVersion = '0.2.14-custom.2' # 按本次上游版本、已有发布记录修改
$versionPath = Join-Path (Get-Location) 'backend/cmd/server/VERSION'
[IO.File]::WriteAllText($versionPath, $releaseVersion + "`n", [Text.UTF8Encoding]::new($false))
git add backend/cmd/server/VERSION
```

### 前端依赖冲突

先合并 `package.json` 中需要的依赖和安全覆盖规则，再处理锁文件。当前 CI 使用 pnpm 9，本机 pnpm 12 还会读取 `pnpm-workspace.yaml` 的 overrides，两处规则要同步。

不要直接删除锁文件升级所有依赖。保留定制依赖版本，移除冲突标记后，在前端目录用 CI 对应版本核验：

```powershell
Push-Location frontend
npm.cmd exec --yes --package=pnpm@9.15.9 -- pnpm install --lockfile-only --no-frozen-lockfile
npm.cmd exec --yes --package=pnpm@9.15.9 -- pnpm install --frozen-lockfile
Pop-Location
git diff -- frontend/package.json frontend/pnpm-workspace.yaml frontend/pnpm-lock.yaml
git add frontend/package.json frontend/pnpm-workspace.yaml frontend/pnpm-lock.yaml
```

不要同时运行不同版本 pnpm 的安装、构建或测试命令：它们可能重建同一个 `node_modules`。依赖源较慢时，可给本次安装命令增加 `--registry=https://registry.npmjs.org`，无需修改全局配置。

其他冲突文件逐个处理、逐个 `git add <文件路径>`。确认下列命令没有未解决文件或空白错误：

```powershell
git diff --name-only --diff-filter=U
git diff --cached --check
git diff --check
```

若尚未提交且决定放弃本次合并，执行 `git merge --abort`。提交后该命令不再适用，备份分支仍可用于对比。

## 4. 本地验证

先完成安装，再使用该目录的工具执行检查，避免全局 pnpm 12 自动调整 pnpm 9 刚安装的目录：

```powershell
Push-Location frontend
& ./node_modules/.bin/eslint.cmd . --ext .vue,.js,.jsx,.cjs,.mjs,.ts,.tsx,.cts,.mts
& ./node_modules/.bin/vue-tsc.cmd -b
& ./node_modules/.bin/vitest.cmd run
& ./node_modules/.bin/vite.cmd build
Pop-Location
```

分别确认命令成功；测试失败时检查原因，不要只看最后一个命令的退出码。前端构建产物会写入 `backend/internal/web/dist`，供后端嵌入。

后端命令必须在 `backend`（Go 模块目录）执行：

```powershell
Push-Location backend
go version
go test -tags=unit ./...
go build -tags embed -trimpath -o "$env:TEMP/sub2api-update-check.exe" ./cmd/server
Pop-Location
```

Go 版本以 `backend/go.mod` 为准。数据库、Redis、Docker 相关集成测试和 Linux lint 以 CI 对应任务为准；本地缺少 Docker时不要直接访问生产数据库代替测试。

本次遇到的环境问题，供后续定位：

- Windows 配置测试可能读到 `E:\app\data\config.yaml`，干扰默认值断言。应在不读取现有部署配置的隔离目录运行测试，而不是修改实际配置。
- 测试提示找不到 `sh`/`bash` 时，可在当前终端临时加上 Git Bash：`$env:PATH = 'D:\install\program\Git\bin;' + $env:PATH`。
- Python 发布测试遇到 GBK 解码错误时，当前终端设 `$env:PYTHONUTF8 = '1'`，再执行 `python -m unittest discover -s .github/release-tools -p 'test_release_matrix.py'`。
- 内存或时序测试偶发失败时，记录失败项并隔离复查；单独重跑通过不等于全量首次通过。

日常继续使用本机 pnpm 12 时，全部 pnpm 9 检查结束后再执行一次 `pnpm --dir frontend install --frozen-lockfile`，等待结束再启动开发服务。

## 5. 提交和推送源码

回到项目根目录，检查暂存区只包含准备发布的文件，不要提交本地密码、配置或临时日志：

```powershell
git status --short
git diff --cached --stat
git diff --cached --check
git commit -m 'Merge upstream/main v0.2.14 into custom' # 修改为本次实际基础版本
git push origin custom
git status --short --branch
```

自动合并的文件已暂存，手工处理的文件要先 `git add`。如果只是自己的后续修改，没有合并上游，则使用描述实际修改的提交消息。

打开 [GitHub Actions](https://github.com/Drunkard-baifeng/sub2api/actions)，检查本次提交的 CI。**分支推送用于同步源码和触发 CI；版本标签推送才触发 Release。** 两个工作流是独立的，Release 不会自动等待另一条 CI，发布前应先确认 CI 的检查结果。若没有运行记录，检查 Fork 仓库是否在 Actions 中启用了工作流。

## 6. 打新标签，触发 GitHub 编译发布

GitHub 编译由 Actions 完成，Git 本身只负责版本管理。采用完整发布模式，同时构建多平台程序和 Docker 镜像；仓库变量 `SIMPLE_RELEASE` 保持未设置或 `false`。

本地 PowerShell，项目根目录：

```powershell
git switch custom
git pull --ff-only origin custom
git status --short
$releaseVersion = (Get-Content backend/cmd/server/VERSION -Raw).Trim()
$releaseTag = 'v' + $releaseVersion
git tag --list $releaseTag
git ls-remote --tags origin "refs/tags/$releaseTag"
```

确认工作区干净、版本正确且本地和远端都不存在同名标签。若标签已经存在，先检查对应 Release，不要移动或覆盖已发布标签。

```powershell
git tag -a $releaseTag -m "release: $releaseTag" -m '同步上游修复，保留定制界面、支付与兑换功能。'
git push origin $releaseTag
```

在 [Actions](https://github.com/Drunkard-baifeng/sub2api/actions) 找到对应标签的 **Release**，等待成功。完整发布包含五平台二进制、校验文件及 Linux amd64/arm64 镜像。

如果需要先试编译，在 **Release → Run workflow** 选择 `custom`，tag/ref 填 `custom`，勾选 `dry_run`、不勾选 `simple_release`；试跑成功不代表正式发布完成。

正式发布后检查：

1. 对应 Release 工作流成功，源码提交与预期一致。
2. [Releases](https://github.com/Drunkard-baifeng/sub2api/releases) 出现目标版本及安装包。
3. [GHCR 镜像包](https://github.com/Drunkard-baifeng/sub2api/pkgs/container/sub2api) 出现去掉 `v` 的标签，服务器能拉取。

构建失败时先看失败步骤；若要修代码，使用新提交和新版本号。只有原提交不变、确认为临时网络等问题时才考虑重跑，并检查已有部分发布产物。

## 7. 1Panel 更新现有应用

以下操作在服务器进行。升级前用 1Panel 备份应用数据库，保存应用数据目录（包括 `config.yaml`）、编排和环境变量，记录当前镜像标签。备份放在受限目录，不要提交 Git。

在 **容器 → 编排 → 当前 Sub2API 编排** 找到应用的 `image:`。仅替换镜像版本，例如：

```yaml
# 示例；替换为本次已成功发布的实际版本
image: ghcr.io/drunkard-baifeng/sub2api:0.2.14-custom.2
```

若原编排通过环境变量选择镜像，修改它实际引用的变量；不要额外增加一个未被引用的新变量。

保留原来的数据库/Redis连接、Docker 网络、端口、环境变量、JWT/TOTP 密钥，以及 `/app/data` 数据挂载。原部署记录的挂载为 `/opt/sub2api-docker/data:/app/data`，以服务器现有编排为准。端口和 Nginx 反代沿用已经正常工作的配置。

保存编排，拉取新镜像，然后重新部署/重建**应用容器**（面板按钮名称随版本不同）。仅“重启”旧容器不会更换镜像。此次不需要重新安装 PostgreSQL、Redis或填写首次安装向导。

### 可选：服务器命令行更新

先在 1Panel 确认实际编排文件路径、项目名和服务名，不要把数据目录当作编排目录。下面的占位符必须替换，`sub2api` 是示例服务名：

```bash
docker compose -p <现有项目名> -f <编排文件绝对路径> pull sub2api
docker compose -p <现有项目名> -f <编排文件绝对路径> up -d --no-deps sub2api
docker compose -p <现有项目名> -f <编排文件绝对路径> ps
docker compose -p <现有项目名> -f <编排文件绝对路径> logs --tail=100 sub2api
```

若现有编排需要 `--env-file` 或多个 `-f`，继续使用原参数。保持项目名和配置来源一致，避免创建另一套容器。

## 8. 验证、回滚与收尾

在 1Panel 查看应用容器使用的镜像、运行状态和启动日志。也可在服务器终端执行（容器名以实际值为准）：

```bash
docker inspect sub2api --format '{{.Config.Image}}'
docker logs --since 10m --tail 200 sub2api
curl -fsS https://api.jisuai.net/health
```

健康接口预期返回 `{"status":"ok"}`。它只证明服务存活，还需检查后台版本号、登录、余额、充值/兑换页面，以及一次正常模型请求的流式输出。支付回调修复发布后，结合一笔受控支付核对订单状态和余额是否一致。

如更新失败，先保存应用日志并核对迁移情况：

- 数据库仍兼容旧版本：将编排镜像改回之前记录的固定标签，再拉取并重建应用容器。
- 已执行不兼容数据库迁移：按该版本迁移说明和升级前备份恢复数据库、配置及对应旧镜像。恢复备份会丢失备份之后的数据，先停止写入并确认恢复范围。

不要用删除数据卷、重建数据库或重复安装解决升级问题。旧标签与备份应保留到新版本验证稳定之后。

发布完成后，本地同步工作流可能回写的版本号：

```powershell
git switch custom
git pull --ff-only origin custom
git status --short --branch
```

可记录本次的 Git 标签、提交 SHA、上游 SHA、镜像标签、备份时间及验证结果，方便下次对比。完整发布配置、GHCR 首次设置和非 Docker 安装见 [CUSTOM_RELEASE.md](CUSTOM_RELEASE.md)。
