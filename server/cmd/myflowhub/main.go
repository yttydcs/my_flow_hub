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
	auditRepo := repository.NewAuditLogRepository(database.DB)
	systemLogRepo := repository.NewSystemLogRepository(database.DB)

	// 初始化 service
	deviceService := service.NewDeviceService(deviceRepo, variableRepo, database.DB)
	variableService := service.NewVariableService(variableRepo)
	authService := service.NewAuthService(deviceRepo, variableRepo)
	permService := service.NewPermissionService(deviceRepo)
	userService := service.NewUserService(userRepo)
	keyService := service.NewKeyService(keyRepo, permRepo, deviceRepo)
	auditService := service.NewAuditService(auditRepo, keyService)
	systemLogService := service.NewSystemLogService(systemLogRepo)
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
	logController := controller.NewLogController(auditService)
	systemLogController := controller.NewSystemLogController(systemLogService)
	// 将统一授权服务注入设备与变量控制器
	deviceController.SetAuthzService(authzService)
	deviceController.SetSystemLogService(systemLogService)
	variableController.SetAuthzService(authzService)
	userController.SetAuthzService(authzService)
	userController.SetAuditService(auditService)
	logController.SetAuthzService(authzService)
	systemLogController.SetAuthzService(authzService)
	keyController.SetAuditService(auditService)
	authController.SetAuditService(auditService)
	authController.SetSystemLogService(systemLogService)

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

	// 注入系统日志服务到 hub（用于连接/断开等事件记录）
	server.Syslog = systemLogService

	// 启动前：按策略初始化默认管理员
	seedDefaultAdmin(userService, permRepo)

	// 创建各域的 Bin 适配器实例
	ab := &controller.AuthBin{C: authController}
	db := &controller.DeviceBin{C: deviceController}
	vb := &controller.VariableBin{C: variableController}
	kb := &controller.KeyBin{C: keyController}
	slb := &controller.SystemLogBin{C: systemLogController}
	ub := &controller.UserBin{Users: userController}
	pb := &controller.ParentAuthBin{C: controller.NewParentAuthController()}

	// 在 hub 包内注册 TypeID，传入具体处理器以避免循环依赖
	hub.RegisterAuthRoutes(server, ab.ManagerAuth, ab.UserLogin, ab.UserMe, ab.UserLogout)
	hub.RegisterSystemLogRoutes(server, slb.List)
	hub.RegisterDeviceRoutes(server, db.QueryNodes, db.Create, db.Update, db.Delete)
	hub.RegisterVariableRoutes(server, vb.Update, vb.Delete, vb.List)
	hub.RegisterKeyRoutes(server, kb.List, kb.Create, kb.Update, kb.Delete)
	hub.RegisterKeyDevicesRoute(server, kb.Devices)
	hub.RegisterUserRoutes(server, ub.List, ub.Create, ub.Update, ub.Delete, ub.PermList, ub.PermAdd, ub.PermRemove, ub.SelfUpdate, ub.SelfPassword)
	hub.RegisterParentAuth(server, pb.Handle)

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
