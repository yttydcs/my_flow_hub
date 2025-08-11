<template>
  <div>
    <n-h2>设备管理</n-h2>
    
    <n-space vertical size="large">
      <!-- 操作栏 -->
      <n-space justify="space-between">
        <n-space>
          <n-input 
            v-model:value="searchKeyword" 
            placeholder="搜索设备..." 
            clearable
            style="width: 300px;"
          >
            <template #prefix>
              <n-icon><SearchIcon /></n-icon>
            </template>
          </n-input>
          <n-select
            v-model:value="selectedRole"
            placeholder="设备角色"
            :options="roleOptions"
            clearable
            style="width: 150px;"
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
            添加设备
          </n-button>
        </n-space>
      </n-space>

      <!-- 设备列表 -->
      <n-data-table
        :columns="columns"
        :data="filteredDevices"
        :loading="loading"
        :pagination="pagination"
        :row-key="(row: any) => row.ID"
      />
    </n-space>

    <!-- 添加设备弹窗 -->
    <n-modal v-model:show="showAddModal" preset="dialog" title="添加设备">
      <n-form :model="newDevice" label-placement="left" :label-width="100">
        <n-form-item label="硬件ID" required>
          <n-input v-model:value="newDevice.hardwareId" placeholder="设备硬件ID" />
        </n-form-item>
        <n-form-item label="设备名称">
          <n-input v-model:value="newDevice.name" placeholder="设备显示名称" />
        </n-form-item>
        <n-form-item label="设备角色" required>
          <n-select v-model:value="newDevice.role" :options="roleOptions" />
        </n-form-item>
        <n-form-item label="父设备">
          <n-select 
            v-model:value="newDevice.parentId" 
            :options="parentOptions"
            placeholder="选择父设备（可选）"
            clearable
          />
        </n-form-item>
      </n-form>
      <template #action>
        <n-space>
          <n-button @click="showAddModal = false">取消</n-button>
          <n-button type="primary" @click="handleAddDevice">添加</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 编辑设备弹窗 -->
    <n-modal v-model:show="showEditModal" preset="dialog" title="编辑设备">
      <n-form :model="editDevice" label-placement="left" :label-width="100">
        <n-form-item label="硬件ID">
          <n-input v-model:value="editDevice.hardwareId" disabled />
        </n-form-item>
        <n-form-item label="设备名称">
          <n-input v-model:value="editDevice.name" placeholder="设备显示名称" />
        </n-form-item>
        <n-form-item label="设备角色">
          <n-select v-model:value="editDevice.role" :options="roleOptions" />
        </n-form-item>
        <n-form-item label="父设备">
          <n-select 
            v-model:value="editDevice.parentId" 
            :options="parentOptions"
            placeholder="选择父设备（可选）"
            clearable
          />
        </n-form-item>
      </n-form>
      <template #action>
        <n-space>
          <n-button @click="showEditModal = false">取消</n-button>
          <n-button type="primary" @click="handleUpdateDevice">更新</n-button>
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
  NModal, NForm, NFormItem, NBadge, NTag, NPopconfirm
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import {
  Search as SearchIcon,
  Refresh as RefreshIcon,
  Add as AddIcon,
  Trash as DeleteIcon,
  Create as EditIcon
} from '@vicons/ionicons5'

const hubStore = useHubStore()
const message = useMessage()

const loading = ref(false)
const searchKeyword = ref('')
const selectedRole = ref('')
const showAddModal = ref(false)
const showEditModal = ref(false)

// 新设备表单
const newDevice = ref({
  hardwareId: '',
  name: '',
  role: 'node',
  parentId: null
})

// 编辑设备表单
const editDevice = ref({
  id: 0,
  hardwareId: '',
  name: '',
  role: 'node',
  parentId: null
})

// 设备角色选项
const roleOptions = [
  { label: '普通节点', value: 'node' },
  { label: '中继节点', value: 'relay' },
  { label: '中枢节点', value: 'hub' },
  { label: '管理器', value: 'manager' }
]

// 父设备选项
const parentOptions = computed(() => {
  return hubStore.devices
    .filter(device => device.Role === 'hub' || device.Role === 'relay')
    .map(device => ({
      label: device.Name || device.HardwareID,
      value: device.ID
    }))
})

