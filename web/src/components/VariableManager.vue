<template>
  <n-card title="设备变量管理" :loading="loading">
    <template #header-extra>
      <n-space>
        <n-button size="small" @click="refreshVariables" :loading="loading">
          <template #icon>
            <n-icon><RefreshIcon /></n-icon>
          </template>
          刷新
        </n-button>
        <n-button size="small" type="primary" @click="showAddModal = true">
          <template #icon>
            <n-icon><AddIcon /></n-icon>
          </template>
          添加变量
        </n-button>
      </n-space>
    </template>

    <!-- 变量列表 -->
    <n-data-table
      :columns="columns"
      :data="variables"
      :pagination="paginationProps"
      :row-key="(row) => row.ID"
    />

    <!-- 添加变量模态框 -->
    <n-modal v-model:show="showAddModal" preset="dialog">
      <template #header>添加新变量</template>
      <n-form ref="formRef" :model="newVariable" :rules="rules">
        <n-form-item label="设备" path="deviceUID">
          <n-select
            v-model:value="newVariable.deviceUID"
            :options="deviceOptions"
            placeholder="选择设备"
          />
        </n-form-item>
        <n-form-item label="变量名" path="name">
          <n-input v-model:value="newVariable.name" placeholder="输入变量名" />
        </n-form-item>
        <n-form-item label="变量值" path="value">
          <n-input v-model:value="newVariable.value" placeholder="输入变量值" />
        </n-form-item>
        <n-form-item label="数据类型">
          <n-select
            v-model:value="newVariable.type"
            :options="typeOptions"
            placeholder="选择数据类型"
          />
        </n-form-item>
      </n-form>
      <template #action>
        <n-space>
          <n-button @click="showAddModal = false">取消</n-button>
          <n-button type="primary" @click="handleAddVariable" :loading="submitting">
            确定
          </n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 编辑变量模态框 -->
    <n-modal v-model:show="showEditModal" preset="dialog">
      <template #header>编辑变量</template>
      <n-form ref="editFormRef" :model="editingVariable" :rules="editRules">
        <n-form-item label="变量名">
          <n-input v-model:value="editingVariable.VariableName" disabled />
        </n-form-item>
        <n-form-item label="当前值" path="Value">
          <n-input 
            v-model:value="editingVariable.Value" 
            placeholder="输入新值"
            :type="getInputType(editingVariable.Value)"
          />
        </n-form-item>
      </n-form>
      <template #action>
        <n-space>
          <n-button @click="showEditModal = false">取消</n-button>
          <n-button type="primary" @click="handleUpdateVariable" :loading="submitting">
            更新
          </n-button>
        </n-space>
      </template>
    </n-modal>
  </n-card>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { 
  NCard, NDataTable, NButton, NIcon, NSpace, NModal, NForm, 
  NFormItem, NInput, NSelect, NTag, NTime, useMessage,
  type DataTableColumns, type FormInst, type FormRules
} from 'naive-ui'
import { Refresh as RefreshIcon, Add as AddIcon, Create as EditIcon } from '@vicons/ionicons5'
import { useHubStore } from '@/stores/hub'
import type { DeviceVariable, Device } from '@/types/api'

const message = useMessage()
const hubStore = useHubStore()

const { variables, selectedDevice, loading, devices } = hubStore

// 表单引用
const formRef = ref<FormInst | null>(null)
const editFormRef = ref<FormInst | null>(null)

// 状态
const showAddModal = ref(false)
const showEditModal = ref(false)
const submitting = ref(false)
const editingVariable = ref<DeviceVariable>({} as DeviceVariable)

// 新变量表单数据
const newVariable = ref({
  deviceUID: null as number | null,
  name: '',
  value: '',
  type: 'string'
})

// 表单验证规则
const rules: FormRules = {
  deviceUID: { required: true, type: 'number', message: '请选择设备' },
  name: { required: true, message: '请输入变量名' },
  value: { required: true, message: '请输入变量值' }
}

const editRules: FormRules = {
  Value: { required: true, message: '请输入变量值' }
}

// 设备选项
const deviceOptions = computed(() => {
  const getAllDevices = (nodes: any[]): Device[] => {
    let allDevices: Device[] = []
    for (const node of nodes) {
      allDevices.push(node)
      if (node.children && node.children.length > 0) {
        allDevices = allDevices.concat(getAllDevices(node.children))
      }
    }
    return allDevices
  }

  return getAllDevices(devices).map(device => ({
    label: `${device.Name || device.HardwareID} (${device.Role.toUpperCase()})`,
    value: device.DeviceUID
  }))
})

// 数据类型选项
const typeOptions = [
  { label: '字符串', value: 'string' },
  { label: '数字', value: 'number' },
  { label: '布尔值', value: 'boolean' },
  { label: 'JSON', value: 'json' }
]

