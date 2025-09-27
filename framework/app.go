package framework

import (
	"fmt"
	"reflect"

	"web_frame/client"
	"web_frame/config"
	"web_frame/logger"
)

type App struct {
	configPath string
	initialized bool
	logger     logger.Logger
}

type DatabaseConfig struct {
	MySQL map[string]client.MySQLConfig `yaml:"mysql"`
	Redis map[string]client.RedisConfig `yaml:"redis"`
}

func New() *App {
	return &App{
		initialized: false,
	}
}

func (app *App) Init(configPath string) error {
	if app.initialized {
		return fmt.Errorf("app already initialized")
	}

	app.configPath = configPath

	if err := config.Init(configPath); err != nil {
		return fmt.Errorf("failed to initialize config: %v", err)
	}

	if err := app.initDatabases(); err != nil {
		return fmt.Errorf("failed to initialize databases: %v", err)
	}

	if err := app.initLogger(); err != nil {
		return fmt.Errorf("failed to initialize logger: %v", err)
	}

	app.initialized = true
	return nil
}

func (app *App) initDatabases() error {
	mysqlConfigs, err := app.getMySQLConfigs()
	if err != nil {
		return fmt.Errorf("failed to get MySQL configs: %v", err)
	}

	for name, cfg := range mysqlConfigs {
		if err := client.InitMySQL(name, cfg); err != nil {
			return fmt.Errorf("failed to initialize MySQL '%s': %v", name, err)
		}
	}

	redisConfigs, err := app.getRedisConfigs()
	if err != nil {
		return fmt.Errorf("failed to get Redis configs: %v", err)
	}

	for name, cfg := range redisConfigs {
		if err := client.InitRedis(name, cfg); err != nil {
			return fmt.Errorf("failed to initialize Redis '%s': %v", name, err)
		}
	}

	return nil
}

func (app *App) initLogger() error {
	loggerConfig, err := app.getLoggerConfig()
	if err != nil {
		return fmt.Errorf("failed to get logger config: %v", err)
	}

	app.logger, err = logger.NewLogger(loggerConfig)
	if err != nil {
		return fmt.Errorf("failed to create logger: %v", err)
	}

	logger.SetGlobalLogger(app.logger)

	return nil
}

func (app *App) getLoggerConfig() (logger.Config, error) {
	loggerData, err := config.Get("log")
	if err != nil {
		return logger.Config{
			Level:      "INFO",
			Output:     "console",
			Format:     "text",
			FilePath:   "./logs",
			FileName:   "app.log",
			MaxSize:    100,
			MaxBackups: 7,
		}, nil
	}

	_, ok := loggerData.(map[string]interface{})
	if !ok {
		return logger.Config{}, fmt.Errorf("log config is not a map")
	}

	var cfg logger.Config
	if err := app.mapToStruct(loggerData, &cfg); err != nil {
		return logger.Config{}, fmt.Errorf("failed to parse logger config: %v", err)
	}

	if cfg.Level == "" {
		cfg.Level = "INFO"
	}
	if cfg.Output == "" {
		cfg.Output = "console"
	}
	if cfg.Format == "" {
		cfg.Format = "text"
	}
	if cfg.FilePath == "" {
		cfg.FilePath = "./logs"
	}
	if cfg.FileName == "" {
		cfg.FileName = "app.log"
	}
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = 100
	}
	if cfg.MaxBackups <= 0 {
		cfg.MaxBackups = 7
	}

	return cfg, nil
}

func (app *App) getMySQLConfigs() (map[string]client.MySQLConfig, error) {
	mysqlData, err := config.Get("mysql")
	if err != nil {
		return map[string]client.MySQLConfig{}, nil
	}

	mysqlMap, ok := mysqlData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("mysql config is not a map")
	}

	configs := make(map[string]client.MySQLConfig)
	for name, configData := range mysqlMap {
		var cfg client.MySQLConfig
		if err := app.mapToStruct(configData, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse MySQL config for '%s': %v", name, err)
		}
		configs[name] = cfg
	}

	return configs, nil
}

func (app *App) getRedisConfigs() (map[string]client.RedisConfig, error) {
	redisData, err := config.Get("redis")
	if err != nil {
		return map[string]client.RedisConfig{}, nil
	}

	redisMap, ok := redisData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("redis config is not a map")
	}

	configs := make(map[string]client.RedisConfig)
	for name, configData := range redisMap {
		var cfg client.RedisConfig
		if err := app.mapToStruct(configData, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse Redis config for '%s': %v", name, err)
		}
		configs[name] = cfg
	}

	return configs, nil
}

