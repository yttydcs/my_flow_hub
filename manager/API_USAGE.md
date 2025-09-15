# Manager API 使用说明

Manager 服务提供了 RESTful API 用于管理 MyFlowHub 系统。以下是可用的 API 端点：

## 基础信息

- 服务地址: `http://localhost:8090`
- 所有 API 路径前缀: `/api/`
- 鉴权: 除 `/api/auth/login` 外，其余接口都需要请求头 `Authorization: Bearer <userKey>`（key-only 模式）。
  提示：`/api/auth/login` 返回的 `token` 字段即 userKey 的明文值，请直接作为 Authorization 使用。

### 安全与密钥

- Manager 在系统中只是前端的后端（BFF），不具备系统级特权。
- 二进制父链路认证（ParentAuth）与管理面登录（ManagerAuth）使用不同的密钥：
  - ManagerAuth 使用 `Server.ManagerToken`（Manager → Server）。
  - ParentAuth 使用 `Server.RelayToken`（上级校验）与 `Relay.SharedToken`（下级发起）。
- 建议将 `RelayToken/SharedToken` 与 `ManagerToken` 分离，避免越权与密钥复用风险。

## API 端点

### 1. 健康检查

**GET** `/health`

返回服务状态信息。

### 2. 调试信息

**GET** `/api/debug/db`

返回数据库连接状态和基本统计信息，用于调试。

### 3. 设备管理

#### 获取所有节点

**GET** `/api/nodes`

获取系统中的所有设备节点信息。

#### 创建设备（管理员或具备对应权限，非管理员需提供 ParentID 且拥有该父节点的控制权）

**POST** `/api/nodes`

创建一个新设备。

说明：服务端从请求头 `Authorization: Bearer <userKey>` 中解析用户身份与权限；非管理员不得将 OwnerUserID 设为他人，未指定时默认归自己。

**请求体**:
```json
{
  "HardwareID": "new-device-001",
  "Name": "New Sensor",
  "Role": "node",
  "ParentID": 1
}
```

#### 更新设备（管理员或具备对应权限；非管理员修改 ParentID 需能控制新父节点，不得将 OwnerUserID 改为他人）

**PUT** `/api/nodes`

更新一个已存在的设备。

**请求体**:
```json
{
  "ID": 2,
  "Name": "Updated Sensor Name",
  "Role": "node"
}
```

#### 删除设备（管理员或具备对应权限）

**DELETE** `/api/nodes`

删除一个设备及其所有关联的变量。

**请求体**:
```json
{
  "id": 2
}
```

### 4. 变量管理（管理员或具备对应权限）

#### 获取变量

**GET** `/api/variables[?deviceId=<设备ID>]`

获取变量信息，可选择性按设备ID过滤。

**参数**:
- `deviceId` (可选): 设备的数字ID

**响应示例**:
```json
{
  "success": true,
  "data": [
    {
      "ID": 1,
      "OwnerDeviceID": 2,
      "VariableName": "temperature",
      "Value": 25.6,
      "UpdatedAt": "2025-08-08T10:29:45Z"
    },
    {
      "ID": 2,
      "OwnerDeviceID": 2,
      "VariableName": "humidity",
      "Value": 65.2,
      "UpdatedAt": "2025-08-08T10:29:45Z"
    }
  ]
}
```

#### 新增或更新变量

**PUT** `/api/variables`

新增或更新一个或多个变量的值。

**请求体**:
```json
{
  "variables": {
    "temperature": 26.5,
    "[10002].humidity": 60.0,
    "(Sensor Node 1).status": "online"
  }
}
```

**响应示例**:
```json
{
  "success": true,
  "message": "Variables update sent"
}
```

#### 删除变量

**DELETE** `/api/variables`

删除一个或多个变量。

**请求体**:
```json
{
  "variables": [
    "[10002].temperature",
    "(Sensor Node 1).status"
  ]
}
```

### 5. 发送管理指令（管理员或具备对应权限）

**POST** `/api/message`

向指定节点发送管理消息/指令。

**请求体**:
```json
{
  "target": 10002,
  "type": "msg_send",
  "payload": {
    "command": "restart",
    "parameters": {
      "delay": 5
    }
  }
}
```

**响应示例**:
```json
{
  "success": true,
  "message": "Message sent successfully",
  "response": {
    "ID": "response-12345",
    "Type": "response",
    "Source": 10002,
    "Payload": {
      "success": true,
      "result": "Restart scheduled"
    }
  }
}
```

### 6. 用户管理（仅管理员）

#### 列出用户

GET `/api/users`

#### 新建用户

POST `/api/users`

请求体：
```json
{ "username": "bob", "displayName": "Bob", "password": "pass" }
```

#### 更新用户

PUT `/api/users`

请求体：
```json
{ "id": 2, "displayName": "Bobby", "disabled": false, "password": "newpass" }
```

