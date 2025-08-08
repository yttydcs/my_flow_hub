<template>
  <div>
    <n-h2>变量管理</n-h2>
    
    <n-space vertical size="large">
      <!-- 操作栏 -->
      <n-space justify="space-between">
        <n-space>
          <n-input 
            v-model:value="searchKeyword" 
            placeholder="搜索变量..." 
            clearable
            style="width: 300px;"
          >
            <template #prefix>
              <n-icon><SearchIcon /></n-icon>
            </template>
          </n-input>
          <n-select
            v-model:value="selectedDevice"
            placeholder="选择设备"
            :options="deviceOptions"
            clearable
            style="width: 200px;"
          />
        </n-space>
        <n-space>
          <n-button @click="refreshData" :loading="loading">
            <template #icon>
              <n-icon><RefreshIcon /></n-icon>
            </template>
            刷新
          </n-button>
        </n-space>
      </n-space>

      <!-- 变量列表 -->
      <n-data-table
        :columns="columns"
        :data="filteredVariables"
        :loading="loading"
        :pagination="pagination"
        :row-key="(row: any) => row.ID"
      />
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useHubStore } from '@/stores/hub'
import { useMessage } from 'naive-ui'
import {
  NH2, NSpace, NInput, NSelect, NButton, NIcon, NDataTable
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import {
  Search as SearchIcon,
  Refresh as RefreshIcon
} from '@vicons/ionicons5'

const hubStore = useHubStore()
const message = useMessage()

const loading = ref(false)
const searchKeyword = ref('')
const selectedDevice = ref('')

// 设备选项
const deviceOptions = computed(() => {
  return hubStore.devices.map(device => ({
    label: device.Name || device.HardwareID,
    value: device.ID.toString()
  }))
})

// 过滤后的变量列表
const filteredVariables = computed(() => {
  let variables = hubStore.variables || []
  
  if (searchKeyword.value) {
    variables = variables.filter(variable => 
      variable.VariableName.toLowerCase().includes(searchKeyword.value.toLowerCase())
    )
  }
  
  if (selectedDevice.value) {
    variables = variables.filter(variable => 
      variable.OwnerDeviceID.toString() === selectedDevice.value
    )
  }
  
  return variables
})

// 分页配置
const pagination = ref({
  pageSize: 15
})

// 表格列配置
const columns: DataTableColumns = [
  {
    title: 'ID',
    key: 'ID',
    width: 80
  },
  {
    title: '变量名',
    key: 'VariableName',
    width: 200
  },
  {
    title: '当前值',
    key: 'Value',
    render(row: any) {
      return String(row.Value)
    }
  },
  {
    title: '所属设备',
    key: 'OwnerDeviceID',
    width: 150,
    render(row: any) {
      const device = hubStore.devices.find(d => d.ID === row.OwnerDeviceID)
      return device ? (device.Name || device.HardwareID) : `设备#${row.OwnerDeviceID}`
    }
  },
  {
    title: '更新时间',
    key: 'UpdatedAt',
    width: 180,
    render(row: any) {
      return new Date(row.UpdatedAt).toLocaleString()
    }
  }
]

// 刷新数据
const refreshData = async () => {
  loading.value = true
  try {
    await Promise.all([
      hubStore.fetchDeviceTree(),
      hubStore.fetchVariables()
    ])
    message.success('变量列表刷新成功')
  } catch (error) {
    message.error('刷新变量列表失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  refreshData()
})
</script>

<style scoped>
</style>
