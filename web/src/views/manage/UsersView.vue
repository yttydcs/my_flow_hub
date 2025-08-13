<template>
  <div>
    <n-space align="center" justify="space-between" style="margin-bottom: 12px; width: 100%;">
      <n-h2 style="margin: 0;">用户管理</n-h2>
      <n-space>
        <n-button type="primary" @click="openCreate">新建用户</n-button>
        <n-button @click="loadUsers" :loading="loading">刷新</n-button>
      </n-space>
    </n-space>

  <n-data-table :columns="columns" :data="users" :loading="loading" :bordered="false" />

    <!-- 创建/编辑用户弹窗 -->
    <n-modal v-model:show="showEdit" preset="dialog" :title="editing ? '编辑用户' : '新建用户'">
      <n-form :model="form" label-placement="left" label-width="96">
        <n-form-item label="用户名" v-if="!editing">
          <n-input v-model:value="form.username" placeholder="用户名" />
        </n-form-item>
        <n-form-item label="显示名">
          <n-input v-model:value="form.displayName" placeholder="显示名" />
        </n-form-item>
        <n-form-item label="密码">
          <n-input v-model:value="form.password" type="password" placeholder="初始密码（可留空）" />
        </n-form-item>
        <n-form-item label="禁用">
          <n-switch v-model:value="form.disabled" />
        </n-form-item>
      </n-form>
      <template #action>
        <n-space>
          <n-button @click="showEdit = false">取消</n-button>
          <n-button type="primary" :loading="saving" @click="saveUser">保存</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 重置密码弹窗 -->
    <n-modal v-model:show="showReset" preset="dialog" title="重置密码">
      <n-form :model="resetForm" label-placement="left" label-width="96">
        <n-form-item label="新密码">
          <n-input v-model:value="resetForm.password" type="password" placeholder="输入新密码" />
        </n-form-item>
      </n-form>
      <template #action>
        <n-space>
          <n-button @click="showReset = false">取消</n-button>
          <n-button type="primary" :loading="saving" @click="confirmReset">确定</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 编辑权限弹窗 -->
    <n-modal v-model:show="showPerms" preset="dialog" title="编辑用户权限">
      <n-space vertical>
        <div>
          <div style="margin-bottom: 8px; font-weight: 500;">当前权限节点</div>
          <n-space wrap>
            <n-tag v-for="node in permNodes" :key="node" closable @close="() => removePerm(node)">{{ node }}</n-tag>
          </n-space>
        </div>
        <n-form :model="{ newNode }" label-placement="left" label-width="96">
          <n-form-item label="新增节点">
            <n-input v-model:value="newNode" placeholder="例如：admin.manage 或 devices.**.read" />
          </n-form-item>
          <n-button type="primary" @click="addPerm">添加</n-button>
        </n-form>
      </n-space>
      <template #action>
        <n-space>
          <n-button @click="showPerms = false">关闭</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
  
</template>

<script setup lang="ts">
import { h, onMounted, reactive, ref } from 'vue'
import { useMessage, NButton, NButtonGroup, NIcon, NTag } from 'naive-ui'
import {
  NH2, NSpace, NDataTable, NModal, NForm, NFormItem, NInput, NSwitch
} from 'naive-ui'
import { apiService } from '@/services/api'
import type { User } from '@/types/api'
import { Pencil as PencilIcon, Trash as TrashIcon, Key as KeyIcon } from '@vicons/ionicons5'

const message = useMessage()
const loading = ref(false)
const saving = ref(false)
const users = ref<User[]>([])

const showEdit = ref(false)
const editing = ref(false)
const currentId = ref<number | null>(null)
const form = reactive<{ username: string; displayName?: string; password?: string; disabled?: boolean }>({ username: '', displayName: '', password: '', disabled: false })

const showReset = ref(false)
const resetForm = reactive<{ password: string }>({ password: '' })

// 权限编辑状态
const showPerms = ref(false)
const permUserId = ref<number | null>(null)
const permNodes = ref<string[]>([])
const newNode = ref('')

