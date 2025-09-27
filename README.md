# Web Frame - Go Web 项目公共框架

一个 Go 语言 web 项目的公共框架，提供配置文件解析、MySQL 数据库连接池、Redis 连接池和日志管理功能。

## 功能特性

1. **配置文件解析**：支持 YAML 格式配置文件，可通过点号语法（如 `app.check.port`）访问配置项
2. **MySQL 连接池**：支持多个 MySQL 数据库连接池管理
3. **Redis 连接池**：支持多个 Redis 连接池管理
4. **日志系统**：支持多级别日志、文件/控制台输出、JSON/Text 格式、自动轮转、trace_id 追踪

## 快速开始

### 1. 引入框架

```go
import "web_frame/framework"
```

### 2. 初始化框架

```go
// 传入配置文件路径初始化框架
if err := framework.Init("config.yaml"); err != nil {
    log.Fatal("Failed to initialize framework:", err)
}

// 程序结束时关闭所有连接
defer framework.Close()
```

### 3. 配置文件示例

```yaml
app:
  name: "my_web_app"
  port: 8080

mysql:
  default:
    host: "localhost"
    port: 3306
    username: "root"
    password: "password"
    database: "test_db"
    max_idle_conns: 10
    max_open_conns: 100
    conn_max_lifetime: 3600

  user_db:
    host: "localhost"
    port: 3306
    username: "user"
    password: "user_pass"
    database: "user_database"

redis:
  default:
    host: "localhost"
    port: 6379
    password: ""
    db: 0
    pool_size: 10

  cache:
    host: "localhost"
    port: 6379
    db: 1

log:
  level: "INFO"                 # 日志级别：DEBUG, INFO, WARN, ERROR, FATAL
  output: "both"                # 输出方式：console, file, both
  format: "json"                # 格式：json, text
  file_path: "./logs"           # 日志文件目录
  file_name: "app.log"          # 日志文件名
  max_size: 100                 # 单个日志文件最大大小 (MB)
  max_backups: 7                # 备份文件数量
```

### 4. 使用配置

```go
// 获取配置值
appName, err := framework.GetConfigString("app.name")
port, err := framework.GetConfigInt("app.port")
```

### 5. 使用 MySQL

```go
// 获取默认 MySQL 连接
defaultDB, err := framework.GetDefaultMySQL()
if err != nil {
    log.Fatal(err)
}

// 获取指定名称的 MySQL 连接
userDB, err := framework.GetMySQL("user_db")
if err != nil {
    log.Fatal(err)
}

// 执行查询
rows, err := defaultDB.Query("SELECT * FROM users")
```

### 6. 使用 Redis

```go
ctx := context.Background()

// 获取默认 Redis 连接
defaultRedis, err := framework.GetDefaultRedis()
if err != nil {
    log.Fatal(err)
}

// 获取指定名称的 Redis 连接
cacheRedis, err := framework.GetRedis("cache")
if err != nil {
    log.Fatal(err)
}

// 设置键值
err = defaultRedis.Set(ctx, "key", "value", time.Minute).Err()

// 获取值
value, err := defaultRedis.Get(ctx, "key").Result()
```

### 7. 使用日志

框架初始化后会自动设置全局日志实例，可以直接使用，无需获取实例：

```go
import "web_frame/logger"

// 最简单的用法 - 直接调用全局方法
logger.Info("Application started")
logger.InfoF("Server listening on port %d", 8080)
logger.Error("Database connection failed")

// 带字段的日志
logger.Warn("High memory usage detected",
    logger.Field{Key: "memory_usage", Value: "85%"},
    logger.Field{Key: "threshold", Value: "80%"},
)

// 创建带预设字段的 logger
loggerWithFields := logger.WithFields(map[string]interface{}{
    "component": "auth",
    "version":   "1.0.0",
})
// 注意：WithFields 返回的 logger 仍需要 context
if loggerWithFields != nil {
    loggerWithFields.Info(context.Background(), "User authentication successful")
}

// 如果需要 trace_id，使用带 Ctx 后缀的方法
import "context"
ctx := logger.WithTraceID(context.Background(), "request-123")
logger.InfoCtx(ctx, "Processing user request with trace ID")
logger.WarnFCtx(ctx, "User %s login failed", "john_doe")
```

#### 日志方法说明

**简化方法（推荐）：**
- `logger.Debug(msg, ...fields)` - 调试日志
- `logger.Info(msg, ...fields)` - 信息日志
- `logger.Warn(msg, ...fields)` - 警告日志
- `logger.Error(msg, ...fields)` - 错误日志
- `logger.Fatal(msg, ...fields)` - 致命错误日志
- `logger.DebugF(format, ...args)` - 格式化调试日志
- `logger.InfoF(format, ...args)` - 格式化信息日志
- 其他格式化方法：`WarnF`, `ErrorF`, `FatalF`