// 表格列定义
const columns: DataTableColumns<DeviceVariable> = [
  {
    title: '设备',
    key: 'Device',
    width: 200,
    render(row) {
      return h('div', [
        h(NTag, { 
          size: 'small', 
          type: getDeviceTagType(row.Device.Role) 
        }, () => row.Device.Role.toUpperCase()),
        h('span', { style: 'margin-left: 8px;' }, 
          row.Device.Name || row.Device.HardwareID)
      ])
    }
  },
  {
    title: '变量名',
    key: 'VariableName',
    width: 150,
    render(row) {
      return h(NTag, { type: 'info', size: 'small' }, () => row.VariableName)
    }
  },
  {
    title: '变量值',
    key: 'Value',
    width: 200,
    render(row) {
      const value = formatValue(row.Value)
      const valueType = typeof row.Value
      return h('div', [
        h('div', { style: 'font-family: monospace;' }, value),
        h('small', { style: 'color: #999;' }, `(${valueType})`)
      ])
    }
  },
  {
    title: '更新时间',
    key: 'UpdatedAt',
    width: 160,
    render(row) {
      return h(NTime, { value: new Date(row.UpdatedAt).getTime() })
    }
  },
  {
    title: '操作',
    key: 'actions',
    width: 120,
    render(row) {
      return h(NSpace, () => [
        h(NButton, {
          size: 'small',
          type: 'primary',
          onClick: () => handleEdit(row)
        }, {
          default: () => '编辑',
          icon: () => h(NIcon, () => h(EditIcon))
        })
      ])
    }
  }
]

// 分页配置
const paginationProps = {
  pageSize: 10,
  showSizePicker: true,
  pageSizes: [5, 10, 20, 50],
  showQuickJumper: true,
  prefix: (info: any) => `共 ${info.itemCount} 条`
}

// 工具函数
const getDeviceTagType = (role: string) => {
  switch (role) {
    case 'hub': return 'success'
    case 'relay': return 'warning'
    case 'node': return 'info'
    case 'manager': return 'error'
    default: return 'default'
  }
}

const formatValue = (value: any): string => {
  if (value === null || value === undefined) return 'null'
  if (typeof value === 'object') return JSON.stringify(value, null, 2)
  return String(value)
}

const getInputType = (value: any): "text" | "textarea" | "password" | undefined => {
  return "text" // 简化处理，都使用text类型
}

// 事件处理
const refreshVariables = async () => {
  if (selectedDevice) {
    await hubStore.fetchVariables(selectedDevice.DeviceUID)
  } else {
    await hubStore.fetchVariables()
  }
}

const handleEdit = (variable: DeviceVariable) => {
  editingVariable.value = { ...variable }
  showEditModal.value = true
}

const handleAddVariable = async () => {
  if (!formRef.value) return
  
  try {
    await formRef.value.validate()
    submitting.value = true
    
    let processedValue: any = newVariable.value.value
    
    // 根据类型处理值
    switch (newVariable.value.type) {
      case 'number':
        processedValue = Number(newVariable.value.value)
        break
      case 'boolean':
        processedValue = newVariable.value.value === 'true'
        break
      case 'json':
        try {
          processedValue = JSON.parse(newVariable.value.value)
        } catch {
          message.error('无效的JSON格式')
          return
        }
        break
    }
    
    const success = await hubStore.updateVariable(
      newVariable.value.deviceUID!,
      newVariable.value.name,
      processedValue
    )
    
    if (success) {
      message.success('变量添加成功')
      showAddModal.value = false
      newVariable.value = { deviceUID: null, name: '', value: '', type: 'string' }
    } else {
      message.error(hubStore.error || '添加变量失败')
    }
  } catch (error) {
    console.error('Validation failed:', error)
  } finally {
    submitting.value = false
  }
}

const handleUpdateVariable = async () => {
  if (!editFormRef.value) return
  
  try {
    await editFormRef.value.validate()
    submitting.value = true
    
    const success = await hubStore.updateVariable(
      editingVariable.value.Device.DeviceUID,
      editingVariable.value.VariableName,
      editingVariable.value.Value
    )
    
    if (success) {
      message.success('变量更新成功')
      showEditModal.value = false
    } else {
      message.error(hubStore.error || '更新变量失败')
    }
  } catch (error) {
    console.error('Validation failed:', error)
  } finally {
    submitting.value = false
  }
}

// 初始化
onMounted(() => {
  refreshVariables()
})
</script>

<style scoped>
:deep(.n-data-table-th) {
  font-weight: 600;
}

:deep(.n-data-table-td) {
  padding: 8px 16px;
}
</style>
