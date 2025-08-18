# MyFlowHub

MyFlowHub 是一个为物联网（IoT）和分布式系统设计的、轻量级的、层级化的消息与数据同步系统。

## 设计理念

项目的核心是构建一个抽象网络，其中所有设备（从云端服务器到嵌入式节点）都是平等的“节点”，可以相互通信。

- **统一服务端**: 系统中的“中枢（Hub）”和“中继（Relay）”是同一种服务端软件的不同配置实例。一个没有上级的服务端即为中枢，有上级的则为中继。
- **持久化身份**: 每个节点（包括中枢和中继）在首次启动时都会在数据库中注册自己，获得一个唯一的、持久化的数字ID。
- **分层路由**: 消息可以在网络中向上（向父节点）或向下（向子节点）传递，实现了高效、可扩展的路由。
- **双向数据绑定**: 以“变量池”为核心，实现了设备状态与云端数据的实时、双向同步。

## 期望功能与实现状态

| 功能点 | 状态 | 备注 |
| :--- | :--- | :--- |
| **核心网络** | | |
| 统一服务端（中枢/中继） | ✅ 已实现 | 通过 `config.json` 配置。 |
| 服务端自举注册 | ✅ 已实现 | 服务器启动时会在数据库中为自己创建持久化身份。 |
| 节点动态注册与认证 | ✅ 已实现 | 客户端可通过 `hardwareId` 注册并获取数字ID和密钥。 |
| 客户端身份持久化 | ✅ 已实现 | Web客户端使用 `localStorage`。 |
| 并发安全的连接管理 | ✅ 已实现 | 采用 Hub-and-Spoke 模型，保证并发安全。 |
| 分层消息路由 | ✅ 已实现 | 支持点对点、广播和向上传递。 |
| **变量池** | | |
| 变量更新 | ✅ 已实现 | 支持跨命名空间更新。 |
| 变量查询 | ✅ 已实现 | 支持按名称或ID进行高级批量查询。 |
| 变量名合法性验证 | ✅ 已实现 | 限制为汉字、字母、数字、下划线。 |
| **双向绑定** | | |
| 下行通知（变量变更） | ✅ 已实现 | 变量被修改后，会主动通知在线的所属节点。 |
| 上线状态同步 | ✅ 已实现 | 节点认证成功后，会收到其变量池的完整初始状态。 |
| **管理后台** | | |
| BFF 架构 | ✅ 已实现 | `manager` 服务作为前端的后端。 |
| 特权节点认证 | ✅ 已实现 | `manager` 节点使用 Token 进行注册。 |
| WebSocket 管理客户端 | ✅ 已实现 | Manager 通过 WebSocket 连接到中枢/中继，并支持自动重连。 |
| RESTful 管理API | ✅ 已实现 | 提供节点和变量的增删改查、消息发送等API。 |
| 数据统一处理 | ✅ 已实现 | Manager 通过 Server 获取数据，不直接访问数据库。 |
| 设备父级关系 | ✅ 已实现 | 自动维护设备的层级关系。 |
| **工具** | | |
| Web GUI 调试客户端 | ✅ 已实现 | `web-client/` 目录下的独立HTML文件。 |
| **其他** | | |
| 外部化配置 | ✅ 已实现 | 所有关键配置均在 `config.json` 中。 |
| 权限管理 | ✅ 已实现 | 基于权限节点与通配符，管理员需 admin.manage；支持用户权限编辑与快照下发。 |
| 二进制协议支持 | ✅ 已实现 | 仅二进制：WebSocket 子协议 myflowhub.bin.v1；JSON 已移除。 |
| 父链路认证（ParentAuth） | ✅ 已实现 | 使用 HMAC-SHA256 + 时间窗 + Nonce 防重放；密钥使用 RelayToken/SharedToken（与 ManagerToken 解耦）。 |

## 项目结构

-   **`pkg/`**: 共享的 Go 包，可被多个项目复用。
    -   `config/`: 配置加载和管理。
    -   `database/`: 数据库模型和操作。
    -   `protocol/`: 定义通信协议的 Go 结构体。
-   **`server/`**: Go 核心消息服务端。
    -   `cmd/myflowhub/main.go`: `main` 包，程序入口。
    -   `internal/`: 采用`controller-service-repository`分层架构。
        - `hub/`: WebSocket 连接管理和核心消息循环（仅二进制）。
        - `controller/`: 处理二进制消息（通过适配器），调用服务。
        - `service/`: 封装核心业务逻辑。
        - `repository/`: 数据持久化操作。
    - 二进制路由注册入口：`server/internal/hub/register.go`（集中将 TypeID 绑定到 controller 二进制适配器，由 main 在启动时完成注入）。
    - 父链路连接与认证：`server/internal/hub/parent_connector.go`（优先 ParentAuth，必要时兼容回退到 ManagerAuth）。
