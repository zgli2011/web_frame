# 示例程序说明

本目录包含 Web Frame 框架的使用示例。

## 文件说明

### 配置文件
- `config.yaml` - 完整配置示例，包含所有功能模块配置
- `logger_only.yaml` - 仅日志配置的简化版本

### 演示程序
- `standalone_demo.go` - **推荐** 独立日志演示，展示最核心的简化日志功能
- `main.go` - 完整功能演示（需要数据库连接）

## 快速体验

### 1. 运行日志演示（推荐）
```bash
go run example/standalone_demo.go
```

展示超简单的日志使用方式：
- `logger.Info("消息")` - 直接调用，无需实例化
- 支持 trace_id 链路追踪
- 支持结构化日志
- 同时输出到控制台和文件

### 2. 运行完整演示
```bash
go run example/main.go
```

展示框架的所有功能（需要配置数据库连接）

## 核心使用方式

```go
import "web_frame/logger"

// 框架初始化后，直接使用
logger.Info("应用启动")
logger.InfoF("端口: %d", 8080)
logger.Error("连接失败")
```

就是这么简单！