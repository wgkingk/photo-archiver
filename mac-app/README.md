# PhotoArchiverMac (SwiftUI)

该目录用于放置 macOS 原生 SwiftUI 前端工程。

当前已提供 `PhotoArchiverMac.xcodeproj`（包含 Bundle Identifier），
可避免 `Cannot index window tabs due to missing main bundle identifier` 提示。
并已内置后端进程启动管理（端口冲突自动尝试 38080-38090，启动后健康检查）。

## 推荐结构

```text
mac-app/
  PhotoArchiverMac/
    App/
    Features/
      Import/
      Jobs/
      Settings/
    Networking/
    Models/
```

## API 基址

- `http://127.0.0.1:38080`

## 首批对接接口

- `POST /v1/scan`
- `POST /v1/import`
- `GET /v1/jobs?limit=30`
- `GET /v1/jobs/{id}`
- `POST /v1/jobs/{id}/retry-failed`
- `POST /v1/jobs/{id}/cancel`
- `DELETE /v1/jobs/{id}`

## 关键交互（当前）

- 导入页：支持目录选择、异步导入、进度条、ETA、停止任务。
- 导入任务跨页面不丢失（切换页面后继续轮询与展示）。
- 任务页：支持刷新、重试失败项、删除任务、中止运行中任务。
- 设置页：可查看后端状态（地址/端口/PID）并执行重启/停止。

## 运行前端

1. 启动后端服务：

```bash
go run ../cmd/service --addr 127.0.0.1:38080 --db ../data/photo_archiver.db --schema ../docs/DB_SCHEMA.sql
```

2. 在 `mac-app/` 目录执行：

```bash
open PhotoArchiverMac.xcodeproj
```

3. Xcode 中选择 `PhotoArchiverMac` scheme，目标选 `My Mac`，点击 Run。

## 重新生成 xcodeproj（可选）

如果后续新增文件，可通过 xcodegen 重建工程：

```bash
brew install xcodegen
xcodegen generate
```

## 一键打包完整 App

在仓库根目录执行：

```bash
./scripts/build_macos_app.sh
```

产物位置：`dist/PhotoArchiverMac.app`
