// 数据库模型定义与初始化

package main

import (
	"log"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

type TrafficRecord struct {
	ID        uint      `gorm:"primaryKey"`
	Timestamp time.Time `gorm:"index"`
	DeviceID  string
	NodeName  string
	UpDelta   int64
	DownDelta int64
	IsProxy   bool
	ActiveConns int
}

type SubSnapshot struct {
	ID     uint      `gorm:"primaryKey"`
	Date   time.Time `gorm:"index"`
	SubUrl string
	Used   int64
	Total  int64
	Expire int64
}

func initDB() {
	var err error

	// 高并发读写优化：启用 WAL 模式 (Write-Ahead Logging) 并设置 busy_timeout
	dsn := conf.DBPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)"

	db, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	// 获取底层的 sql.DB 进行连接池配置
	sqlDB, err := db.DB()
	if err == nil {
		// 设置最大打开的连接数，推荐 SQLite WAL 模式下可以适当增大，但不宜过多避免锁竞争
		sqlDB.SetMaxOpenConns(1) // SQLite 最安全的并发写入配置是强串行写入（1）+ WAL 并发读
		// 但由于 WAL 模式支持多读一写，Go 内置的 sql 包也能很好的处理 sqlite 的并发，我们可以稍微增大或者直接不设限制使用默认
		// 结合 CGO / Pure Go SQLite 的差异，glebarez/sqlite (pure go) 在设为 1 时写入最稳
		// 结合业务，这里写入是高频小包，建议设置为 1 强制串行写入，避免 "database is locked"
		sqlDB.SetMaxOpenConns(1)
	}

	db.AutoMigrate(&TrafficRecord{}, &SubSnapshot{})
}