# MyFlowHub 通信协议规范

## 1. 概述

本协议定义了 MyFlowHub 中节点、中继和中枢之间的通信标准。

- **传输层**: WebSocket
- **数据格式**: JSON
- **通信方式**: 客户端-服务器模式，双向通信

## 2. 基础消息结构

所有通过 WebSocket 发送的消息都应遵循以下增强的基础结构：

```json
{
  "id": "unique-message-id",
  "source": "source-device-id",
  "target": "target-device-id",
  "type": "message-type",
  "timestamp": "2023-10-27T10:00:00Z",
  "payload": {
    // ... 具体内容 ...
  }
}
```

- **id**: `string` - 消息的唯一标识符，建议使用 UUID。
- **source**: `string` - 消息发送方的设备ID。**对于客户端发往服务器的消息，此字段可留空，由服务器根据连接信息填充**。
- **target**: `string` - 消息接收方的设备ID。对于发往服务器自身的消息，可以使用预留的ID，如 `server`。
- **type**: `string` - 消息类型。
- **timestamp**: `string` (ISO 8601) - 消息发送时的 UTC 时间戳。
- **payload**: `object` - 消息的具体内容。

## 3. 消息类型与交互流程

### 3.1. 设备注册与认证

#### 3.1.1. 动态注册 (DHCP-like)

适用于首次接入网络的全新设备。

1.  **客户端 -> 服务器: `register_request`**
    一个没有凭证的设备请求注册。

    **Payload 结构:**
    ```json
    {
      "hardwareId": "device-hardware-unique-id", // e.g., MAC address, CPU serial
      "role": "manager", // 可选, 用于注册特权节点
      "token": "a-super-secret-manager-token" // 可选, 配合 role 使用
    }
    ```

2.  **服务器 -> 客户端: `register_response`**
    服务器分配新的凭证并返回给设备。设备应**持久化存储**这些信息。

    **Payload 结构:**
    ```json
    {
      "success": true,
      "deviceId": "generated-unique-device-id",
      "secretKey": "generated-strong-secret-key"
    }
    ```

#### 3.1.2. 身份认证

对于已注册的设备。

1.  **客户端 -> 服务器: `auth_request`**
    设备使用已有的凭证进行认证。

    **Payload 结构:**
    ```json
    {
      "deviceId": "my-persistent-device-id",
      "secretKey": "my-persistent-secret-key"
    }
    ```

2.  **服务器 -> 客户端: `auth_response`**
    服务器对认证请求的响应。

    **Payload 结构:**
    ```json
    {
      "success": true,
      "message": "Authentication successful"
    }
    ```

### 3.2. 级联心跳

心跳不仅用于保持连接，更承载了网络状态向上传递的重要职责。

1.  **下级 -> 上级: `ping`**
    任何设备（节点或中继）定期向上级发送心跳。

    **Payload 结构:**
    ```json
    {
      "status": "ok", // 当前设备自身的状态
      "subordinates": { // 可选，仅当中继上报时使用
        "changed": [
          { "deviceId": "sub-node-01", "status": "running" },
          { "deviceId": "sub-node-02", "status": "error", "error_code": 500 }
        ],
        "removed": ["sub-node-03"]
      }
    }
    ```
    - **status**: `string` - 描述当前设备自身的状态。
    - **subordinates**: `object` - 包含其下级网络状态的变更。
      - **changed**: `array` - 自上次心跳以来，状态有变更的下级设备列表。
      - **removed**: `array` - 自上次心跳以来，已失联或被移除的下级设备ID列表。
    - **带宽优化**: 如果下级网络无任何变化，`subordinates` 字段可以省略不传。

2.  **上级 -> 下级: `pong`**
    服务器对 `ping` 消息的响应，确认收到心跳。

    **Payload 结构:**
    ```json
    {}
    ```

### 3.3. 通用消息与变量操作

得益于新的基础消息结构，通用消息传递和变量操作可以统一。`target` 字段天然地指定了操作对象。

1.  **更新变量: `var_update`**
    设备更新**自身**的变量。`target` 应设为发送方自己的 `deviceId` 或服务器ID。

    **Payload 结构:**
    ```json
    {
      "variables": { "temperature": 25.5, "status": "running" }
    }
    ```

2.  **获取变量: `var_get`**
    一个设备请求获取另一个设备的变量。

    **Payload 结构:**
    ```json
    {
      "keys": ["config", "threshold"]
    }
    ```
    - **路由**: `target` 字段设为目标设备的ID。

3.  **通用消息: `msg_send`**
    一个设备向另一个设备发送任意数据。

    **Payload 结构:**
    ```json
    {
      "data": { "command": "reboot", "delay": "30s" }
    }
    ```
    - **路由**: `target` 字段设为目标设备的ID。

#### 响应与通知

服务器和客户端之间的响应和通知将使用统一的 `response` 和 `notify` 类型。

1.  **变量变更通知: `var_notify`**
    当一个设备的变量被修改，或一个设备上线需要同步其全部状态时，服务器会向该设备推送此消息。

    **Payload 结构:**
    ```json
    {
      "source": "device-id-or-server", // 变更来源
      "variables": { // 包含一个或多个变更的变量
        "temperature": 25.5,
        "status": "running"
      }
    }
    ```

2.  **通用响应: `response`**
    服务器对客户端请求的通用响应。

    **Payload 结构:**
    ```json
    {
      "success": true,
      "original_id": "request-message-id", // 原始请求ID
      "data": {
        // ... 响应的具体内容，例如获取到的变量 ...
        "variables": { "config": { "timeout": 30 } }
      }
    }
    ```

## 4. 错误处理

当请求处理失败时，服务器会返回一个 `error` 类型的消息。

**Payload 结构:**
```json
{
  "code": 1001, // 错误码
  "message": "Permission denied", // 错误信息
  "original_id": "request-message-id" // 可选，关联的原始请求ID
}
```

---
下一步将是设计数据库表结构。
