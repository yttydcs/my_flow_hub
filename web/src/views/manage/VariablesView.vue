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
          <n-button type="primary" @click="showAddModal = true">
            <template #icon>
              <n-icon><AddIcon /></n-icon>
            </template>
            添加变量
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

    <!-- 添加变量弹窗 -->
    <n-modal v-model:show="showAddModal" preset="dialog" title="添加变量">
      <n-form :model="newVariable" label-placement="left" :label-width="100">
        <n-form-item label="变量名" required>
          <n-input v-model:value="newVariable.name" placeholder="变量名称" />
        </n-form-item>
        <n-form-item label="变量值" required>
          <n-input v-model:value="newVariable.value" placeholder="初始值" />
        </n-form-item>
        <n-form-item label="所属设备" required>
          <n-select v-model:value="newVariable.deviceId" :options="deviceOptions" />
        </n-form-item>
      </n-form>
      <template #action>
        <n-space>
          <n-button @click="showAddModal = false">取消</n-button>
          <n-button type="primary" @click="handleAddVariable">添加</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 编辑变量弹窗 -->
    <n-modal v-model:show="showEditModal" preset="dialog" title="编辑变量">
      <n-form :model="editVariable" label-placement="left" :label-width="100">
        <n-form-item label="变量名" required>
          <n-input v-model:value="editVariable.name" placeholder="变量名称" />
        </n-form-item>
        <n-form-item label="变量值" required>
          <n-input v-model:value="editVariable.value" placeholder="变量值" />
        </n-form-item>
        <n-form-item label="所属设备" required>
          <n-select v-model:value="editVariable.deviceId" :options="deviceOptions" />
        </n-form-item>
      </n-form>
      <template #action>
        <n-space>
          <n-button @click="showEditModal = false">取消</n-button>
          <n-button type="primary" @click="handleUpdateVariable">保存</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { useHubStore } from '@/stores/hub'
import { useMessage } from 'naive-ui'
import { apiService } from '@/services/api'
import {
  NH2, NSpace, NInput, NSelect, NButton, NIcon, NDataTable,
  NModal, NForm, NFormItem, NPopconfirm
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import {
  Search as SearchIcon,
  Refresh as RefreshIcon,
  Add as AddIcon,
  Create as EditIcon,
  Trash as DeleteIcon
} from '@vicons/ionicons5'

const hubStore = useHubStore()
const message = useMessage()

const loading = ref(false)
const searchKeyword = ref('')
const selectedDevice = ref('')
const showAddModal = ref(false)
const showEditModal = ref(false)

// 新变量表单
const newVariable = ref({
  name: '',
  value: '',
  deviceId: ''
})

// 编辑变量表单
const editVariable = ref({
  id: 0,
  name: '',
  value: '',
  deviceId: ''
})

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
  },
  {
    title: '操作',
    key: 'actions',
    width: 150,
    render(row: any) {
      return h(NSpace, null, {
        default: () => [
          h(NButton, { 
            size: 'small', 
            onClick: () => handleEditVariable(row) 
          }, { 
            default: () => '编辑',
            icon: () => h(NIcon, null, { default: () => h(EditIcon) })
          }),
          h(NPopconfirm, {
            onPositiveClick: () => handleDeleteVariable(row.ID)
          }, {
            default: () => '确认删除这个变量吗？',
            trigger: () => h(NButton, { 
              size: 'small', 
              type: 'error'
            }, { 
              default: () => '删除',
              icon: () => h(NIcon, null, { default: () => h(DeleteIcon) })
            })
          })
        ]
      })
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

// 添加变量
const handleAddVariable = async () => {
  try {
    const response = await apiService.createVariable({
      name: newVariable.value.name,
      value: newVariable.value.value,
      deviceId: parseInt(newVariable.value.deviceId)
    })
    
    if (response.success) {
      message.success('变量添加成功')
      showAddModal.value = false
      // 重置表单
      newVariable.value = {
        name: '',
        value: '',
        deviceId: ''
      }
      // 刷新数据
      refreshData()
    } else {
      message.error(response.message || '添加变量失败')
    }
  } catch (error) {
    console.error('添加变量失败:', error)
    message.error('添加变量失败: ' + (error instanceof Error ? error.message : '未知错误'))
  }
}

// 编辑变量
const handleEditVariable = (variable: any) => {
  editVariable.value = {
    id: variable.ID,
    name: variable.VariableName,
    value: String(variable.Value),
    deviceId: variable.OwnerDeviceID.toString()
  }
  showEditModal.value = true
}

// 更新变量
const handleUpdateVariable = async () => {
  try {
    const response = await apiService.updateVariableNew({
      id: editVariable.value.id,
      name: editVariable.value.name,
      value: editVariable.value.value
    })
    
    if (response.success) {
      message.success('变量更新成功')
      showEditModal.value = false
      // 刷新数据
      refreshData()
    } else {
      message.error(response.message || '更新变量失败')
    }
  } catch (error) {
    console.error('更新变量失败:', error)
    message.error('更新变量失败: ' + (error instanceof Error ? error.message : '未知错误'))
  }
}

// 删除变量
const handleDeleteVariable = async (variableId: number) => {
  try {
    const response = await apiService.deleteVariable(variableId)
    
    if (response.success) {
      message.success('变量删除成功')
      // 刷新数据
      refreshData()
    } else {
      message.error(response.message || '删除变量失败')
    }
  } catch (error) {
    console.error('删除变量失败:', error)
    message.error('删除变量失败: ' + (error instanceof Error ? error.message : '未知错误'))
  }
}

onMounted(() => {
  refreshData()
})
</script>

<style scoped>
</style>
