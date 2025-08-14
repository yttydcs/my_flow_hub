// API 响应类型定义
export interface ApiResponse<T = any> {
  success: boolean
  data?: T
  message?: string
}

// 设备相关类型
export interface Device {
  ID: number
  DeviceUID: number
  HardwareID: string
  Role: 'hub' | 'relay' | 'node' | 'manager'
  Name: string
  ParentID?: number | null
  Parent?: Device | null
  Children?: Device[] | null
  children?: DeviceTreeNode[]  // 用于树形结构
  LastSeen?: string | null
  CreatedAt: string
  UpdatedAt: string
}

// 树形节点类型
export interface DeviceTreeNode extends Device {
  children: DeviceTreeNode[]
}

// 变量类型
export interface DeviceVariable {
  ID: number
  OwnerDeviceID: number
  Device: Device
  VariableName: string
  Value: any
  CreatedAt: string
  UpdatedAt: string
}

// WebSocket 消息类型
export interface WSMessage {
  id: string
  source?: number
  target?: number
  type: string
  timestamp: string
  payload: any
}

// 用户类型
export interface User {
  ID: number
  Username: string
  DisplayName?: string
  Disabled?: boolean
  CreatedAt: string
  UpdatedAt: string
}

// 密钥类型（前端使用字段子集）
export interface Key {
  ID: number
  OwnerUserID?: number | null
  BindSubjectType?: 'user' | 'device' | null
  BindSubjectID?: number | null
  ExpiresAt?: string | null
  MaxUses?: number | null
  RemainingUses?: number | null
  Revoked: boolean
  IssuedAt: string
  // SecretHash 与 Meta 通常不在列表中直接展示
}

// 审计日志类型
export interface AuditLog {
  ID: number
  SubjectType: string
  SubjectID?: number | null
  Action: string
  Resource: string
  Decision: string
  IP: string
  UA: string
  At: string
  Extra?: any
}

export interface PagedAuditLogs {
  Items: AuditLog[]
  Total: number
  Page: number
  Size: number
}

// 系统日志类型
export interface SystemLog {
  ID: number
  Level: 'info' | 'warn' | 'error' | string
  Source: string
  Message: string
  Details?: any
  At: string
}

export interface PagedSystemLogs {
  Items: SystemLog[]
  Total: number
  Page: number
  Size: number
}