func (app *App) mapToStruct(data interface{}, result interface{}) error {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("data is not a map")
	}

	resultValue := reflect.ValueOf(result).Elem()
	resultType := resultValue.Type()

	for i := 0; i < resultValue.NumField(); i++ {
		field := resultValue.Field(i)
		fieldType := resultType.Field(i)

		yamlTag := fieldType.Tag.Get("yaml")
		if yamlTag == "" {
			continue
		}

		if value, exists := dataMap[yamlTag]; exists && field.CanSet() {
			switch field.Kind() {
			case reflect.String:
				if str, ok := value.(string); ok {
					field.SetString(str)
				}
			case reflect.Int:
				if intVal, ok := value.(int); ok {
					field.SetInt(int64(intVal))
				} else if floatVal, ok := value.(float64); ok {
					field.SetInt(int64(floatVal))
				}
			}
		}
	}

	return nil
}

func (app *App) GetConfig(key string) (interface{}, error) {
	if !app.initialized {
		return nil, fmt.Errorf("app not initialized")
	}
	return config.Get(key)
}

func (app *App) GetConfigString(key string) (string, error) {
	if !app.initialized {
		return "", fmt.Errorf("app not initialized")
	}
	return config.GetString(key)
}

func (app *App) GetConfigInt(key string) (int, error) {
	if !app.initialized {
		return 0, fmt.Errorf("app not initialized")
	}
	return config.GetInt(key)
}

func (app *App) GetConfigBool(key string) (bool, error) {
	if !app.initialized {
		return false, fmt.Errorf("app not initialized")
	}
	return config.GetBool(key)
}

func (app *App) GetMySQL(name string) (*client.MySQLClient, error) {
	if !app.initialized {
		return nil, fmt.Errorf("app not initialized")
	}
	return client.GetMySQL(name)
}

func (app *App) GetDefaultMySQL() (*client.MySQLClient, error) {
	return app.GetMySQL("default")
}

func (app *App) GetRedis(name string) (*client.RedisClient, error) {
	if !app.initialized {
		return nil, fmt.Errorf("app not initialized")
	}
	return client.GetRedis(name)
}

func (app *App) GetDefaultRedis() (*client.RedisClient, error) {
	return app.GetRedis("default")
}

func (app *App) GetLogger() logger.Logger {
	return app.logger
}

func (app *App) Close() {
	if app.initialized {
		client.CloseAllMySQL()
		client.CloseAllRedis()
		app.initialized = false
	}
}

var globalApp *App

func Init(configPath string) error {
	if globalApp != nil {
		return fmt.Errorf("framework already initialized")
	}

	globalApp = New()
	return globalApp.Init(configPath)
}

func GetConfig(key string) (interface{}, error) {
	if globalApp == nil {
		return nil, fmt.Errorf("framework not initialized")
	}
	return globalApp.GetConfig(key)
}

func GetConfigString(key string) (string, error) {
	if globalApp == nil {
		return "", fmt.Errorf("framework not initialized")
	}
	return globalApp.GetConfigString(key)
}

func GetConfigInt(key string) (int, error) {
	if globalApp == nil {
		return 0, fmt.Errorf("framework not initialized")
	}
	return globalApp.GetConfigInt(key)
}

func GetConfigBool(key string) (bool, error) {
	if globalApp == nil {
		return false, fmt.Errorf("framework not initialized")
	}
	return globalApp.GetConfigBool(key)
}

func GetMySQL(name string) (*client.MySQLClient, error) {
	if globalApp == nil {
		return nil, fmt.Errorf("framework not initialized")
	}
	return globalApp.GetMySQL(name)
}

func GetDefaultMySQL() (*client.MySQLClient, error) {
	if globalApp == nil {
		return nil, fmt.Errorf("framework not initialized")
	}
	return globalApp.GetDefaultMySQL()
}

func GetRedis(name string) (*client.RedisClient, error) {
	if globalApp == nil {
		return nil, fmt.Errorf("framework not initialized")
	}
	return globalApp.GetRedis(name)
}

func GetDefaultRedis() (*client.RedisClient, error) {
	if globalApp == nil {
		return nil, fmt.Errorf("framework not initialized")
	}
	return globalApp.GetDefaultRedis()
}

func GetLogger() logger.Logger {
	if globalApp == nil {
		return nil
	}
	return globalApp.GetLogger()
}

func Close() {
	if globalApp != nil {
		globalApp.Close()
		globalApp = nil
	}
}