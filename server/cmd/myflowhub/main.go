package main

import (
	"fmt"
	"myflowhub/pkg/config"
	"myflowhub/pkg/database"
	"myflowhub/server/internal/controller"
	"myflowhub/server/internal/hub"
	"myflowhub/server/internal/repository"
	"myflowhub/server/internal/service"
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

	// 初始化 repository
	deviceRepo := repository.NewDeviceRepository(database.DB)
	variableRepo := repository.NewVariableRepository(database.DB)
	userRepo := repository.NewUserRepository(database.DB)
	permRepo := repository.NewPermissionRepository(database.DB)
	keyRepo := repository.NewKeyRepository(database.DB)

	// 初始化 service
	deviceService := service.NewDeviceService(deviceRepo, variableRepo, database.DB)
	variableService := service.NewVariableService(variableRepo)
	authService := service.NewAuthService(deviceRepo, variableRepo)
	permService := service.NewPermissionService(deviceRepo)
	userService := service.NewUserService(userRepo)
	keyService := service.NewKeyService(keyRepo, permRepo, deviceRepo)
	authzService := service.NewAuthzService(keyService, deviceRepo, permRepo)

	// 初始化 controller
	deviceController := controller.NewDeviceController(deviceService, permService)
	variableController := controller.NewVariableController(variableService, deviceService, permService)
	authController := controller.NewAuthController(authService, deviceService)
	// inject optional services
	// 仅 key 模式：不再注入 SessionService；login 将返回 key
	authController.SetPermissionRepository(permRepo)
	authController.SetKeyService(keyService)
	authController.SetUserRepository(userRepo)
	_ = permService // reserved for future auth controller checks
	userController := controller.NewUserController(userService, permService, permRepo)
	keyController := controller.NewKeyController(keyService)
	// 将统一授权服务注入设备与变量控制器
	deviceController.SetAuthzService(authzService)
	variableController.SetAuthzService(authzService)
	userController.SetAuthzService(authzService)

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

	// 注册路由
	server.RegisterRoute("query_nodes", deviceController.HandleQueryNodes)
	server.RegisterRoute("create_device", deviceController.HandleCreateDevice)
	server.RegisterRoute("update_device", deviceController.HandleUpdateDevice)
	server.RegisterRoute("delete_device", deviceController.HandleDeleteDevice)
	server.RegisterRoute("query_variables", variableController.HandleQueryVariables)
	server.RegisterRoute("vars_query", variableController.HandleVarsQuery)
	server.RegisterRoute("var_update", variableController.HandleVarUpdate)
	server.RegisterRoute("var_delete", variableController.HandleVarDelete)
	// 用户管理（仅管理员）
	server.RegisterRoute("user_list", userController.HandleUserList)
	server.RegisterRoute("user_create", userController.HandleUserCreate)
	server.RegisterRoute("user_update", userController.HandleUserUpdate)
	server.RegisterRoute("user_delete", userController.HandleUserDelete)
	// 用户权限管理（仅管理员）
	server.RegisterRoute("user_perm_list", userController.HandleUserPermList)
	server.RegisterRoute("user_perm_add", userController.HandleUserPermAdd)
	server.RegisterRoute("user_perm_remove", userController.HandleUserPermRemove)
	// 密钥管理（按用户权限范围）
	server.RegisterRoute("key_list", keyController.HandleKeyList)
	server.RegisterRoute("key_create", keyController.HandleKeyCreate)
	server.RegisterRoute("key_update", keyController.HandleKeyUpdate)
	server.RegisterRoute("key_delete", keyController.HandleKeyDelete)
	// 密钥创建辅助：可见设备列表
	server.RegisterRoute("key_devices", keyController.HandleKeyDevices)

	// 启动前：按策略初始化默认管理员
	seedDefaultAdmin(userService, permRepo)
	server.RegisterRoute("auth_request", authController.HandleAuthRequest)
	server.RegisterRoute("manager_auth", authController.HandleManagerAuthRequest)
	server.RegisterRoute("register_request", authController.HandleRegisterRequest)
	server.RegisterRoute("user_login", authController.HandleUserLogin)

	server.Start() // 阻塞式启动
}

func seedDefaultAdmin(userSvc *service.UserService, permRepo *repository.PermissionRepository) {
	username := config.AppConfig.Server.DefaultAdmin.Username
	password := config.AppConfig.Server.DefaultAdmin.Password
	if username == "" || password == "" { // 提供安全默认值需用户修改
		username = "admin"
		password = "admin123!" // 建议首次登录后立即修改
	}
	// 根据用户要求：仅在本次启动新建数据库或用户表时自动创建/赋权；
	// 若用户表已存在，则不做任何操作（不创建 admin，亦不赋权）。
	if !(database.WasUserTableCreated || database.WasDBCreated) {
		log.Info().Msg("检测到用户表已存在，跳过默认管理员创建与赋权")
		return
	}

	// 全新安装场景：创建或修复默认管理员权限
	if u, err := userSvc.GetByUsername(username); err == nil {
		_ = permRepo.AddUserNode(u.ID, "admin.manage", nil)
		_ = permRepo.AddUserNode(u.ID, "**", nil)
		return
	}
	if u, err := userSvc.Create(username, "System Administrator", password); err == nil {
		log.Info().Str("username", username).Msg("默认管理员已创建（新建用户表/数据库）")
		_ = permRepo.AddUserNode(u.ID, "admin.manage", nil)
		_ = permRepo.AddUserNode(u.ID, "**", nil)
	}
}
