import type { ApiResponse, DeviceTreeNode, DeviceVariable } from '@/types/api'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8090/api'

console.log('API_BASE_URL:', API_BASE_URL)
console.log('Environment variables:', import.meta.env)

class ApiService {
  private async request<T>(endpoint: string, options?: RequestInit): Promise<ApiResponse<T>> {
    const url = `${API_BASE_URL}${endpoint}`
    
    try {
      const response = await fetch(url, {
        headers: {
          'Content-Type': 'application/json',
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
  async createDevice(data: { hardwareId: string; name: string; role: string; parentId?: number }): Promise<ApiResponse> {
    return this.request('/nodes', {
      method: 'POST',
      body: JSON.stringify(data)
    })
  }

  async updateDevice(data: { id: number; name: string; role: string; parentId?: number }): Promise<ApiResponse> {
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
  async createVariable(data: { name: string; value: any; deviceId: number }): Promise<ApiResponse> {
    return this.request('/variables', {
      method: 'POST',
      body: JSON.stringify(data)
    })
  }

  async updateVariableNew(data: { id: number; name: string; value: any }): Promise<ApiResponse> {
    return this.request('/variables', {
      method: 'PUT',
      body: JSON.stringify(data)
    })
  }

  async deleteVariable(id: number): Promise<ApiResponse> {
    return this.request('/variables', {
      method: 'DELETE',
      body: JSON.stringify({ id })
    })
  }

  // 调试：获取数据库信息
  async getDebugInfo(): Promise<ApiResponse> {
    return this.request('/debug/db')
  }
}

export const apiService = new ApiService()
