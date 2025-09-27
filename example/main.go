package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"web_frame/check"
	"web_frame/framework"
	"web_frame/logger"
)

func main() {
	fmt.Println("🚀 Web Frame Demo - 展示框架的所有功能")
	fmt.Println("====================================================")

	// 确保程序结束时正确关闭所有连接
	defer framework.Close()

	// 初始化框架 - 一次初始化，全部可用
	if err := framework.Init("example/config.yaml"); err != nil {
		log.Fatal("❌ 框架初始化失败:", err)
	}

	fmt.Println("✅ 框架初始化成功！")

	// 获取检测端口配置
	checkPort, err := framework.GetConfigInt("app.check_port")
	if err != nil {
		checkPort = 9090 // 默认端口
		fmt.Printf("⚠️  未找到check_port配置，使用默认端口: %d\n", checkPort)
	}

	// 获取端点配置
	healthEnabled, _ := framework.GetConfigBool("app.endpoints.health")
	apisEnabled, _ := framework.GetConfigBool("app.endpoints.apis")
	metricsEnabled, _ := framework.GetConfigBool("app.endpoints.metrics")

	// 创建端点配置
	endpointConfig := &check.EndpointConfig{
		HealthEnabled:  healthEnabled,
		APIsEnabled:    apisEnabled,
		MetricsEnabled: metricsEnabled,
	}

	// 启动HTTP服务器
	srv := check.NewServerWithConfig(checkPort, endpointConfig)

	// 在独立的goroutine中启动服务器
	go func() {
		fmt.Println("\n🌐 === HTTP服务器启动 ===")
		if err := srv.Start(); err != nil {
			log.Printf("❌ 服务器启动失败: %v", err)
		}
	}()

	// 按顺序演示各个功能模块
	demoLogger()
	demoConfig()
	// demoMySQL() // 注释掉，因为需要实际的数据库连接
	// demoRedis() // 注释掉，因为需要实际的 Redis 连接

	fmt.Println("\n🎉 所有演示完成！框架已准备就绪！")
	fmt.Println("📊 可用的API端点:")
	if healthEnabled {
		fmt.Printf("   - http://localhost:%d/health  (健康检查)\n", checkPort)
	}
	if apisEnabled {
		fmt.Printf("   - http://localhost:%d/apis    (API列表)\n", checkPort)
	}
	if metricsEnabled {
		fmt.Printf("   - http://localhost:%d/metrics (Prometheus指标)\n", checkPort)
	}
	if !healthEnabled && !apisEnabled && !metricsEnabled {
		fmt.Println("   - 无端点启用")
	}
	fmt.Println("🔥 按 Ctrl+C 退出...")

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n🛑 正在关闭服务器...")

	// 优雅关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Stop(ctx); err != nil {
		log.Printf("❌ 服务器强制关闭: %v", err)
	} else {
		fmt.Println("✅ 服务器已优雅关闭")
	}
}

// 演示日志功能 - 最重要的功能，放在第一位
func demoLogger() {
	fmt.Println("\n📝 === 日志系统演示 ===")

	// 🌟 最简单的用法 - 这是重点！
	logger.Info("应用程序启动 - 超简单的日志记录！")
	logger.InfoF("服务器监听端口: %d", 8080)

	// 不同级别的日志
	logger.Debug("这是调试信息 (可能不显示，取决于日志级别设置)")
	logger.Warn("这是警告信息")
	logger.Error("这是错误信息")

	// 结构化日志 - 带字段
	logger.Info("用户登录成功",
		logger.Field{Key: "user_id", Value: 12345},
		logger.Field{Key: "username", Value: "john_doe"},
		logger.Field{Key: "ip", Value: "192.168.1.100"},
	)

	// 格式化日志
	logger.WarnF("内存使用率: %d%%, 阈值: %d%%", 85, 80)
	logger.ErrorF("数据库连接失败: %s", "连接超时")

	// 🎯 trace_id 演示 - 用于请求追踪
	ctx := logger.WithTraceID(context.Background(), "req-abc123")
	logger.InfoCtx(ctx, "开始处理用户请求")
	logger.WarnFCtx(ctx, "用户 %s 尝试未授权访问", "hacker")
	logger.InfoCtx(ctx, "请求处理完成")

	// 🔧 预设字段的 logger - 用于组件日志
	dbLogger := logger.WithFields(map[string]interface{}{
		"component": "database",
		"version":   "1.0.0",
	})

	if dbLogger != nil {
		dbLogger.Info(context.Background(), "数据库连接建立")
		dbLogger.Warn(context.Background(), "连接池资源不足")
	}

	fmt.Println("✅ 日志演示完成 - 检查控制台输出和 ./logs/app.log 文件")
}

