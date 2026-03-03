# photo-archiver

一个使用 Go 开发的照片归档与备份工具仓库。

## AI 开发说明

本项目从需求整理、架构设计、代码实现、UI 开发、打包发布到文档沉淀，**全程由 AI（Codex）协作完成**。

- 需求与设计：PRD、技术设计、迁移方案、开发日志
- 后端能力：Go 导入核心、SQLite、API 服务
- 前端能力：SwiftUI 页面、任务管理、状态面板
- 交付流程：一键打包与 GitHub Release 制品

## 运行界面

### 运行中

![运行中](docs/images/running.png)

### 导入页

![导入页](docs/images/import.png)

### 目录选择

![目录选择](docs/images/select_dic.png)

### 任务页

![任务页](docs/images/task.png)

### 设置页

![设置页](docs/images/setting.png)

## 目录说明

- `docs/`: 需求、设计与开发文档
- `cmd/`: 可执行程序入口（后续）
- `internal/`: 业务实现（后续）

## 文档

- `docs/PRD.md`: 产品需求文档（范围、用例、验收标准）
- `docs/TECH_DESIGN.md`: 技术设计文档（含当前实现状态与 SwiftUI 迁移路线）
- `docs/DB_SCHEMA.sql`: SQLite 表结构初始化脚本
- `docs/SWIFTUI_MIGRATION.md`: SwiftUI 迁移落地说明
- `docs/PROJECT_JOURNEY.md`: 项目动机、设计与开发过程总结
- `docs/JOURNAL.md`: 项目日志与运行界面记录（含截图）
- `docs/prd-design.md`: 早期草案（已由上述文档细化）
- `mac-app/`: SwiftUI 原生前端骨架

## 快速开始

推荐普通用户使用“直接下载运行”，**无需命令行**。

1. 打开 Releases 页面下载最新版本：

- https://github.com/wgkingk/photo-archiver/releases

2. 下载并解压 `PhotoArchiverMac-*.zip`，双击打开 `PhotoArchiverMac.app`。

> 如果首次打开出现 macOS 安全提示（“无法验证开发者”），请按以下方式处理：
>
> - 在 Finder 中对 `PhotoArchiverMac.app` 点击右键 -> “打开”
> - 在弹窗中再次点击“打开”
> - 或前往“系统设置 -> 隐私与安全性”，允许该应用后再打开

3. 首次启动时，进入“导入”页：

- 选择来源目录（内存卡挂载目录）
- 选择目标目录（备份目录）
- 点击“开始导入”

4. 在“任务”页查看进度、失败重试、删除或中止任务；在“设置”页查看后端状态并可重启。

## 开发者运行（可选）

如果你是开发者或需要本地调试，可以使用以下命令：

1. 初始化数据库：

```bash
go run ./cmd/cli migrate --db ./data/photo_archiver.db --schema ./docs/DB_SCHEMA.sql
```

2. 执行导入（先 dry-run）：

```bash
go run ./cmd/cli import --source /path/to/source --dest /path/to/backup --db ./data/photo_archiver.db --dry-run
```

3. 正式导入并校验：

```bash
go run ./cmd/cli import --source /path/to/source --dest /path/to/backup --db ./data/photo_archiver.db --verify hash
```

4. 启动 GUI：

```bash
go run ./cmd/app --db ./data/photo_archiver.db --schema ./docs/DB_SCHEMA.sql
```

5. 启动本地 API 服务（供 SwiftUI 对接）：

```bash
go run ./cmd/service --addr 127.0.0.1:38080 --db ./data/photo_archiver.db --schema ./docs/DB_SCHEMA.sql
```

6. 启动 SwiftUI 前端：

```bash
cd ./mac-app && open PhotoArchiverMac.xcodeproj
```

7. 一键打包完整 macOS App（内含后端）：

```bash
./scripts/build_macos_app.sh
```
