# SwiftUI 迁移落地说明

## 目标

- 保留 Go 核心导入能力。
- 通过本地 HTTP 服务解耦 UI 与业务。
- 在 macOS 端使用 SwiftUI 构建原生界面。

## 已落地内容

- 新增本地服务入口：`cmd/service/main.go`
- 新增 HTTP API：`internal/api/httpapi/server.go`
- 新增异步导入能力（可指定 job_id）：`internal/core/importer/importer.go`
- 已实现失败项重试接口：`POST /v1/jobs/{id}/retry-failed`
- 已实现任务取消接口：`POST /v1/jobs/{id}/cancel`
- 已实现任务删除接口：`DELETE /v1/jobs/{id}`
- 新增 SwiftUI 对接骨架：`mac-app/PhotoArchiverMac/`

## 当前 API

- `GET /health`
- `POST /v1/scan`
- `POST /v1/import`
- `GET /v1/jobs?limit=30`
- `GET /v1/jobs/{id}`
- `POST /v1/jobs/{id}/retry-failed`
- `POST /v1/jobs/{id}/cancel`
- `DELETE /v1/jobs/{id}`

## SwiftUI 对接建议

1. 使用 `NavigationSplitView` 实现左导航。
2. 首屏对接 `POST /v1/scan` 和 `POST /v1/import`。
3. 任务页对接 `GET /v1/jobs` 与 `GET /v1/jobs/{id}`。
4. 失败项通过 `POST /v1/jobs/{id}/retry-failed` 重试。
5. 对运行中任务做轮询刷新。

## 当前 SwiftUI 骨架

- `App/ContentView.swift`：侧栏导航与页面切换
- `Features/Import/ImportView.swift`：扫描与导入（含目录选择器）
- `Features/Import/ImportView.swift`：扫描与导入（含目录选择器、进度条、ETA、停止任务）
- `Features/Jobs/JobsView.swift`：任务列表、详情、重试失败项、删除任务、中止任务（含运行中轮询与状态标签）
- `Networking/APIClient.swift`：本地 API 调用
- `Models/APIModels.swift`：接口模型
- `App/ImportDraftStore.swift`：导入参数草稿状态
- `App/ImportTaskStore.swift`：导入任务状态与轮询生命周期

## 后端内置与启动策略（已落地）

- `App/BackendProcessManager.swift` 在应用启动时拉起内置后端。
- 后端端口冲突自动尝试 `38080-38090`。
- 启动后通过 `/health` 健康检查，成功后再开放业务页面。
- 导入任务轮询增加启动宽限，避免“已提交但尚未建档”阶段出现误报错误。

## 一键打包

- 脚本：`scripts/build_macos_app.sh`
- 输出：`dist/PhotoArchiverMac.app`

## 服务运行

```bash
go run ./cmd/service --addr 127.0.0.1:38080 --db ./data/photo_archiver.db --schema ./docs/DB_SCHEMA.sql
```
