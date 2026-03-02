# 照片归档备份工具技术设计（v1.2）

## 1. 技术栈

- Go 1.22+
- GUI（当前原型）: Fyne
- GUI（目标方案）: SwiftUI（macOS 原生）
- DB: SQLite
- 识别引擎: ONNX Runtime（本地），可扩展云 Provider

## 2. 系统架构

采用分层架构：
- 表现层：Fyne 原型 GUI + CLI（当前）；SwiftUI（目标）
- 应用层：任务编排（scan/plan/import/verify/report/recognition）
- 领域层：文件归档、去重、校验、标签管理
- 基础设施层：SQLite、文件系统、日志、模型推理

## 3. 项目结构（当前）

```text
photo-archiver/
  cmd/
    app/                   # Fyne GUI 入口
    cli/                   # CLI 入口（migrate/import）
  internal/
    app/                   # 应用服务（use cases）
    config/                # 配置读取与校验
    core/
      importer/            # 导入编排
      planner/             # 路径与任务计划
      verifier/            # 校验策略
    media/
      exifmeta/            # EXIF 时间提取
    storage/
      sqlite/              # repository 实现
    gui/                   # Fyne 页面
  docs/
```

## 4. 当前实现状态（已完成）

### 4.1 CLI
- `migrate`：初始化/升级 SQLite 表结构（`docs/DB_SCHEMA.sql`）。
- `import`：执行导入，支持 `--dry-run`、`--verify size|hash`。
- `service`：启动本地 HTTP API（默认 `127.0.0.1:38080`）。

### 4.2 导入核心流程
- 已实现 `scan -> plan -> copy -> verify -> persist` 全链路。
- 已实现文件类型白名单过滤。
- 已实现重名冲突自动追加序号。
- 已实现 SHA256 去重（基于 DB 指纹）。
- 已实现校验（size/hash）。

### 4.3 时间归档规则
- 已实现 EXIF 优先：优先读取 EXIF 拍摄时间归档。
- 回退策略：若 EXIF 缺失或解析失败，回退到文件修改时间（mtime）。

### 4.4 数据持久化
- 已实现任务表、文件表、任务条目表读写。
- 已实现任务进度更新与完成状态回写。
- 已实现任务列表查询（用于 GUI 任务记录页）。
- 已实现失败项查询与重试前置数据读取。
- 已实现任务删除（非 running）与任务取消状态支持（`cancelled`）。

### 4.5 GUI 原型（Fyne）
- 已实现目录选择（点击选择来源/目标目录）。
- 已实现导入页、任务记录页、设置页。
- 已实现状态摘要、进度提示、任务表格展示。

## 5. 核心流程

### 5.1 导入流程
1. 读取配置与 GUI 参数。
2. 扫描来源并过滤支持格式。
3. 生成导入计划（目标路径、重命名、去重结果）。
4. 用户确认后进入执行阶段。
5. 并发复制文件。
6. 执行校验（size/hash）。
7. 持久化任务和文件结果。
8. 输出报告并触发识别任务（可选）。

### 5.2 识别流程
1. 导入成功文件写入识别待处理队列。
2. worker 拉取队列并调用识别 Provider。
3. 过滤低置信度标签并入库。
4. 更新识别任务状态，通知 GUI 刷新。

## 6. 模块设计

### 5.1 Importer
- 输入：`ImportRequest`
- 输出：`ImportJobResult`
- 责任：编排扫描、计划、复制、校验、落库。

### 5.2 Planner
- 输入：源文件列表 + 模板 + 目标目录。
- 输出：`[]PlannedItem`。
- 责任：计算目标路径、重名冲突处理、元数据补齐。

### 6.3 Dedup
- 当前实现：`HashCheck(sha256)`。
- 规划：补充 `FastCheck(size, mtime, name)`，支持 hybrid。

### 6.4 Verifier
- 接口：`Verify(src, dst, mode)`
- mode: `size | hash`

### 6.5 Recognition
- 接口：`Recognize(imagePath) ([]TagScore, error)`
- Provider：`local_onnx`、`cloud_xxx`（后续）

## 7. 应用接口（当前与目标）

当前（CLI/Fyne 与本地 API）：
- `importer.Run(req, store) -> ImportResult`
- `store.ListImportJobs(limit) -> []ImportJobSummary`
- `GET /health`
- `POST /v1/scan`
- `POST /v1/import`
- `GET /v1/jobs?limit=30`
- `GET /v1/jobs/{id}`
- `POST /v1/jobs/{id}/retry-failed`
- `POST /v1/jobs/{id}/cancel`
- `DELETE /v1/jobs/{id}`

目标（SwiftUI 通过本地 API 调用）：
- `POST /v1/scan`
- `POST /v1/import`
- `GET /v1/jobs?limit=30`
- `GET /v1/jobs/{id}`
- `POST /v1/jobs/{id}/retry-failed`
- `POST /v1/jobs/{id}/cancel`
- `DELETE /v1/jobs/{id}`

## 8. 配置设计

配置优先级：CLI 参数 > GUI 本次输入 > 配置文件默认值。

关键配置：
- `backup_root`
- `backup2_root`
- `folder_template`
- `dedup_mode`
- `verify_mode`
- `workers`
- `io_rate_limit_mb`
- `recognition.enabled`
- `recognition.confidence_threshold`

## 9. 并发与容错

- 当前实现：串行复制。
- 规划：复制 worker 池（默认 4，可配置）。
- 单文件失败记录后继续处理其他文件。
- 任务状态机：`pending -> running -> success|partial_failed|failed|cancelled`。
- 进程中断后可通过数据库状态重试未完成项。

## 10. 日志与监控

- 日志格式：JSON 行日志。
- 关键字段：job_id、file_id、source_path、target_path、error_code。
- 报告输出：成功/跳过/失败数、总字节、耗时、吞吐。

## 11. 安全与隐私

- 默认本地离线处理，不上传图片。
- 删除源文件开关默认关闭。
- 开启云识别时需要显式告知用户数据外发。

## 12. UI 技术路线更新

- 当前 Fyne 版本用于功能验证与流程联调。
- 为匹配 macOS 设计语言，后续 UI 切换到 SwiftUI。
- Go 侧新增 `cmd/service` 本地 API 服务层，SwiftUI 作为前端壳。
- 迁移原则：保留现有 `internal/core` 与 `internal/storage`，避免重写业务逻辑。

## 13. 近期交互增强（SwiftUI）

- 导入页支持异步任务 + 轮询，避免大任务请求超时。
- 导入页新增进度条、ETA、停止任务按钮。
- 导入页增加启动宽限：任务提交到真正开始执行期间不报错。
- 任务页新增删除任务按钮（带确认）与中止运行中任务按钮。
- 导入参数草稿与当前任务状态采用全局 Store，跨页面切换不丢失。
