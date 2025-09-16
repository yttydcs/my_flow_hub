<template>
  <div>
    <n-space justify="space-between" align="center" style="margin-bottom: 12px">
      <h2 style="margin:0">设备审批</h2>
      <n-button type="primary" @click="refresh">刷新</n-button>
    </n-space>
  <n-data-table :columns="columns" :data="pendingDevices" :loading="loading" />
  <n-empty v-if="!loading && pendingDevices.length===0" description="暂无待审批设备" style="margin-top:16px" />
  </div>
  <n-modal v-model:show="confirmShow" preset="dialog" title="审批确认" positive-text="通过" negative-text="取消" @positive-click="doApprove">
    确认通过设备 {{ currentRow?.HardwareID }} 的接入？
  </n-modal>
  <n-message-provider>
    <n-message-api />
  </n-message-provider>
</template>

<script setup lang="ts">
import { onMounted, ref, computed, h } from 'vue'
import { useHubStore } from '@/stores/hub'
import { apiService } from '@/services/api'
import type { Device } from '@/types/api'
import { NButton, NDataTable, NSpace, NModal, NEmpty, useMessage } from 'naive-ui'

const hub = useHubStore()
const loading = ref(false)
const msg = useMessage()

const flatDevices = computed<Device[]>(() => {
  // 将树拍平
  const out: Device[] = []
  const dfs = (nodes: any[]) => {
    for (const n of nodes) {
      out.push(n)
      if (n.children?.length) dfs(n.children)
    }
  }
  dfs(hub.devices as any[])
  return out
})

const pendingDevices = computed<Device[]>(() =>
  flatDevices.value.filter(d => d.Role !== 'manager' && (d.Approved === false))
)

const columns = [
  { title: 'ID', key: 'ID', width: 80 },
  { title: 'UID', key: 'DeviceUID', width: 120 },
  { title: '硬件ID', key: 'HardwareID' },
  { title: '名称', key: 'Name' },
  { title: '角色', key: 'Role', width: 100 },
  {
    title: '操作', key: 'actions', width: 140,
    render: (row: Device) => h(NButton, { type: 'success', size: 'small', onClick: () => openConfirm(row) }, { default: () => '通过' })
  }
]

const confirmShow = ref(false)
const currentRow = ref<Device | null>(null)
function openConfirm(row: Device) {
  currentRow.value = row
  confirmShow.value = true
}
async function doApprove() {
  if (!currentRow.value) return
  try {
    loading.value = true
    const body: any = { ID: currentRow.value.ID, Role: currentRow.value.Role, Name: currentRow.value.Name, HardwareID: currentRow.value.HardwareID, Approved: true }
    const resp = await apiService.updateDevice(body)
    if (resp.success) {
      msg.success('已通过')
      await refresh()
    } else {
      msg.error(resp.message || '审批失败')
    }
  } finally {
    confirmShow.value = false
    loading.value = false
  }
}

async function refresh() {
  loading.value = true
  try {
    await hub.fetchDeviceTree()
  } finally {
    loading.value = false
  }
}

onMounted(refresh)
</script>

<style scoped>
</style>
