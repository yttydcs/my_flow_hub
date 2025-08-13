<template>
  <div>
    <n-space align="center" justify="space-between" style="margin-bottom: 12px; width: 100%;">
      <n-h2 style="margin: 0;">密钥管理</n-h2>
      <n-space>
        <n-button type="primary" @click="openCreate">新建密钥</n-button>
        <n-button @click="loadKeys" :loading="loading">刷新</n-button>
      </n-space>
    </n-space>

    <n-data-table :columns="columns" :data="keys" :loading="loading" :bordered="false" />

    <n-modal v-model:show="showEdit" preset="dialog" title="新建密钥">
      <n-form :model="form" label-placement="left" label-width="120">
        <n-form-item label="绑定类型">
          <n-select v-model:value="form.bindType" :options="bindTypeOptions" placeholder="可选：不绑定/用户/设备" />
        </n-form-item>
        <n-form-item label="绑定ID" v-if="form.bindType">
          <n-input-number v-model:value="form.bindId" placeholder="对应的用户或设备ID" />
        </n-form-item>
        <n-form-item label="密钥内容">
          <n-input v-model:value="form.secret" placeholder="密钥明文（演示用途）" />
        </n-form-item>
        <n-form-item label="到期时间">
          <n-input v-model:value="form.expiresAt" placeholder="ISO 时间，如 2025-12-31T23:59:59Z（可空）" />
        </n-form-item>
        <n-form-item label="最大使用次数">
          <n-input-number v-model:value="form.maxUses" placeholder="可空" />
        </n-form-item>
      </n-form>
      <template #action>
        <n-space>
          <n-button @click="showEdit = false">取消</n-button>
          <n-button type="primary" :loading="saving" @click="saveKey">保存</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { h, onMounted, ref, reactive, computed } from 'vue'
import { useMessage, NButton, NButtonGroup, NIcon } from 'naive-ui'
import { NH2, NSpace, NDataTable, NModal, NForm, NFormItem, NInput, NInputNumber, NSelect } from 'naive-ui'
import { apiService } from '@/services/api'
import type { Key } from '@/types/api'
import { Trash as TrashIcon } from '@vicons/ionicons5'

const message = useMessage()
const loading = ref(false)
const saving = ref(false)
const keys = ref<Key[]>([])

const showEdit = ref(false)
const form = reactive<{ bindType?: 'user'|'device'; bindId?: number; secret: string; expiresAt?: string; maxUses?: number | null }>({ secret: '' })

const bindTypeOptions = computed(() => ([
  { label: '不绑定', value: undefined },
  { label: '用户', value: 'user' },
  { label: '设备', value: 'device' },
]))

const columns = [
  { title: 'ID', key: 'ID', width: 80 },
  { title: 'OwnerUserID', key: 'OwnerUserID' },
  { title: '绑定', key: 'Bind', render(row: Key) { return `${row.BindSubjectType || '-'}:${row.BindSubjectID ?? '-'}` } },
  { title: '到期', key: 'ExpiresAt', render(row: Key) { return row.ExpiresAt || '-' } },
  { title: '次数', key: 'RemainingUses', render(row: Key) { return row.RemainingUses ?? '-' } },
  { title: '状态', key: 'Revoked', render(row: Key) { return row.Revoked ? '已撤销' : '有效' } },
  { title: '操作', key: 'actions', width: 160, render(row: Key) {
      return h(NButtonGroup, null, {
        default: () => [
          h(NButton, { size: 'small', type: 'error', onClick: () => delKey(row) }, { default: () => '删除', icon: () => h(NIcon, null, { default: () => h(TrashIcon) }) }),
        ]
      })
    }
  }
]

async function loadKeys() {
  loading.value = true
  try {
    const res = await apiService.getKeys()
    if (res.success) keys.value = (res.data as Key[]) || []
    else message.error(res.message || '加载失败')
  } catch (e) {
    message.error('网络错误')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  form.bindType = undefined
  form.bindId = undefined
  form.secret = ''
  form.expiresAt = undefined
  form.maxUses = null
  showEdit.value = true
}

async function saveKey() {
  saving.value = true
  try {
    const res = await apiService.createKey({ bindType: form.bindType, bindId: form.bindId, secret: form.secret, expiresAt: form.expiresAt, maxUses: form.maxUses ?? undefined })
    if (res.success) { message.success('已创建'); showEdit.value = false; loadKeys() } else message.error(res.message || '创建失败')
  } catch (e) {
    message.error('网络错误')
  } finally {
    saving.value = false
  }
}

async function delKey(row: Key) {
  try {
    const res = await apiService.deleteKey(row.ID)
    if (res.success) { message.success('已删除'); loadKeys() } else message.error(res.message || '删除失败')
  } catch (e) {
    message.error('网络错误')
  }
}

onMounted(loadKeys)
</script>

<style scoped>
</style>
