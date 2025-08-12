package database

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// DB 封装了数据库连接
var DB *gorm.DB

// createDBIfNotExist 检查数据库是否存在，如果不存在则创建。
// 返回一个布尔值，指示是否创建了新数据库。
func createDBIfNotExist(postgresDsn, dbName string) bool {
	tempDB, err := gorm.Open(postgres.Open(postgresDsn), &gorm.Config{})
	if err != nil {
		log.Fatal().Err(err).Msg("无法连接到 'postgres' 数据库以进行设置")
	}
	defer func() {
		sqlDB, _ := tempDB.DB()
		sqlDB.Close()
	}()

	var exists bool
	query := "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = ?)"
	if err := tempDB.Raw(query, dbName).Scan(&exists).Error; err != nil {
		log.Fatal().Err(err).Msg("检查数据库是否存在时失败")
	}

	if !exists {
		log.Info().Str("database", dbName).Msg("数据库不存在，正在创建...")
		createCmd := fmt.Sprintf("CREATE DATABASE %s", dbName)
		if err := tempDB.Exec(createCmd).Error; err != nil {
			log.Fatal().Err(err).Msg("创建数据库失败")
		}
		log.Info().Str("database", dbName).Msg("数据库创建成功")
		return true // 数据库被创建
	}

	log.Info().Str("database", dbName).Msg("数据库已存在，跳过创建")
	return false // 数据库已存在
}

// InitDatabase 初始化数据库连接并运行迁移
func InitDatabase(dsn, postgresDsn, dbName string) {
	wasCreated := createDBIfNotExist(postgresDsn, dbName)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal().Err(err).Msg("无法连接到目标数据库")
	}

	log.Info().Msg("数据库连接成功")

	// 每次启动都运行自动迁移，确保新模型就绪
	log.Info().Msg("正在运行数据库迁移...")
	err = DB.AutoMigrate(&Device{}, &DeviceVariable{}, &AccessPermission{}, &User{}, &Permission{}, &Key{}, &Grant{}, &AuditLog{})
	if err != nil {
		log.Fatal().Err(err).Msg("数据库迁移失败")
	}
	if wasCreated {
		log.Info().Msg("数据库首次创建并迁移完成")
	} else {
		log.Info().Msg("数据库迁移完成（已存在数据库）")
	}
}
