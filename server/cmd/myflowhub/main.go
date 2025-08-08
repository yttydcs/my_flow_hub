package main

import (
	"fmt"
	"myflowhub/server/internal/config"
	"myflowhub/server/internal/database"
	"myflowhub/server/internal/hub"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	config.LoadConfig()

	dbConf := config.AppConfig.Database
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		dbConf.Host, dbConf.User, dbConf.Password, dbConf.DBName, dbConf.Port)

	postgresDsn := fmt.Sprintf("host=%s user=%s password=%s dbname=postgres port=%s sslmode=disable",
		dbConf.Host, dbConf.User, dbConf.Password, dbConf.Port)

	database.InitDatabase(dsn, postgresDsn, dbConf.DBName)

	// 启动中枢
	serverConf := config.AppConfig.Server
	hubSrv := hub.NewServer("", serverConf.ListenAddr, serverConf.HardwareID)
	go hubSrv.Start()

	// 根据配置决定是否启动中继
	if config.AppConfig.Relay.Enabled {
		relayConf := config.AppConfig.Relay
		relay := hub.NewServer(relayConf.ParentAddr, relayConf.ListenAddr, relayConf.HardwareID)
		go relay.Start()
	}

	select {}
}
