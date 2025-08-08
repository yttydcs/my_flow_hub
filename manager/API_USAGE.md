# Manager API 使用说明

Manager 服务提供了 RESTful API 用于管理 MyFlowHub 系统。以下是可用的 API 端点：

## 基础信息

- **服务地址**: `http://localhost:8090`
- **所有API路径前缀**: `/api/`

## API 端点

### 1. 健康检查

**GET** `/health`

返回服务状态信息。

### 2. 调试信息

**GET** `/api/debug/db`

返回数据库连接状态和基本统计信息，用于调试。

### 3. 获取所有节点

**GET** `/api/nodes`

获取系统中的所有设备节点信息。

**响应示例**:
```json
{
  "success": true,
  "data": [
    {
      "ID": 1,
      "DeviceUID": 10001,
      "HardwareID": "hub-001",
      "Role": "hub",
      "Name": "Main Hub",
      "LastSeen": "2025-08-08T10:30:00Z",
      "CreatedAt": "2025-08-08T09:00:00Z"
    },
    {
      "ID": 2,
      "DeviceUID": 10002,
      "HardwareID": "node-001",
      "Role": "node",
      "Name": "Sensor Node 1",
      "LastSeen": "2025-08-08T10:29:45Z",
      "CreatedAt": "2025-08-08T09:15:00Z"
    }
  ]
}
```

### 3. 获取变量

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

### 4. 更新变量

**POST** `/api/variables`

更新一个或多个变量的值。

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

### 5. 发送管理指令

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

### 使用 curl 获取所有节点：

```bash
curl -X GET http://localhost:8090/api/nodes
```

### 使用 curl 更新变量：

```bash
curl -X POST http://localhost:8090/api/variables \
  -H "Content-Type: application/json" \
  -d '{"variables": {"temperature": 25.5, "status": "active"}}'
```

### 使用 curl 发送管理指令：

```bash
curl -X POST http://localhost:8090/api/message \
  -H "Content-Type: application/json" \
  -d '{
    "target": 10002,
    "type": "msg_send",
    "payload": {
      "command": "get_status"
    }
  }'
```
