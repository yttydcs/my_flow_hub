<template>
  <div style="height: calc(100vh - 60px); overflow: hidden;">
    <!-- 状态卡片 -->
    <n-space style="padding: 24px; padding-bottom: 16px;" size="large">
      <n-card style="min-width: 150px;">
        <n-statistic label="设备数量" :value="deviceCount" />
      </n-card>
      
      <n-card style="min-width: 150px;">
        <n-statistic label="变量数量" :value="variables.length" />
      </n-card>
      
      <n-card style="min-width: 200px;">
        <n-statistic label="最后更新">
          {{ lastUpdated ? lastUpdated.toLocaleString() : '未更新' }}
        </n-statistic>
      </n-card>
    </n-space>

    <!-- 设备树区域 -->
    <div style="height: calc(100% - 120px); padding: 0 24px 24px;">
      <n-card style="height: 100%;">
        <template #header>
          <n-space align="center">
            <n-icon size="20"><TreeIcon /></n-icon>
            <span>设备树</span>
          </n-space>
        </template>
        <div style="height: calc(100% - 60px); overflow-y: auto; padding: 16px;">
          <DeviceTree />
        </div>
      </n-card>
    </div>

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
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, watch } from 'vue'
import { 
  NSpace, NIcon, NStatistic, NAlert, 
  NMessageProvider, NCard
} from 'naive-ui'
import { 
  GitNetwork as TreeIcon
} from '@vicons/ionicons5'
import { useHubStore } from '@/stores/hub'
import { useMessage } from 'naive-ui'
import DeviceTree from '@/components/DeviceTree.vue'

const message = useMessage()
const hubStore = useHubStore()

const {
  devices,
  variables,
  deviceCount,
  lastUpdated,
  error,
  clearError
} = hubStore

let pollingInterval: number | null = null

// 轮询数据更新
const startPolling = () => {
  // 立即执行一次
  hubStore.fetchDeviceTree()
  hubStore.fetchVariables()
  
  // 设置定时轮询
  pollingInterval = window.setInterval(async () => {
    try {
      await hubStore.fetchDeviceTree()
      await hubStore.fetchVariables()
    } catch (err) {
      console.error('Polling error:', err)
    }
  }, 1000) // 1秒轮询
}

// 停止轮询
const stopPolling = () => {
  if (pollingInterval) {
    clearInterval(pollingInterval)
    pollingInterval = null
  }
}

// 监听错误变化，自动显示消息
watch(() => error, (newError) => {
  if (newError) {
    message.error(newError)
  }
})

// 组件挂载时开始轮询
onMounted(() => {
  console.log('DashboardView: Starting polling')
  startPolling()
})

// 组件卸载时停止轮询
onUnmounted(() => {
  console.log('DashboardView: Stopping polling')
  stopPolling()
})
</script>

<<style scoped>
:deep(.n-card .n-card-header) {
  padding-bottom: 8px;
}

:deep(.n-statistic) {
  text-align: center;
}
</style>
