import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import type { DeviceTreeNode, DeviceVariable, Device } from '@/types/api'
import { apiService } from '@/services/api'

export const useHubStore = defineStore('hub', () => {
  // 状态
  const devices = ref<DeviceTreeNode[]>([])
  const variables = ref<DeviceVariable[]>([])
  const selectedDevice = ref<Device | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const lastUpdated = ref<Date | null>(null)

  // 计算属性
  const deviceCount = computed(() => {
    const countDevices = (nodes: DeviceTreeNode[]): number => {
      let count = 0
      for (const node of nodes) {
        count += 1 + countDevices(node.children || [])
      }
      return count
    }
    return countDevices(devices.value)
  })

  const hubDevices = computed(() => 
    devices.value.filter(device => device.Role === 'hub')
  )

  const selectedDeviceVariables = computed(() => 
    selectedDevice.value ? 
      variables.value.filter(v => v.Device.DeviceUID === selectedDevice.value?.DeviceUID) :
      []
  )

  // 动作
  async function fetchDeviceTree() {
    console.log('fetchDeviceTree: Starting...')
    loading.value = true
    error.value = null
    
    try {
      console.log('fetchDeviceTree: Calling apiService.getDeviceTree()')
      const response = await apiService.getDeviceTree()
      console.log('fetchDeviceTree: Response received:', JSON.stringify(response, null, 2))
      
      if (response.success && response.data) {
        devices.value = response.data
        lastUpdated.value = new Date()
        console.log('fetchDeviceTree: Success, devices updated:', devices.value.length, 'devices')
      } else if (response.data) {
        // 如果没有success字段但有data字段，直接使用data
        devices.value = response.data
        lastUpdated.value = new Date()
        console.log('fetchDeviceTree: Success (without success field), devices updated:', devices.value.length, 'devices')
      } else {
        error.value = response.message || '获取设备树失败'
        console.log('fetchDeviceTree: Failed with message:', error.value)
      }
    } catch (err) {
      error.value = err instanceof Error ? err.message : '网络请求失败'
      console.error('Failed to fetch device tree:', err)
    } finally {
      loading.value = false
      console.log('fetchDeviceTree: Completed, loading set to false')
    }
  }

  async function fetchVariables(deviceUID?: number) {
    loading.value = true
    error.value = null
    
    try {
      const response = deviceUID ? 
        await apiService.getDeviceVariables(deviceUID) :
        await apiService.getAllVariables()
      
      console.log('fetchVariables: Response received:', JSON.stringify(response, null, 2))
        
      if (response.success && response.data) {
        variables.value = response.data
        lastUpdated.value = new Date()
        console.log('fetchVariables: Success, variables updated:', variables.value.length, 'variables')
      } else if (response.data) {
        // 如果没有success字段但有data字段，直接使用data
        variables.value = response.data
        lastUpdated.value = new Date()
        console.log('fetchVariables: Success (without success field), variables updated:', variables.value.length, 'variables')
      } else {
        error.value = response.message || '获取变量失败'
        console.log('fetchVariables: Failed with message:', error.value)
      }
    } catch (err) {
      error.value = err instanceof Error ? err.message : '网络请求失败'
      console.error('Failed to fetch variables:', err)
    } finally {
      loading.value = false
    }
  }

  async function updateVariable(deviceUID: number, name: string, value: any) {
    try {
      const response = await apiService.updateVariable(deviceUID, name, value)
      if (response.success) {
        // 重新获取变量以更新UI
        await fetchVariables()
        return true
      } else {
        error.value = response.message || '更新变量失败'
        return false
      }
    } catch (err) {
      error.value = err instanceof Error ? err.message : '网络请求失败'
      console.error('Failed to update variable:', err)
      return false
    }
  }

  async function sendMessage(deviceUID: number, message: string) {
    try {
      const response = await apiService.sendMessage(deviceUID, message)
      if (response.success) {
        return true
      } else {
        error.value = response.message || '发送消息失败'
        return false
      }
    } catch (err) {
      error.value = err instanceof Error ? err.message : '网络请求失败'
      console.error('Failed to send message:', err)
      return false
    }
  }

  function selectDevice(device: Device | null) {
    selectedDevice.value = device
    if (device) {
      // 当选择设备时，获取其变量
      fetchVariables(device.DeviceUID)
    }
  }

  function clearError() {
    error.value = null
  }

  // 递归查找设备
  function findDeviceByUID(uid: number, nodes: DeviceTreeNode[] = devices.value): Device | null {
    for (const node of nodes) {
      if (node.DeviceUID === uid) {
        return node
      }
      if (node.children && node.children.length > 0) {
        const found = findDeviceByUID(uid, node.children)
        if (found) return found
      }
    }
    return null
  }

  return {
    // 状态
    devices,
    variables,
    selectedDevice,
    loading,
    error,
    lastUpdated,
    
    // 计算属性
    deviceCount,
    hubDevices,
    selectedDeviceVariables,
    
    // 动作
    fetchDeviceTree,
    fetchVariables,
    updateVariable,
    sendMessage,
    selectDevice,
    clearError,
    findDeviceByUID,
  }
})