**带 Context 的方法（需要 trace_id 时使用）：**
- `logger.InfoCtx(ctx, msg, ...fields)`
- `logger.InfoFCtx(ctx, format, ...args)`
- 其他方法：`DebugCtx`, `WarnCtx`, `ErrorCtx`, `FatalCtx` 和对应的格式化版本

#### 日志输出格式

**Text 格式示例：**
```
[2006-01-02 15:04:05.123] [INFO] [12345] [trace-abc123] [main.go:25] Application started
```

**JSON 格式示例：**
```json
{
  "time": "2006-01-02 15:04:05.123",
  "level": "INFO",
  "pid": 12345,
  "trace_id": "trace-abc123",
  "file": "main.go:25",
  "message": "Application started",
  "ip": "192.168.1.100",
  "fields": {
    "component": "auth",
    "version": "1.0.0"
  }
}
```

## 运行示例

```bash
# 进入示例目录
cd example

# 运行示例程序
go run main.go
```

## 目录结构

```
web_frame/
├── config/          # 配置文件解析模块
│   ├── config.go
│   └── parser.go
├── client/          # 数据库客户端模块
│   ├── mysql.go
│   └── redis.go
├── logger/          # 日志模块
│   ├── types.go
│   ├── utils.go
│   ├── formatter.go
│   ├── writer.go
│   └── logger.go
├── framework/       # 框架核心模块
│   └── app.go
├── example/         # 示例项目
│   ├── config.yaml
│   └── main.go
├── go.mod
└── README.md
```

## 配置项说明

### MySQL 配置

- `host`: 数据库主机地址
- `port`: 数据库端口
- `username`: 用户名
- `password`: 密码
- `database`: 数据库名
- `max_idle_conns`: 最大空闲连接数（默认：10）
- `max_open_conns`: 最大打开连接数（默认：100）
- `conn_max_lifetime`: 连接最大生命周期，单位秒（默认：3600）

### Redis 配置

- `host`: Redis 主机地址
- `port`: Redis 端口
- `password`: Redis 密码
- `db`: 数据库编号
- `pool_size`: 连接池大小（默认：10）
- `min_idle_conns`: 最小空闲连接数（默认：2）
- `max_conn_age`: 连接最大生命周期，单位秒（默认：3600）
- `pool_timeout`: 连接池获取连接超时时间，单位秒（默认：4）
- `idle_timeout`: 空闲连接超时时间，单位秒（默认：300）
- `idle_check_freq`: 空闲连接检查频率，单位秒（默认：60）

### 日志配置

- `level`: 日志级别，可选值：DEBUG, INFO, WARN, ERROR, FATAL（默认：INFO）
- `output`: 输出方式，可选值：console（控制台）, file（文件）, both（两者）（默认：console）
- `format`: 输出格式，可选值：text（文本）, json（JSON）（默认：text）
- `file_path`: 日志文件存储目录（默认：./logs）
- `file_name`: 日志文件名称（默认：app.log）
- `max_size`: 单个日志文件最大大小，单位 MB（默认：100）
- `max_backups`: 保留的备份日志文件数量（默认：7）

## 特性说明

### 日志系统特性

1. **多级别日志**：支持 DEBUG、INFO、WARN、ERROR、FATAL 五个级别
2. **灵活输出**：支持控制台、文件或同时输出到两个地方
3. **多种格式**：支持 Text 和 JSON 两种输出格式
4. **文件轮转**：支持基于文件大小的自动轮转和备份数量限制
5. **trace_id 追踪**：
   - 自动从 context 中获取 trace_id
   - 如果 context 中没有 trace_id，会自动生成一个
   - 同一个请求上下文中的所有日志都使用相同的 trace_id
6. **结构化日志**：支持添加自定义字段，便于日志分析和检索
7. **线程安全**：支持并发环境下的安全日志记录
8. **自动信息收集**：自动收集进程ID、文件位置、机器IP等信息

### trace_id 使用建议

在 HTTP 请求处理中，建议在请求开始时设置 trace_id：

```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    // 从请求头获取或生成 trace_id
    traceID := r.Header.Get("X-Trace-ID")
    if traceID == "" {
        traceID = generateTraceID() // 自定义生成函数
    }

    // 将 trace_id 设置到 context 中
    ctx := logger.WithTraceID(r.Context(), traceID)

    // 使用带 trace_id 的 context 进行日志记录
    logger := framework.GetLogger()
    logger.Info(ctx, "Request started")

    // ... 业务逻辑处理

    logger.Info(ctx, "Request completed")
}
```

## 许可证

MIT License