package client

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLConfig struct {
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	Username        string `yaml:"username"`
	Password        string `yaml:"password"`
	Database        string `yaml:"database"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	MaxOpenConns    int    `yaml:"max_open_conns"`
	ConnMaxLifetime int    `yaml:"conn_max_lifetime"`
}

type MySQLClient struct {
	db *sql.DB
}

var (
	mysqlClients map[string]*MySQLClient
	mysqlMutex   sync.RWMutex
)

func init() {
	mysqlClients = make(map[string]*MySQLClient)
}

func InitMySQL(name string, config MySQLConfig) error {
	mysqlMutex.Lock()
	defer mysqlMutex.Unlock()

	if _, exists := mysqlClients[name]; exists {
		return fmt.Errorf("MySQL client '%s' already exists", name)
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open MySQL connection for '%s': %v", name, err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping MySQL database for '%s': %v", name, err)
	}

	setConnectionDefaults := func(maxIdle, maxOpen, lifetime int) {
		if maxIdle > 0 {
			db.SetMaxIdleConns(maxIdle)
		} else {
			db.SetMaxIdleConns(10)
		}

		if maxOpen > 0 {
			db.SetMaxOpenConns(maxOpen)
		} else {
			db.SetMaxOpenConns(100)
		}

		if lifetime > 0 {
			db.SetConnMaxLifetime(time.Duration(lifetime) * time.Second)
		} else {
			db.SetConnMaxLifetime(time.Hour)
		}
	}

	setConnectionDefaults(config.MaxIdleConns, config.MaxOpenConns, config.ConnMaxLifetime)

	mysqlClients[name] = &MySQLClient{db: db}
	return nil
}

func GetMySQL(name string) (*MySQLClient, error) {
	mysqlMutex.RLock()
	defer mysqlMutex.RUnlock()

	client, exists := mysqlClients[name]
	if !exists {
		return nil, fmt.Errorf("MySQL client '%s' not found", name)
	}

	return client, nil
}

func GetDefaultMySQL() (*MySQLClient, error) {
	return GetMySQL("default")
}

func (c *MySQLClient) DB() *sql.DB {
	return c.db
}

func (c *MySQLClient) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return c.db.Query(query, args...)
}

func (c *MySQLClient) QueryRow(query string, args ...interface{}) *sql.Row {
	return c.db.QueryRow(query, args...)
}

func (c *MySQLClient) Exec(query string, args ...interface{}) (sql.Result, error) {
	return c.db.Exec(query, args...)
}

func (c *MySQLClient) Begin() (*sql.Tx, error) {
	return c.db.Begin()
}

func (c *MySQLClient) Close() error {
	return c.db.Close()
}

func CloseAllMySQL() {
	mysqlMutex.Lock()
	defer mysqlMutex.Unlock()

	for name, client := range mysqlClients {
		if err := client.Close(); err != nil {
			fmt.Printf("Error closing MySQL client '%s': %v\n", name, err)
		}
	}

	mysqlClients = make(map[string]*MySQLClient)
}