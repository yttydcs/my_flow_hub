import type { ApiResponse, DeviceTreeNode, DeviceVariable, Key, Device, PagedSystemLogs } from '@/types/api'
import type { User } from '@/types/api'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8090/api'

console.log('API_BASE_URL:', API_BASE_URL)
console.log('Environment variables:', import.meta.env)

class ApiService {
  private token: string | null = null

  setToken(t: string | null) {
    this.token = t
  }

  private async request<T>(endpoint: string, options?: RequestInit): Promise<ApiResponse<T>> {
    const url = `${API_BASE_URL}${endpoint}`
    
    try {
      const response = await fetch(url, {
        headers: {
          'Content-Type': 'application/json',
          ...(this.token ? { 'Authorization': `Bearer ${this.token}` } : {}),
          ...options?.headers,
        },
        ...options,
      })
      
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`)
      }
      
      return await response.json()
    } catch (error) {
      console.error(`API request failed: ${url}`, error)
      throw error
    }
  }

  // 登录
  async login(username: string, password: string): Promise<ApiResponse<{ token: string }>> {
    return this.request<{ token: string }>(`/auth/login`, {
      method: 'POST',
      body: JSON.stringify({ username, password })
    })
  }

  // 获取当前用户
  async me(): Promise<ApiResponse<{ user: any; permissions: string[] }>> {
    return this.request<{ user: any; permissions: string[] }>(`/auth/me`, { method: 'GET' })
  }

  // 登出（撤销当前 token）
  async logout(): Promise<ApiResponse> {
    return this.request(`/auth/logout`, { method: 'POST' })
  }

  // 获取设备树
  async getDeviceTree(): Promise<ApiResponse<DeviceTreeNode[]>> {
    return this.request<DeviceTreeNode[]>('/nodes')
  }

  // 获取所有变量
  async getAllVariables(): Promise<ApiResponse<DeviceVariable[]>> {
    return this.request<DeviceVariable[]>('/variables')
  }

  // 获取特定设备的变量
  async getDeviceVariables(deviceId: number): Promise<ApiResponse<DeviceVariable[]>> {
    return this.request<DeviceVariable[]>(`/variables?deviceId=${deviceId}`)
  }

  // 更新变量值
  async updateVariable(deviceUID: number, name: string, value: any): Promise<ApiResponse> {
    return this.request('/variables', {
      method: 'POST',
      body: JSON.stringify({
        device_uid: deviceUID,
        name: name,
        value: value
      })
    })
  }

  // 发送消息给设备
  async sendMessage(deviceUID: number, message: string): Promise<ApiResponse> {
    return this.request('/message', {
      method: 'POST',
      body: JSON.stringify({
        device_uid: deviceUID,
        message: message
      })
    })
  }

  // 设备管理CRUD操作
  async createDevice(data: { HardwareID: string; Name: string; Role: string; ParentID?: number | null }): Promise<ApiResponse> {
    return this.request('/nodes', {
      method: 'POST',
      body: JSON.stringify(data)
    })
  }

  async updateDevice(data: { ID: number; HardwareID: string; Name: string; Role: string; ParentID?: number | null }): Promise<ApiResponse> {
    return this.request('/nodes', {
      method: 'PUT',
      body: JSON.stringify(data)
    })
  }

  async deleteDevice(id: number): Promise<ApiResponse> {
    return this.request('/nodes', {
      method: 'DELETE',
      body: JSON.stringify({ id })
    })
  }

  // 变量管理CRUD操作
  async updateVariableNew(data: { [key: string]: any }): Promise<ApiResponse> {
    return this.request('/variables', {
      method: 'PUT',
      body: JSON.stringify(data)
    })
  }

  async deleteVariable(data: { variables: string[] }): Promise<ApiResponse> {
    return this.request('/variables', {
      method: 'DELETE',
      body: JSON.stringify(data)
    })
  }

  // 调试：获取数据库信息
  async getDebugInfo(): Promise<ApiResponse> {
    return this.request('/debug/db')
  }

  // ===== 用户管理 =====
  async getUsers(): Promise<ApiResponse<User[]>> {
    return this.request<User[]>('/users', { method: 'GET' })
  }

  async createUser(data: { username: string; displayName?: string; password: string }): Promise<ApiResponse<User>> {
    return this.request<User>('/users', { method: 'POST', body: JSON.stringify(data) })
  }

  async updateUser(data: { id: number; displayName?: string; disabled?: boolean; password?: string }): Promise<ApiResponse> {
    return this.request('/users', { method: 'PUT', body: JSON.stringify(data) })
  }

  async deleteUser(id: number): Promise<ApiResponse> {
    return this.request('/users', { method: 'DELETE', body: JSON.stringify({ id }) })
  }

  // 个人资料（自助）
  async updateProfile(data: { displayName?: string }): Promise<ApiResponse> {
    return this.request('/profile', { method: 'PUT', body: JSON.stringify(data) })
  }
  async changeMyPassword(oldPassword: string, newPassword: string): Promise<ApiResponse> {
    return this.request('/profile/password', { method: 'PUT', body: JSON.stringify({ oldPassword, newPassword }) })
  }

  // 用户权限管理
  async listUserPerms(userId: number): Promise<ApiResponse<string[]>> {
    return this.request<string[]>('/users/perms/list', { method: 'POST', body: JSON.stringify({ userId }) })
  }
  async addUserPerm(userId: number, node: string): Promise<ApiResponse> {
    return this.request('/users/perms/add', { method: 'POST', body: JSON.stringify({ userId, node }) })
  }
  async removeUserPerm(userId: number, node: string): Promise<ApiResponse> {
    return this.request('/users/perms/remove', { method: 'POST', body: JSON.stringify({ userId, node }) })
  }

  // ===== 密钥管理 =====
  async getKeys(): Promise<ApiResponse<Key[]>> {
    return this.request<Key[]>('/keys', { method: 'GET' })
  }
  async createKey(data: { bindType?: 'user' | 'device'; bindId?: number; expiresAt?: string; maxUses?: number; meta?: any; nodes?: string[] }): Promise<ApiResponse<any>> {
    return this.request<any>('/keys', { method: 'POST', body: JSON.stringify(data) })
  }
  async updateKey(data: Partial<Key> & { ID: number }): Promise<ApiResponse> {
    return this.request('/keys', { method: 'PUT', body: JSON.stringify(data) })
  }
  async deleteKey(id: number): Promise<ApiResponse> {
    return this.request('/keys', { method: 'DELETE', body: JSON.stringify({ id }) })
  }

  // 可见设备（用于创钥时提供设备选项）
  async getKeyDevices(): Promise<ApiResponse<Device[]>> {
    return this.request<Device[]>('/keys/devices', { method: 'GET' })
  }

  // ===== 日志 =====
  async getLogs(params: { keyword?: string; level?: string; source?: string; startAt?: number; endAt?: number; page?: number; pageSize?: number } = {}): Promise<ApiResponse<PagedSystemLogs>> {
    // 使用 POST 方便传较多筛选项
    return this.request<PagedSystemLogs>('/logs', { method: 'POST', body: JSON.stringify(params) })
  }
}

export const apiService = new ApiService()
