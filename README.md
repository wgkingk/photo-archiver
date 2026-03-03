# photo-archiver

一个使用 Go 开发的照片归档与备份工具仓库。

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

## 下一步

1. 初始化 Go 模块并搭建基础工程结构。
2. 根据 `docs/TECH_DESIGN.md` 先实现 import 核心流程。
3. 根据 `docs/DB_SCHEMA.sql` 完成 migration 与 repository 层。

## 快速开始

1. 执行数据库初始化：

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

6. 启动 SwiftUI 前端（Xcode 打开 Package）：

```bash
cd ./mac-app && open PhotoArchiverMac.xcodeproj
```

7. 一键打包完整 macOS App（内含后端）：

```bash
./scripts/build_macos_app.sh
```