// 演示配置系统
func demoConfig() {
	fmt.Println("\n⚙️ === 配置系统演示 ===")

	// 获取应用配置
	if appName, err := framework.GetConfigString("app.name"); err == nil {
		fmt.Printf("应用名称: %s\n", appName)
	}

	if version, err := framework.GetConfigString("app.version"); err == nil {
		fmt.Printf("应用版本: %s\n", version)
	}

	if port, err := framework.GetConfigInt("app.port"); err == nil {
		fmt.Printf("应用端口: %d\n", port)
	}

	// 获取服务器配置
	if host, err := framework.GetConfigString("server.host"); err == nil {
		fmt.Printf("服务器地址: %s\n", host)
	}

	if timeout, err := framework.GetConfigInt("server.timeout"); err == nil {
		fmt.Printf("服务器超时: %d秒\n", timeout)
	}

	fmt.Println("✅ 配置系统演示完成")
}

// MySQL 演示 (需要实际数据库连接)
func demoMySQL() {
	fmt.Println("\n🗄️ === MySQL 数据库演示 ===")

	// 获取默认数据库连接
	defaultDB, err := framework.GetDefaultMySQL()
	if err != nil {
		fmt.Printf("⚠️  获取默认MySQL连接失败: %v\n", err)
		return
	}

	// 测试连接
	if err := defaultDB.DB().Ping(); err != nil {
		fmt.Printf("⚠️  MySQL连接测试失败: %v\n", err)
		return
	}

	fmt.Println("✅ 默认MySQL连接正常")

	// 获取指定名称的数据库连接
	userDB, err := framework.GetMySQL("user_db")
	if err != nil {
		fmt.Printf("⚠️  获取用户数据库连接失败: %v\n", err)
		return
	}

	if err := userDB.DB().Ping(); err != nil {
		fmt.Printf("⚠️  用户数据库连接测试失败: %v\n", err)
		return
	}

	fmt.Println("✅ 用户数据库连接正常")

	// 执行简单查询
	row := defaultDB.QueryRow("SELECT 'Hello from MySQL' as message")
	var message string
	if err := row.Scan(&message); err != nil {
		fmt.Printf("⚠️  查询执行失败: %v\n", err)
	} else {
		fmt.Printf("📄 查询结果: %s\n", message)
	}
}

// Redis 演示 (需要实际 Redis 连接)
func demoRedis() {
	fmt.Println("\n🔴 === Redis 缓存演示 ===")

	ctx := context.Background()

	// 获取默认 Redis 连接
	defaultRedis, err := framework.GetDefaultRedis()
	if err != nil {
		fmt.Printf("⚠️  获取默认Redis连接失败: %v\n", err)
		return
	}

	// 设置键值
	key := "demo:test"
	value := "Hello from Redis!"
	if err := defaultRedis.Set(ctx, key, value, time.Minute).Err(); err != nil {
		fmt.Printf("⚠️  Redis SET操作失败: %v\n", err)
		return
	}

	fmt.Printf("✅ Redis SET成功: %s = %s\n", key, value)

	// 获取值
	result, err := defaultRedis.Get(ctx, key).Result()
	if err != nil {
		fmt.Printf("⚠️  Redis GET操作失败: %v\n", err)
		return
	}

	fmt.Printf("📄 Redis GET结果: %s\n", result)

	// 使用缓存 Redis
	cacheRedis, err := framework.GetRedis("cache")
	if err != nil {
		fmt.Printf("⚠️  获取缓存Redis连接失败: %v\n", err)
		return
	}

	// Hash 操作
	hashKey := "user:123"
	if err := cacheRedis.HSet(ctx, hashKey, "name", "Alice", "age", 25).Err(); err != nil {
		fmt.Printf("⚠️  Redis HSET操作失败: %v\n", err)
		return
	}

	userData, err := cacheRedis.HGetAll(ctx, hashKey).Result()
	if err != nil {
		fmt.Printf("⚠️  Redis HGETALL操作失败: %v\n", err)
		return
	}

	fmt.Printf("📄 用户数据: %+v\n", userData)
	fmt.Println("✅ Redis演示完成")
}
