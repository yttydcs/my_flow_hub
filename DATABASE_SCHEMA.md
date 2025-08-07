# MyFlowHub 数据库表结构设计 (PostgreSQL)

本文档定义了 MyFlowHub 项目所需的 PostgreSQL 数据库表结构。

## 1. 表: `devices`

此表存储网络中所有设备（包括中枢、中继和节点）的元数据。

```sql
-- 设备角色枚举类型
CREATE TYPE device_role AS ENUM ('hub', 'relay', 'node');

-- 设备表
CREATE TABLE devices (
    -- 设备唯一ID，建议使用UUID，但为简化，此处使用自增ID
    id BIGSERIAL PRIMARY KEY,
    
    -- 用于程序中识别的、由服务器分配的唯一ID
    device_uid VARCHAR(255) UNIQUE NOT NULL,
    
    -- 用于认证的密钥，应存储哈希值
    secret_key_hash VARCHAR(255) NOT NULL,
    
    -- 设备的物理硬件ID (如MAC地址)，用于首次注册时识别设备
    hardware_id VARCHAR(255) UNIQUE,
    
    -- 设备角色
    role device_role NOT NULL,
    
    -- 上级设备的ID (自引用外键)
    -- 节点的上级是中继或中枢，中继的上级是中枢
    -- 中枢没有上级，此字段为 NULL
    parent_id BIGINT REFERENCES devices(id) ON DELETE SET NULL,
    
    -- 设备名称或描述
    name VARCHAR(255),
    
    -- 最后一次心跳时间
    last_seen TIMESTAMPTZ,
    
    -- 创建和更新时间戳
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 为常用查询字段创建索引
CREATE INDEX idx_devices_device_uid ON devices(device_uid);
CREATE INDEX idx_devices_hardware_id ON devices(hardware_id);
CREATE INDEX idx_devices_parent_id ON devices(parent_id);
```

### 字段说明:
- `id`: 主键。
- `device_uid`: 由服务器在注册时分配的、全局唯一的设备标识符。
- `secret_key_hash`: **绝不存储明文密钥**。这里存储使用 bcrypt 或 scrypt 等算法哈希后的值。
- `hardware_id`: 设备的物理唯一标识（如MAC地址、序列号）。首次注册时提交，用于防止重复注册。
- `role`: 明确设备的职责。
- `parent_id`: 构建设备间的层级关系。
- `last_seen`: 可用于判断设备在线状态。

---

## 2. 表: `device_variables`

此表是“变量池”的核心实现，存储每个设备的所有变量。

```sql
-- 变量池表
CREATE TABLE device_variables (
    -- 变量ID
    id BIGSERIAL PRIMARY KEY,
    
    -- 所属设备的ID
    owner_device_id BIGINT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    
    -- 变量名
    variable_name VARCHAR(255) NOT NULL,
    
    -- 变量值，使用 JSONB 类型以支持多种数据结构
    value JSONB NOT NULL,
    
    -- 创建和更新时间戳
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- 确保每个设备下的变量名是唯一的
    UNIQUE (owner_device_id, variable_name)
);

-- 为常用查询字段创建索引
CREATE INDEX idx_device_variables_owner_id ON device_variables(owner_device_id);
-- 创建 GIN 索引以加速 JSONB 内部的查询
CREATE INDEX idx_device_variables_value ON device_variables USING GIN(value);
```

### 字段说明:
- `owner_device_id`: 指明这个变量属于哪个设备。
- `variable_name`: 变量的键。
- `value`: 变量的值。使用 `JSONB` 可以高效地存储和查询数字、字符串、布尔值、数组和对象。

---

## 3. 表: `access_permissions`

此表定义了哪个设备（请求者）可以对哪个设备（目标）的哪个变量执行何种操作。

```sql
-- 权限操作枚举类型
-- 'read'/'write' 用于变量池访问
-- 'send_message' 用于通用消息传递
CREATE TYPE permission_action AS ENUM ('read', 'write', 'send_message');

-- 访问权限表
CREATE TABLE access_permissions (
    id BIGSERIAL PRIMARY KEY,
    
    -- 请求方设备的ID
    requester_device_id BIGINT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    
    -- 目标设备的ID
    target_device_id BIGINT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    
    -- 目标变量名。对于 'read'/'write' 操作，此字段指定变量名。
    -- 对于 'send_message' 操作，此字段应为 NULL。
    -- 如果为 NULL 或 '*'，规则适用于目标设备的所有变量。
    target_variable_name VARCHAR(255),
    
    -- 允许的操作
    action permission_action NOT NULL,
    
    -- 创建时间戳
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- 确保权限规则的唯一性
    UNIQUE (requester_device_id, target_device_id, target_variable_name, action)
);

-- 为常用查询字段创建索引
CREATE INDEX idx_access_permissions_requester_id ON access_permissions(requester_device_id);
CREATE INDEX idx_access_permissions_target_id ON access_permissions(target_device_id);
```

### 字段说明:
- `requester_device_id`: 谁发起的请求。
- `target_device_id`: 请求的目标是谁。
- `target_variable_name`: 对于变量操作，指定变量名（`*` 作为通配符）。对于消息发送，此字段通常为 `NULL`。
- `action`: 允许的操作是 `read`（读变量）、`write`（写变量）还是 `send_message`（发送通用消息）。

---

文档编写阶段基本完成。下一步是进入**概念验证 (PoC)** 阶段，我们将使用 Go 语言和这个数据库设计来实现一个最小化的中枢服务器和节点模拟器。
