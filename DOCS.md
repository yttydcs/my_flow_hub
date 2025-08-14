## 日志（审计）

Manager API
- GET/POST /api/logs 获取审计日志，支持筛选：keyword, subjectType, decision, action, startAt, endAt, page, pageSize。
	- 需要权限：log.read 或 admin.manage。

Hub 消息
- log_list：同上，payload 附带 userKey。

# 文档索引

- 顶层 README：项目概览、运行方式与权限说明（README.md）
- 权限系统设计：节点语法、隐式权限、管理员与首启规则、API 约束（权限设计.md）
- Manager API 使用说明：REST 端点与示例（manager/API_USAGE.md）
- Web 前端说明：环境、启动、登录/权限、目录结构（web/README.md）

## 快速开始（PowerShell）

```powershell
# 1) 启动 Server
cd d:\rj\MyFlowHub\server
go run .\cmd\myflowhub

# 2) 启动 Manager
cd d:\rj\MyFlowHub\manager
go run .\cmd\manager

# 3) 启动 Web（开发）
cd d:\rj\MyFlowHub\web
npm install
npm run dev
```

## 认证与权限

- 除 /api/auth/login 外，其余管理 API 需 Authorization: Bearer <token>
- 登录返回 token、user、permissions；前端基于 `admin.manage` 控制管理员页面

## 默认管理员与首启规则

- 仅在“新建数据库或首次创建 users 表”时自动创建 admin，并授予 admin.manage 与 **
- 已有用户表时不创建、不赋权
