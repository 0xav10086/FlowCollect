// 配置管理。负责加载/监听 ServerSetting.ini

package main

import (
	"log"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/ini.v1"
)

// 配置结构体
type ServerConfig struct {
	ListenPort  string
	ServerToken string
	DBPath      string
	SMTPHost    string
	SMTPPort    string
	EmailUser   string
	EmailPass   string
	EmailTo     string
	SubUrls     []string
}

var (
	conf     ServerConfig
	confLock sync.RWMutex
	iniPath  = "ServerSetting.ini"
)

func loadConfig() error {
	cfg, err := ini.Load(iniPath)
	if err != nil {
		return err
	}

	confLock.Lock()
	defer confLock.Unlock()

	section := cfg.Section("server")
	smtpSec := cfg.Section("smtp")

	conf = ServerConfig{
		ListenPort:  section.Key("ListenPort").MustString(":8686"),
		ServerToken: section.Key("ServerToken").MustString("YourSecretToken"),
		DBPath:      section.Key("DBPath").MustString("traffic.db"),
		SMTPHost:    smtpSec.Key("SMTPHost").MustString("smtp.qq.com"),
		SMTPPort:    smtpSec.Key("SMTPPort").MustString("587"),
		EmailUser:   smtpSec.Key("EmailUser").MustString(""),
		EmailPass:   smtpSec.Key("EmailPass").MustString(""),
		EmailTo:     smtpSec.Key("EmailTo").MustString(""),
		SubUrls:     section.Key("SubUrls").Strings(","),
	}

	log.Printf("[%s] 服务端配置已更新", time.Now().Format("15:04:05"))
	return nil
}

func watchConfig() {
	watcher, _ := fsnotify.NewWatcher()
	defer watcher.Close()
	_ = watcher.Add(iniPath)
	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				loadConfig()
			}
		}
	}
}