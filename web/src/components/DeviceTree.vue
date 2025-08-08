<template>
  <div>
    <n-space justify="space-between" align="center" style="margin-bottom: 16px;">
      <n-h3>设备树 ({{ deviceCount }} 个设备)</n-h3>
      <n-button-group>
        <n-button size="small" @click="refreshTree" :loading="loading">
          <template #icon>
            <n-icon><RefreshIcon /></n-icon>
          </template>
          刷新
        </n-button>
        <n-button size="small" type="primary" @click="expandAll">
          展开全部
        </n-button>
        <n-button size="small" @click="collapseAll">
          收起全部
        </n-button>
      </n-button-group>
    </n-space>

    <n-tree
      ref="treeRef"
      :data="treeData"
      :expanded-keys="expandedKeys"
      :selected-keys="selectedKeys"
      block-line
      selectable
      @update:selected-keys="handleSelectDevice"
      @update:expanded-keys="handleExpandChange"
      :render-label="renderLabel"
      :render-prefix="renderPrefix"
    />
    
    <n-divider />
    
    <!-- 设备统计 -->
    <n-space justify="space-between">
      <n-statistic label="总设备数" :value="deviceCount" />
      <n-statistic label="中枢数量" :value="getDeviceCountByRole('hub')" />
      <n-statistic label="中继数量" :value="getDeviceCountByRole('relay')" />
      <n-statistic label="节点数量" :value="getDeviceCountByRole('node')" />
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { 
  NTree, NButton, NButtonGroup, NIcon, NH3, NSpace, NDivider, 
  NStatistic, NTag, type TreeOption 
} from 'naive-ui'
import { Refresh as RefreshIcon, Server, GitNetworkOutline as RouterIcon, Desktop, Settings } from '@vicons/ionicons5'
import { useHubStore } from '@/stores/hub'
import type { DeviceTreeNode } from '@/types/api'

const hubStore = useHubStore()
const treeRef = ref()
const expandedKeys = ref<string[]>([])
const selectedKeys = ref<string[]>([])

const { devices, loading, deviceCount } = hubStore

// 将设备数据转换为Tree组件需要的格式
const treeData = computed<TreeOption[]>(() => {
  const convertToTreeData = (nodes: DeviceTreeNode[]): TreeOption[] => {
    return nodes.map(device => ({
      key: `device-${device.DeviceUID}`,
      label: device.Name || device.HardwareID || `设备 ${device.DeviceUID}`,
      children: device.children && device.children.length > 0 ? 
        convertToTreeData(device.children) : 
        undefined,
      device: device, // 存储原始设备数据
      role: device.Role,
    }))
  }
  return convertToTreeData(devices)
})

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

// 设备颜色映射
const getDeviceColor = (role: string) => {
  switch (role) {
    case 'hub': return 'success'
    case 'relay': return 'warning' 
    case 'node': return 'info'
    case 'manager': return 'error'
    default: return 'default'
  }
}

// 渲染设备标签
const renderLabel = ({ option }: { option: TreeOption }) => {
  const device = option.device as DeviceTreeNode
  return h('span', { style: 'display: flex; align-items: center; gap: 8px;' }, [
    h(NTag, { 
      size: 'small', 
      type: getDeviceColor(device.Role) as any 
    }, () => device.Role.toUpperCase()),
    h('span', option.label),
    device.LastSeen && h('span', { 
      style: 'color: #999; font-size: 12px; margin-left: 8px;' 
    }, `(${new Date(device.LastSeen).toLocaleString()})`)
  ])
}

// 渲染设备前缀图标
const renderPrefix = ({ option }: { option: TreeOption }) => {
  const device = option.device as DeviceTreeNode
  const IconComponent = getDeviceIcon(device.Role)
  return h(NIcon, { 
    size: 16,
    color: getDeviceColor(device.Role) === 'success' ? '#18a058' :
           getDeviceColor(device.Role) === 'warning' ? '#f0a020' :
           getDeviceColor(device.Role) === 'error' ? '#d03050' : '#2080f0'
  }, () => h(IconComponent))
}

// 计算指定角色的设备数量
const getDeviceCountByRole = (role: string) => {
  const countByRole = (nodes: DeviceTreeNode[], targetRole: string): number => {
    let count = 0
    for (const node of nodes) {
      if (node.Role === targetRole) count++
      if (node.children && node.children.length > 0) {
        count += countByRole(node.children, targetRole)
      }
    }
    return count
  }
  return countByRole(devices, role)
}

// 处理设备选择
const handleSelectDevice = (keys: string[]) => {
  selectedKeys.value = keys
  if (keys.length > 0) {
    const deviceKey = keys[0]
    const deviceUID = parseInt(deviceKey.replace('device-', ''))
    const device = hubStore.findDeviceByUID(deviceUID)
    hubStore.selectDevice(device)
  } else {
    hubStore.selectDevice(null)
  }
}

// 处理展开/收起
const handleExpandChange = (keys: string[]) => {
  expandedKeys.value = keys
}

// 展开所有节点
const expandAll = () => {
  const getAllKeys = (options: TreeOption[]): string[] => {
    const keys: string[] = []
    for (const option of options) {
      keys.push(option.key as string)
      if (option.children && option.children.length > 0) {
        keys.push(...getAllKeys(option.children))
      }
    }
    return keys
  }
  expandedKeys.value = getAllKeys(treeData.value)
}

// 收起所有节点
const collapseAll = () => {
  expandedKeys.value = []
}

// 刷新树数据
const refreshTree = async () => {
  await hubStore.fetchDeviceTree()
}

// 组件挂载时获取数据
onMounted(() => {
  refreshTree()
})
</script>

<style scoped>
.n-tree {
  max-height: 60vh;
  overflow-y: auto;
}
</style>