// 过滤后的设备列表
const filteredDevices = computed(() => {
  let devices = hubStore.devices || []
  
  if (searchKeyword.value) {
    devices = devices.filter(device => 
      (device.Name || '').toLowerCase().includes(searchKeyword.value.toLowerCase()) ||
      device.HardwareID.toLowerCase().includes(searchKeyword.value.toLowerCase())
    )
  }
  
  if (selectedRole.value) {
    devices = devices.filter(device => device.Role === selectedRole.value)
  }
  
  return devices
})

// 分页配置
const pagination = ref({
  pageSize: 10
})

// 表格列配置
const columns: DataTableColumns = [
  {
    title: 'ID',
    key: 'ID',
    width: 80
  },
  {
    title: '硬件ID',
    key: 'HardwareID',
    width: 200
  },
  {
    title: '设备名称',
    key: 'Name',
    render(row: any) {
      return row.Name || '未命名设备'
    }
  },
  {
    title: '角色',
    key: 'Role',
    width: 120,
    render(row: any) {
      const roleMap: Record<string, { type: string, text: string }> = {
        'node': { type: 'default', text: '普通节点' },
        'relay': { type: 'warning', text: '中继节点' },
        'hub': { type: 'success', text: '中枢节点' },
        'manager': { type: 'error', text: '管理器' }
      }
      const config = roleMap[row.Role] || { type: 'default', text: '未知' }
      return h(NTag, { type: config.type as any }, { default: () => config.text })
    }
  },
  {
    title: '父设备',
    key: 'ParentID',
    width: 120,
    render(row: any) {
      return row.ParentID ? `#${row.ParentID}` : '-'
    }
  },
  {
    title: '最后在线',
    key: 'LastSeen',
    width: 180,
    render(row: any) {
      return row.LastSeen ? new Date(row.LastSeen).toLocaleString() : '从未在线'
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
            onClick: () => handleEditDevice(row) 
          }, { 
            default: () => '编辑',
            icon: () => h(NIcon, null, { default: () => h(EditIcon) })
          }),
          h(NPopconfirm, {
            onPositiveClick: () => handleDeleteDevice(row.ID)
          }, {
            default: () => '确认删除这个设备吗？',
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
    await hubStore.fetchDeviceTree()
    message.success('设备列表刷新成功')
  } catch (error) {
    message.error('刷新设备列表失败')
  } finally {
    loading.value = false
  }
}

// 添加设备
const handleAddDevice = async () => {
  try {
    const response = await apiService.createDevice({
      hardwareId: newDevice.value.hardwareId,
      name: newDevice.value.name,
      role: newDevice.value.role,
      parentId: newDevice.value.parentId || undefined
    })
    
    if (response.success) {
      message.success('设备添加成功')
      showAddModal.value = false
      // 重置表单
      newDevice.value = {
        hardwareId: '',
        name: '',
        role: 'node',
        parentId: null
      }
      // 刷新数据
      refreshData()
    } else {
      message.error(response.message || '添加设备失败')
    }
  } catch (error) {
    console.error('添加设备失败:', error)
    message.error('添加设备失败: ' + (error instanceof Error ? error.message : '未知错误'))
  }
}

// 编辑设备
const handleEditDevice = (device: any) => {
  editDevice.value = {
    id: device.ID,
    hardwareId: device.HardwareID,
    name: device.Name || '',
    role: device.Role,
    parentId: device.ParentID || null
  }
  showEditModal.value = true
}

// 更新设备
const handleUpdateDevice = async () => {
  try {
    const response = await apiService.updateDevice({
      id: editDevice.value.id,
      name: editDevice.value.name,
      role: editDevice.value.role,
      parentId: editDevice.value.parentId || undefined
    })
    
    if (response.success) {
      message.success('设备更新成功')
      showEditModal.value = false
      // 刷新数据
      refreshData()
    } else {
      message.error(response.message || '更新设备失败')
    }
  } catch (error) {
    console.error('更新设备失败:', error)
    message.error('更新设备失败: ' + (error instanceof Error ? error.message : '未知错误'))
  }
}

// 删除设备
const handleDeleteDevice = async (deviceId: number) => {
  try {
    const response = await apiService.deleteDevice(deviceId)
    
    if (response.success) {
      message.success('设备删除成功')
      // 刷新数据
      refreshData()
    } else {
      message.error(response.message || '删除设备失败')
    }
  } catch (error) {
    console.error('删除设备失败:', error)
    message.error('删除设备失败: ' + (error instanceof Error ? error.message : '未知错误'))
  }
}

onMounted(() => {
  refreshData()
})
</script>

<style scoped>
</style>
