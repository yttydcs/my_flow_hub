package config

import (
	"encoding/json"
	"os"

	"github.com/rs/zerolog/log"
)

// Config 结构体定义了整个应用的配置
type Config struct {
	Database struct {
		Host     string `json:"Host"`
		User     string `json:"User"`
		Password string `json:"Password"`
		DBName   string `json:"DBName"`
		Port     string `json:"Port"`
	} `json:"Database"`
	Server struct {
		ListenAddr   string `json:"ListenAddr"`
		HardwareID   string `json:"HardwareID"`
		ManagerToken string `json:"ManagerToken"`
		RelayToken   string `json:"RelayToken"`
		DefaultAdmin struct {
			Username string `json:"Username"`
			Password string `json:"Password"`
		} `json:"DefaultAdmin"`
	} `json:"Server"`
	Hub struct {
		Address string `json:"Address"`
	} `json:"Hub"`
	Relay struct {
		Enabled     bool   `json:"Enabled"`
		ParentAddr  string `json:"ParentAddr"`
		ListenAddr  string `json:"ListenAddr"`
		HardwareID  string `json:"HardwareID"`
		SharedToken string `json:"SharedToken"`
	} `json:"Relay"`
}

// AppConfig 是全局配置实例
var AppConfig Config

// LoadConfig 从 config.json 文件加载配置
func LoadConfig() {
	file, err := os.Open("config.json")
	if err != nil {
		log.Fatal().Err(err).Msg("无法打开配置文件")
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&AppConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("无法解析配置文件")
	}

	log.Info().Msg("配置加载成功")
}