const columns = [
  { title: 'ID', key: 'ID', width: 80 },
  { title: '用户名', key: 'Username' },
  { title: '显示名', key: 'DisplayName' },
  { title: '状态', key: 'Disabled', render(row: User) { return row.Disabled ? '已禁用' : '启用' } },
  {
    title: '操作', key: 'actions', width: 320, render(row: User) {
      return h(NButtonGroup, null, {
        default: () => [
          h(NButton, { size: 'small', onClick: () => openEdit(row) }, { default: () => '编辑', icon: () => h(NIcon, null, { default: () => h(PencilIcon) }) }),
          h(NButton, { size: 'small', onClick: () => openPerms(row) }, { default: () => '编辑权限' }),
          h(NButton, { size: 'small', type: 'warning', onClick: () => openReset(row) }, { default: () => '重置密码', icon: () => h(NIcon, null, { default: () => h(KeyIcon) }) }),
          h(NButton, { size: 'small', type: 'error', onClick: () => delUser(row) }, { default: () => '删除', icon: () => h(NIcon, null, { default: () => h(TrashIcon) }) }),
        ]
      })
    }
  }
]

async function loadUsers() {
  loading.value = true
  try {
    const res = await apiService.getUsers()
    if (res.success) users.value = (res.data as User[]) || []
    else message.error(res.message || '加载失败')
  } catch (e) {
    message.error('网络错误')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = false
  currentId.value = null
  form.username = ''
  form.displayName = ''
  form.password = ''
  form.disabled = false
  showEdit.value = true
}

function openEdit(row: User) {
  editing.value = true
  currentId.value = row.ID
  form.username = row.Username
  form.displayName = row.DisplayName || ''
  form.password = ''
  form.disabled = !!row.Disabled
  showEdit.value = true
}

async function saveUser() {
  saving.value = true
  try {
    if (editing.value && currentId.value) {
      const payload: any = { id: currentId.value, displayName: form.displayName, disabled: form.disabled }
      if (form.password) payload.password = form.password
      const res = await apiService.updateUser(payload)
      if (res.success) { message.success('已保存'); showEdit.value = false; loadUsers() } else message.error(res.message || '保存失败')
    } else {
      const res = await apiService.createUser({ username: form.username, displayName: form.displayName, password: form.password || '' })
      if (res.success) { message.success('已创建'); showEdit.value = false; loadUsers() } else message.error(res.message || '创建失败')
    }
  } catch (e) {
    message.error('网络错误')
  } finally {
    saving.value = false
  }
}

function openReset(row: User) {
  currentId.value = row.ID
  resetForm.password = ''
  showReset.value = true
}

async function confirmReset() {
  if (!currentId.value) return
  saving.value = true
  try {
    const res = await apiService.updateUser({ id: currentId.value, password: resetForm.password })
    if (res.success) { message.success('密码已重置'); showReset.value = false } else message.error(res.message || '重置失败')
  } catch (e) {
    message.error('网络错误')
  } finally {
    saving.value = false
    loadUsers()
  }
}

async function delUser(row: User) {
  if (!confirm(`确认删除用户 ${row.Username} ?`)) return
  try {
    const res = await apiService.deleteUser(row.ID)
    if (res.success) { message.success('已删除'); loadUsers() } else message.error(res.message || '删除失败')
  } catch (e) {
    message.error('网络错误')
  }
}

function openPerms(row: User) {
  permUserId.value = row.ID
  showPerms.value = true
  loadPerms()
}

async function loadPerms() {
  if (!permUserId.value) return
  try {
    const res = await apiService.listUserPerms(permUserId.value)
    if (res.success) permNodes.value = (res.data as string[]) || []
    else permNodes.value = []
  } catch {
    permNodes.value = []
  }
}

async function addPerm() {
  if (!permUserId.value || !newNode.value.trim()) return
  const node = newNode.value.trim()
  const res = await apiService.addUserPerm(permUserId.value, node)
  if (res.success) { newNode.value = ''; loadPerms() }
}

async function removePerm(node: string) {
  if (!permUserId.value) return
  const res = await apiService.removeUserPerm(permUserId.value, node)
  if (res.success) loadPerms()
}

onMounted(loadUsers)
</script>

<style scoped>
</style>
