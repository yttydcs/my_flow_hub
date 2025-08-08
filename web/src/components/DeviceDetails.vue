<template>
  <n-card title="设备详情" v-if="selectedDevice">
    <template #header-extra>
      <n-tag :type="getDeviceTagType(selectedDevice.Role)" size="small">
        {{ selectedDevice.Role.toUpperCase() }}
      </n-tag>
    </template>

    <!-- 基本信息 -->
    <n-descriptions label-placement="left" bordered :column="1" style="margin-bottom: 24px;">
      <n-descriptions-item label="设备ID">
        <n-tag type="info">{{ selectedDevice.DeviceUID }}</n-tag>
      </n-descriptions-item>
      <n-descriptions-item label="硬件ID">
        <n-text code>{{ selectedDevice.HardwareID }}</n-text>
      </n-descriptions-item>
      <n-descriptions-item label="设备名称">
        {{ selectedDevice.Name || '未设置' }}
      </n-descriptions-item>
      <n-descriptions-item label="设备角色">
        <n-tag :type="getDeviceTagType(selectedDevice.Role)">
          {{ getRoleDisplayName(selectedDevice.Role) }}
        </n-tag>
      </n-descriptions-item>
      <n-descriptions-item label="父设备" v-if="selectedDevice.Parent">
        <n-space>
          <n-tag type="default">{{ selectedDevice.Parent.Name || selectedDevice.Parent.HardwareID }}</n-tag>
          <n-text depth="3">({{ selectedDevice.Parent.Role.toUpperCase() }})</n-text>
        </n-space>
      </n-descriptions-item>
      <n-descriptions-item label="子设备数量">
        <n-tag type="info">{{ (selectedDevice.children || []).length }} 个</n-tag>
      </n-descriptions-item>
      <n-descriptions-item label="最后在线">
        <n-time v-if="selectedDevice.LastSeen" :value="new Date(selectedDevice.LastSeen).getTime()" />
        <n-text depth="3" v-else>从未连接</n-text>
      </n-descriptions-item>
      <n-descriptions-item label="创建时间">
        <n-time :value="new Date(selectedDevice.CreatedAt).getTime()" />
      </n-descriptions-item>
      <n-descriptions-item label="更新时间">
        <n-time :value="new Date(selectedDevice.UpdatedAt).getTime()" />
      </n-descriptions-item>
    </n-descriptions>

    <!-- 子设备列表 -->
    <n-divider title-placement="left">子设备</n-divider>
    <div v-if="(selectedDevice.children || []).length > 0">
      <n-space>
        <n-card 
          v-for="child in selectedDevice.children" 
          :key="child.DeviceUID"
          size="small"
          hoverable
          @click="selectChild(child)"
          style="cursor: pointer; min-width: 200px;"
        >
          <template #header>
            <n-space align="center">
              <n-icon :component="getDeviceIcon(child.Role)" />
              <span>{{ child.Name || child.HardwareID }}</span>
            </n-space>
          </template>
          <template #header-extra>
            <n-tag :type="getDeviceTagType(child.Role)" size="small">
              {{ child.Role.toUpperCase() }}
            </n-tag>
          </template>
          <n-text depth="3">设备ID: {{ child.DeviceUID }}</n-text>
        </n-card>
      </n-space>
    </div>
    <n-empty v-else description="无子设备" size="small" />

    <!-- 设备变量 -->
    <n-divider title-placement="left">设备变量 ({{ selectedDeviceVariables.length }})</n-divider>
    <div v-if="selectedDeviceVariables.length > 0">
      <n-space vertical>
        <n-card 
          v-for="variable in selectedDeviceVariables" 
          :key="variable.ID"
          size="small"
          embedded
        >
          <template #header>
            <n-space align="center">
              <n-tag type="info" size="small">{{ variable.VariableName }}</n-tag>
              <n-text depth="3">{{ typeof variable.Value }}</n-text>
            </n-space>
          </template>
          <template #header-extra>
            <n-button size="tiny" @click="editVariable(variable)">
              <template #icon>
                <n-icon><EditIcon /></n-icon>
              </template>
              编辑
            </n-button>
          </template>
          
          <n-text code style="display: block; white-space: pre-wrap;">{{ 
            formatVariableValue(variable.Value) 
          }}</n-text>
          
          <template #footer>
            <n-text depth="3" style="font-size: 12px;">
              更新于: {{ new Date(variable.UpdatedAt).toLocaleString() }}
            </n-text>
          </template>
        </n-card>
      </n-space>
    </div>
    <n-empty v-else description="暂无变量" size="small">
      <template #extra>
        <n-button size="small" type="primary">添加变量</n-button>
      </template>
    </n-empty>

    <!-- 操作按钮 -->
    <n-divider />
    <n-space>
      <n-button type="primary" @click="refreshDeviceData">
        <template #icon>
          <n-icon><RefreshIcon /></n-icon>
        </template>
        刷新数据
      </n-button>
      <n-button @click="sendTestMessage" :loading="sendingMessage">
        <template #icon>
          <n-icon><SendIcon /></n-icon>
        </template>
        发送测试消息
      </n-button>
      <n-button secondary @click="showDeviceJson = true">
        <template #icon>
          <n-icon><CodeIcon /></n-icon>
        </template>
        查看JSON
      </n-button>
    </n-space>

    <!-- JSON查看模态框 -->
    <n-modal v-model:show="showDeviceJson">
      <n-card 
        style="width: 80vw; max-height: 80vh; overflow: auto;" 
        title="设备JSON数据"
        closable
        @close="showDeviceJson = false"
      >
        <n-code :code="JSON.stringify(selectedDevice, null, 2)" language="json" />
      </n-card>
    </n-modal>
  </n-card>

  <!-- 未选择设备 -->
  <n-card v-else>
    <n-empty description="请在左侧选择一个设备以查看详情" />
  </n-card>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { 
  NCard, NDescriptions, NDescriptionsItem, NTag, NText, NTime, 
  NSpace, NIcon, NEmpty, NButton, NDivider, NModal, NCode,
  useMessage
} from 'naive-ui'
import { 
  Refresh as RefreshIcon, 
  Send as SendIcon, 
  Create as EditIcon,
  Code as CodeIcon,
  Server, 
  GitNetworkOutline as RouterIcon, 
  Desktop, 
  Settings 
} from '@vicons/ionicons5'
import { useHubStore } from '@/stores/hub'
import type { DeviceVariable, Device } from '@/types/api'