#### 删除用户

DELETE `/api/users`

请求体：
```json
{ "id": 2 }
```

### 7. 用户权限管理（仅管理员）

POST `/api/users/perms/list`  请求体：`{ "userId": 2 }`

POST `/api/users/perms/add`   请求体：`{ "userId": 2, "node": "admin.manage" }`

POST `/api/users/perms/remove` 请求体：`{ "userId": 2, "node": "var.read.**" }`

## 错误响应

所有API在发生错误时都会返回统一的错误格式：

```json
{
  "success": false,
  "error": "错误描述信息"
}
```

常见的HTTP状态码：
- `200`: 成功
- `400`: 请求参数错误
- `404`: 资源未找到
- `500`: 服务器内部错误
- `503`: 服务不可用（通常是与Hub连接断开）

## 变量名称规范

在更新变量时，支持以下格式的变量名称：

1. **简单变量名**: `temperature` - 针对发送请求的设备
2. **设备ID格式**: `[10002].temperature` - 针对指定设备ID的变量
3. **设备名称格式**: `(Sensor Node 1).temperature` - 针对指定设备名称的变量

## 使用示例

### 登录并获取所有节点：

```bash
# 登录（获取 userKey 明文）
TOKEN=$(curl -s -X POST http://localhost:8090/api/auth/login -H "Content-Type: application/json" -d '{"username":"admin","password":"admin123!"}')
USER_KEY=$(echo $TOKEN | jq -r .token)

# 获取所有节点
curl -H "Authorization: Bearer $USER_KEY" -X GET http://localhost:8090/api/nodes
```

### 更新变量：

```bash
curl -X POST http://localhost:8090/api/variables \
  -H "Content-Type: application/json" -H "Authorization: Bearer $USER_KEY" \
  -d '{"variables": {"temperature": 25.5, "status": "active"}}'
```

### 发送管理指令：

```bash
curl -X POST http://localhost:8090/api/message \
  -H "Content-Type: application/json" -H "Authorization: Bearer $USER_KEY" \
  -d '{
    "target": 10002,
    "type": "msg_send",
    "payload": {
      "command": "get_status"
    }
  }'
```

---

## Protobuf 负载调试指南（WS 抓包与解析）

自 2025-09 起，Hub 二进制协议的负载改为 Protobuf。以下提供抓包与调试建议：

### 抓包建议

- 使用浏览器开发者工具（Network → WS）或 `mitmproxy/Wireshark` 抓取 WebSocket 二进制帧。
- 帧头固定 38B（小端）：TypeID(2) + Reserved(4) + MsgID(8) + Source(8) + Target(8) + Timestamp(8)。第 39 字节起为 Protobuf 负载。
- 关注 TypeID 与 `DOCS.md` 的“TypeID → Protobuf 消息”对照表。

### 负载解码（Go 端验证）

在服务端或本地小程序中，可快速验证某帧负载：

```go
// p 为去掉帧头后的负载 []byte；以 USER_LOGIN_RESP 为例
var resp pb.UserLoginResp
if err := proto.Unmarshal(p, &resp); err != nil {
    panic(err)
}
fmt.Println(resp.GetUserId(), resp.GetUsername(), resp.GetPermissions())
```

### 负载解码（Node/浏览器）

推荐采用 `ts-proto` 或官方/第三方 protobufjs 生成器基于 `pkg/protocol/pb/myflowhub.proto` 生成前端解码代码：

1) 使用 protobufjs：

```ts
import { Root } from 'protobufjs';

const root = await Root.load('/path/to/myflowhub.proto');
const UserLoginResp = root.lookupType('myflowhub.v1.UserLoginResp');

// 去掉 38B 帧头后：
const payload = new Uint8Array(frame.slice(38));
const msg = UserLoginResp.decode(payload);
console.log(UserLoginResp.toObject(msg));
```

2) 使用 ts-proto 生成声明良好的 TS 类型：

- 生成命令示例（在前端工程中执行）：
  ```bash
  protoc \
    --ts_out=. \
    --ts_opt=esModuleInterop=true,unrecognizedEnum=false,env=browser \
    myflowhub.proto
  ```
- 使用：
  ```ts
  import { myflowhub } from './gen/myflowhub';
  const m = myflowhub.v1.UserLoginResp.decode(payload);
  ```

### 常见问题

- 负载长度正常但解码失败：确认是否已去除 38 字节帧头；确认 TypeID 与选择的 pb 消息匹配。
- optional 字段为零值：proto3 optional 未设置时不会出现在编码里，前端需要判空逻辑。
- 大包抓包难阅读：建议在服务端/Manager 加日志钩子输出 `protojson.Marshal` 的可读版供调试（仅在开发环境开启）。

### 参考

- `pkg/protocol/pb/myflowhub.proto`：权威 Schema
- `DOCS.md`：帧头、TypeID 映射与说明