-   **`manager/`**: Go 后台管理服务 (BFF)。
    -   `cmd/manager/main.go`: `main` 包，程序入口。
    -   `internal/`:
        - `api/`: 提供RESTful API。
        - `client/`: 作为WebSocket客户端连接到`server`。
-   **`web/`**: Vue 3 前端管理界面。
-   **`web-client/`**: 用于快速调试的、独立的 HTML 客户端。

## 如何运行

1.  配置
    - 打开 `server/config.json`，在 `Database` 部分配置本地 PostgreSQL；可设置 `Server.DefaultAdmin`（见下文“默认管理员与首启规则”）。
    - Manager（BFF）仅负责 HTTP→Server 的桥接，并不具备特权。它使用 `Server.ManagerToken` 与 Server 进行 ManagerAuth。
    - 父链路 ParentAuth 使用独立密钥：
        - 上级 Server 使用 `Server.RelayToken` 校验；
        - 下级 Relay 使用 `Relay.SharedToken` 发起认证；
        - 为兼容旧配置，若上述为空才会回退到 `Server.ManagerToken`（不推荐）。
    - （可选）在 `server/config.json` 的 `Relay` 部分，将 `Enabled` 设置为 `true` 启动中继实例，并同时配置：
        - 上级节点的 `Server.RelayToken` 与本中继的 `Relay.SharedToken` 为相同的强随机值。
2.  启动核心服务（PowerShell）
    ```powershell
    cd d:\rj\MyFlowHub\server
    go run .\cmd\myflowhub
    ```
3.  启动管理后台服务（PowerShell）
    ```powershell
    cd d:\rj\MyFlowHub\manager
    go run .\cmd\manager
    ```
4.  开发前端（PowerShell）
    - 前端需要 Node 20.19+ 或 22.12+；首次安装依赖后再启动。
    ```powershell
    cd d:\rj\MyFlowHub\web
    npm install
    npm run dev
    ```
5.  访问
    *   **核心服务WebSocket**: `ws://localhost:8080/ws`
    *   **管理API服务**: `http://localhost:8090/api/`
    *   **管理后台前端**: `http://localhost:5173`（Vite 开发服务器）
    *   **调试客户端**: 打开 `web-client/index.html`

## 权限与认证（重要）

- 管理 API 除 `/api/auth/login` 外，均需携带 `Authorization: Bearer <token>` 请求头。
- 登录接口会返回 token、user、permissions（权限节点快照）；前端将 token 保存在状态中并自动添加到后续请求头。
- 前端路由已配置登录守卫：未登录只能访问 `/login`；仅拥有 `admin.manage` 的用户可访问“用户管理”等管理员页面。
- 权限节点及匹配语义详见《权限系统设计文档》（`权限设计.md`）。

### ParentAuth（父链路认证）

- 目的：为中继（Relay）连接父节点（Hub/上级 Server）提供二进制认证。
- 协议：TypeID 130/131，HMAC-SHA256(密钥, ts|nonce|hardware_id|caps)。
- 安全：±5 分钟时间窗口校验；16 字节随机 Nonce，服务器端 10 分钟 TTL 去重，防重放。
- 密钥：
    - 上级校验用 `Server.RelayToken`；
    - 下级发起用 `Relay.SharedToken`；
    - 建议二者一致；避免与 `ManagerToken` 混用。

Manager 角色澄清：Manager 只是前端的后端（BFF），负责 HTTP→Server 的中转，不再拥有系统级特权。ParentAuth 与 ManagerAuth 的密钥应分离。

## 默认管理员与首启规则

- 首次创建数据库或首次创建 `users` 表时，系统会：
  1) 自动创建默认管理员账户（用户名取自 `Server.DefaultAdmin.Username`，默认为 `admin`）。
  2) 为该账户授予 `admin.manage` 与 `**` 两个权限节点。
- 如果在启动时检测到用户表已存在，则不会自动创建 admin，也不会自动赋予任何权限（完全跳过）。
- 建议在首次登录后修改默认管理员密码。

## 下一步计划

1.  **完善前端界面**: 持续完善设备树与变量管理视图，增强交互与状态提示。
2.  **权限系统增强**: 引入密钥/借权机制的完整流程（签发、撤销、限次/到期、设备安装密钥）。
3.  **增强管理功能**: 
    - 实时监控节点状态
    - 批量操作支持
    - 历史数据查询
    - 系统日志查看
4.  **性能优化**: 
    - 二进制编解码与路由热路径优化（必要处可用代码生成/内联）
    - 添加消息压缩
    - 优化数据库查询性能
5.  **部署和运维**:
    - Docker 容器化支持
    - 配置管理优化
    - 监控和告警系统
