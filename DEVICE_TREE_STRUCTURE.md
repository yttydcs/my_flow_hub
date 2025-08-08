# 设备树形结构实现

## 修改内容

### 1. 数据库模型更新 (`pkg/database/models.go`)
- 在 `Device` 结构体中添加了 `Children []Device` 字段
- 这允许 GORM 自动处理父子关系的双向关联

```go
type Device struct {
    // ... 其他字段
    ParentID      *uint64
    Parent        *Device   `gorm:"foreignKey:ParentID"`
    Children      []Device  `gorm:"foreignKey:ParentID"`  // 新增
    // ... 其他字段
}
```

### 2. 查询逻辑更新 (`server/internal/hub/queries.go`)

#### 新增 DeviceTreeNode 结构
```go
type DeviceTreeNode struct {
    database.Device
    Children []DeviceTreeNode `json:"children"`
}
```

#### 新增 buildDeviceTree 方法
- 构建完整的设备树形结构
- 自动处理任意深度的嵌套关系
- 将所有设备组织成层次结构

#### 更新 handleQueryNodes 方法
- 使用新的树形结构构建器
- 返回完整的设备树而不是平面列表

## 树形结构示例

假设有以下设备结构：
```
Hub (中枢) - DeviceUID: 10000
├── Relay1 (中继) - DeviceUID: 10001
│   ├── Node1 (节点) - DeviceUID: 10002
│   └── Node2 (节点) - DeviceUID: 10003
├── Relay2 (中继) - DeviceUID: 10004
│   └── Node3 (节点) - DeviceUID: 10005
└── Node4 (直连节点) - DeviceUID: 10006
```

### API 响应示例
```json
{
  "success": true,
  "data": [
    {
      "DeviceUID": 10000,
      "HardwareID": "hub-001",
      "Role": "hub",
      "Name": "Main Hub",
      "children": [
        {
          "DeviceUID": 10001,
          "HardwareID": "relay-001", 
          "Role": "relay",
          "Name": "Relay 1",
          "children": [
            {
              "DeviceUID": 10002,
              "Role": "node",
              "Name": "Node 1",
              "children": []
            },
            {
              "DeviceUID": 10003,
              "Role": "node", 
              "Name": "Node 2",
              "children": []
            }
          ]
        },
        {
          "DeviceUID": 10004,
          "HardwareID": "relay-002",
          "Role": "relay",
          "Name": "Relay 2", 
          "children": [
            {
              "DeviceUID": 10005,
              "Role": "node",
              "Name": "Node 3",
              "children": []
            }
          ]
        },
        {
          "DeviceUID": 10006,
          "Role": "node",
          "Name": "Direct Node",
          "children": []
        }
      ]
    }
  ]
}
```

## 优势

1. **完整的层次关系**：可以看到任意深度的父子关系
2. **高效查询**：只需要一次数据库查询，然后在内存中构建树形结构
3. **易于前端展示**：前端可以直接使用这个树形结构渲染设备树
4. **灵活性**：支持任意深度的嵌套，不限制层级数量
5. **性能优化**：避免了多次递归查询数据库

## 使用方法

Manager API 调用 `/nodes` 端点时，现在会返回完整的设备树形结构，包含所有父子关系信息。前端可以直接使用这个结构来构建设备管理界面的树形视图。