const message = useMessage()
const hubStore = useHubStore()

const { selectedDevice, selectedDeviceVariables } = hubStore

const sendingMessage = ref(false)
const showDeviceJson = ref(false)

// 设备角色显示名称映射
const getRoleDisplayName = (role: string) => {
  switch (role) {
    case 'hub': return '中枢'
    case 'relay': return '中继'
    case 'node': return '节点'
    case 'manager': return '管理器'
    default: return '未知'
  }
}

// 设备标签类型映射
const getDeviceTagType = (role: string) => {
  switch (role) {
    case 'hub': return 'success'
    case 'relay': return 'warning'
    case 'node': return 'info'
    case 'manager': return 'error'
    default: return 'default'
  }
}

// 设备图标映射
const getDeviceIcon = (role: string) => {
  switch (role) {
    case 'hub': return Server
    case 'relay': return RouterIcon
    case 'node': return Desktop
    case 'manager': return Settings
    default: return Desktop
  }
}

// 格式化变量值显示
const formatVariableValue = (value: any): string => {
  if (value === null || value === undefined) {
    return 'null'
  }
  if (typeof value === 'object') {
    return JSON.stringify(value, null, 2)
  }
  return String(value)
}

// 选择子设备
const selectChild = (child: Device) => {
  hubStore.selectDevice(child)
}

// 编辑变量
const editVariable = (variable: DeviceVariable) => {
  // 这里可以触发一个事件或者打开编辑模态框
  // 由于编辑功能在 VariableManager 组件中，这里可以发出事件
  console.log('Edit variable:', variable)
}

// 刷新设备数据
const refreshDeviceData = async () => {
  if (!selectedDevice) return
  
  // 刷新设备树和变量
  await Promise.all([
    hubStore.fetchDeviceTree(),
    hubStore.fetchVariables(selectedDevice.DeviceUID)
  ])
  
  message.success('数据已刷新')
}

// 发送测试消息
const sendTestMessage = async () => {
  if (!selectedDevice) return
  
  sendingMessage.value = true
  try {
    const success = await hubStore.sendMessage(
      selectedDevice.DeviceUID, 
      `测试消息 - ${new Date().toISOString()}`
    )
    
    if (success) {
      message.success('测试消息发送成功')
    } else {
      message.error('发送失败：' + (hubStore.error || '未知错误'))
    }
  } finally {
    sendingMessage.value = false
  }
}
</script>

<style scoped>
.n-card {
  margin-bottom: 16px;
}

:deep(.n-descriptions-table-content) {
  word-break: break-all;
}
</style>
