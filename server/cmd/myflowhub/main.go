package main

import (
	"fmt"
	"myflowhub/pkg/config"
	"myflowhub/pkg/database"
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

	var server *hub.Server

	if config.AppConfig.Relay.Enabled {
		// 作为中继启动
		relayConf := config.AppConfig.Relay
		log.Info().Msg("以中继模式启动...")
		server = hub.NewServer(relayConf.ParentAddr, relayConf.ListenAddr, relayConf.HardwareID)
	} else {
		// 作为中枢启动
		serverConf := config.AppConfig.Server
		log.Info().Msg("以中枢模式启动...")
		server = hub.NewServer("", serverConf.ListenAddr, serverConf.HardwareID)
	}

	server.Start() // 阻塞式启动
}
