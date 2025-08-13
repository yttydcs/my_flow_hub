# Manager API 使用说明

Manager 服务提供了 RESTful API 用于管理 MyFlowHub 系统。以下是可用的 API 端点：

## 基础信息

- 服务地址: `http://localhost:8090`
- 所有 API 路径前缀: `/api/`
- 鉴权: 除 `/api/auth/login` 外，其余接口都需要请求头 `Authorization: Bearer <userKey>`（key-only 模式）。
  提示：`/api/auth/login` 返回的 `token` 字段即 userKey 的明文值，请直接作为 Authorization 使用。

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
