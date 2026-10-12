<div align="center">

<img src="frontend/public/jisu-ai-logo.svg" alt="定制版 Logo" width="88" />

# Sub2API · 个人自用定制版

基于上游项目，按自己的使用需求持续调整。

</div>

## 项目说明

**这是个人开发、测试和自用部署的 Fork，不是上游官方版本。**

项目基于 [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api) 开发，保留原项目的核心能力，在此基础上调整界面、功能入口和部署流程。感谢原作者及所有贡献者。

- **上游项目**：[Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api)。
- **本仓库**：[Drunkard-baifeng/sub2api](https://github.com/Drunkard-baifeng/sub2api)。
- **维护分支**：直接在 `main` 开发，并按需合并 `upstream/main` 的更新。

本仓库的修改与发布由维护者自行维护，不代表上游立场；功能、界面及默认配置可能与上游不同。

## 主要调整

- 首页、登录和注册页面：青绿色毛玻璃风格，支持浅色与深色模式。
- 钱包、充值与兑换页面：优化余额展示、支付方式和布局，支持配置购买 CDK 的链接。
- 工具入口：增加「极速画布」，支持后台设置显示开关与跳转地址。
- 发布流程：保留多平台程序与 Docker 镜像构建，使用本 Fork 的独立发布渠道。

## 部署与更新

- [发布配置与安装说明](deploy/CUSTOM_RELEASE.md)
- [日常更新：合并上游 → 验证 → 推送 → 发布 → 1Panel 更新](deploy/CUSTOM_UPDATE.md)
- [本仓库发布包](https://github.com/Drunkard-baifeng/sub2api/releases)

Docker 镜像：

```text
ghcr.io/drunkard-baifeng/sub2api:<版本号>
```

部署时选择本仓库已发布的具体版本。版本号中的 `-custom.N` 仅用于区分定制发布，日常维护分支仍是 `main`。

通用功能和配置可参考[上游文档](https://github.com/Wei-Shaw/sub2api/blob/main/README_CN.md)；本定制版的安装、发布和更新以本仓库文档为准。

## 使用说明

本仓库以个人使用需求为主，不作为上游官方支持渠道。使用时请遵守所在地法律法规及相关服务商条款，自行评估部署和使用风险。

## 许可证与致谢

保留上游的 [GNU Lesser General Public License v3.0](LICENSE)（或更高版本）及原作者署名。个人自用的维护定位不改变原有许可证。

Copyright (c) 2026 Wesley Liddick

感谢 [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api) 及其贡献者。
