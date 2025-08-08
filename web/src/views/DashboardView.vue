<template>
  <n-layout style="height: 100vh">
    <!-- 头部导航 -->
    <n-layout-header 
      style="height: 64px; padding: 0 24px; display: flex; align-items: center; justify-content: space-between;" 
      bordered
    >
      <div style="display: flex; align-items: center; gap: 16px;">
        <n-icon size="32" color="#18a058"><ServerIcon /></n-icon>
        <div>
          <div style="font-size: 1.5rem; font-weight: bold;">MyFlowHub</div>
          <div style="font-size: 0.8rem; color: #999;">设备管理中心</div>
        </div>
      </div>
      
      <n-space>
        <n-tag :type="connectionStatus.type" size="large">
          <template #icon>
            <n-icon><component :is="connectionStatus.icon" /></n-icon>
          </template>
          {{ connectionStatus.text }}
        </n-tag>
        <n-statistic label="设备数量" :value="deviceCount" />
        <n-statistic label="变量数量" :value="variables.length" />
        <n-button @click="handleRefreshAll" :loading="loading" type="primary">
          <template #icon>
            <n-icon><RefreshIcon /></n-icon>
          </template>
          刷新全部
        </n-button>
      </n-space>
    </n-layout-header>

    <!-- 主体布局 -->
    <n-layout has-sider>
      <!-- 左侧设备树 -->
      <n-layout-sider
        bordered
        collapse-mode="width"
        :collapsed-width="64"
        :width="320"
        show-trigger
        content-style="padding: 16px;"
      >
        <DeviceTree />
      </n-layout-sider>

      <!-- 主要内容区域 -->
      <n-layout has-sider>
        <!-- 中间设备详情 -->
        <n-layout-content content-style="padding: 24px;">
          <DeviceDetails />
        </n-layout-content>

        <!-- 右侧变量管理 -->
        <n-layout-sider
          bordered
          collapse-mode="width"
          :collapsed-width="0"
          :width="400"
          show-trigger
          content-style="padding: 16px;"
        >
          <VariableManager />
        </n-layout-sider>
      </n-layout>
    </n-layout>

    <!-- 错误提示 -->
    <n-message-provider>
      <n-alert 
        v-if="error" 
        type="error" 
        :title="'错误'" 
        closable
        @close="clearError"
        style="position: fixed; top: 80px; right: 20px; z-index: 1000; max-width: 400px;"
      >
        {{ error }}
      </n-alert>
    </n-message-provider>

    <!-- 加载指示器 -->
    <n-back-top :right="40" />
  </n-layout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, watch } from 'vue'
import { 
  NLayout, NLayoutHeader, NLayoutSider, NLayoutContent, 
  NSpace, NButton, NIcon, NTag, NStatistic, NAlert, 
  NMessageProvider, NBackTop
} from 'naive-ui'
import { 
  Server as ServerIcon, 
  Refresh as RefreshIcon, 
  CheckmarkCircle as ConnectedIcon, 
  CloseCircle as DisconnectedIcon 
} from '@vicons/ionicons5'
import { useHubStore } from '@/stores/hub'
import { useMessage } from 'naive-ui'
import DeviceTree from '@/components/DeviceTree.vue'
import DeviceDetails from '@/components/DeviceDetails.vue'
import VariableManager from '@/components/VariableManager.vue'

const message = useMessage()
const hubStore = useHubStore()

const {
  devices,
  variables,
  selectedDevice,
  loading,
  error,
  deviceCount,
  lastUpdated,
  clearError
} = hubStore

let refreshInterval: number | null = null

// 连接状态
const connectionStatus = computed(() => {
  console.log('Connection status check:', {
    lastUpdated: lastUpdated,
    loading: loading,
    error: error,
    deviceCount: deviceCount
  })
  
  // 如果正在加载，显示连接中
  if (loading) {
    return {
      type: 'warning' as const,
      icon: RefreshIcon,
      text: '连接中...'
    }
  }
  
  // 如果有错误，显示连接失败
  if (error) {
    return {
      type: 'error' as const,
      icon: DisconnectedIcon,
      text: '连接失败'
    }
  }
  
  // 如果有数据（设备数量大于0），认为已连接
  if (deviceCount > 0) {
    return {
      type: 'success' as const,
      icon: ConnectedIcon,
      text: '已连接'
    }
  }
  
  // 如果最近30秒内有更新，认为已连接
  if (lastUpdated && (Date.now() - lastUpdated.getTime()) < 30000) {
    return {
      type: 'success' as const,
      icon: ConnectedIcon,
      text: '已连接'
    }
  }
  
  // 默认未连接状态
  return {
    type: 'error' as const,
    icon: DisconnectedIcon,
    text: '未连接'
  }
})

// 刷新所有数据
const handleRefreshAll = async () => {
  console.log('handleRefreshAll: Starting...')
  try {
    await Promise.all([
      hubStore.fetchDeviceTree(),
      hubStore.fetchVariables()
    ])
    message.success('数据刷新完成')
    console.log('handleRefreshAll: All requests completed successfully')
  } catch (err) {
    const errorMessage = '刷新失败：' + (err instanceof Error ? err.message : '未知错误')
    message.error(errorMessage)
    console.error('handleRefreshAll: Error occurred:', err)
  }
}

// 监听错误变化，自动显示消息
watch(() => error, (newError) => {
  if (newError) {
    message.error(newError)
  }
})

// 监听设备选择变化
watch(() => selectedDevice, (newDevice) => {
  if (newDevice) {
    // 当选择新设备时，可以执行一些操作
    console.log('Selected device:', newDevice.Name || newDevice.HardwareID)
  }
})

// 组件挂载时初始化数据
onMounted(async () => {
  console.log('DashboardView: onMounted called')
  await handleRefreshAll()
  
  // 设置定期刷新
  refreshInterval = window.setInterval(async () => {
    if (!loading) {
      await hubStore.fetchVariables()
    }
  }, 30000) // 每30秒刷新一次变量数据
  
  console.log('DashboardView: Refresh interval set')
})

// 组件卸载时清理定时器
onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
})
</script>

<style scoped>
.n-layout-header {
  background-color: white;
  border-bottom: 1px solid #e0e0e0;
}

.n-layout-sider {
  background-color: #fafafa;
}

.n-layout-content {
  background-color: #f5f5f5;
}

:deep(.n-layout-sider .n-layout-sider-scroll-container) {
  display: flex;
  flex-direction: column;
}

:deep(.n-back-top) {
  z-index: 999;
}
</style>
