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
	db, err = gorm.Open(sqlite.Open(conf.DBPath), &gorm.Config{})
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	db.AutoMigrate(&TrafficRecord{}, &SubSnapshot{})
}