<template>
  <div class="profile-wrap">
    <n-card title="个人资料" size="small">
      <n-form :model="profile" :rules="rules" label-placement="left" label-width="80">
        <n-form-item label="用户名">
          <n-input :value="auth.user?.username" disabled />
        </n-form-item>
        <n-form-item label="显示名" path="displayName">
          <n-input v-model:value="profile.displayName" placeholder="请输入显示名" />
        </n-form-item>
        <div class="actions">
          <n-button type="primary" :loading="savingProfile" @click="saveProfile">保存资料</n-button>
        </div>
      </n-form>
    </n-card>

    <n-card title="修改密码" size="small" class="mt">
      <n-form :model="pwd" :rules="pwdRules" label-placement="left" label-width="80">
        <n-form-item label="旧密码" path="oldPassword">
          <n-input v-model:value="pwd.oldPassword" type="password" show-password-on="click" placeholder="请输入旧密码" />
        </n-form-item>
        <n-form-item label="新密码" path="newPassword">
          <n-input v-model:value="pwd.newPassword" type="password" show-password-on="click" placeholder="请输入新密码" />
        </n-form-item>
        <n-form-item label="确认密码" path="confirmPassword">
          <n-input v-model:value="pwd.confirmPassword" type="password" show-password-on="click" placeholder="请再次输入" />
        </n-form-item>
        <div class="actions">
          <n-button type="primary" :loading="savingPwd" @click="changePassword">更新密码</n-button>
        </div>
        
      </n-form>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { apiService } from '@/services/api'
import { useMessage } from 'naive-ui'
import { NCard, NForm, NFormItem, NInput, NButton } from 'naive-ui'

const auth = useAuthStore()
const message = useMessage()

const profile = ref({ displayName: auth.user?.displayName || '' })
const savingProfile = ref(false)

const rules = {
  displayName: { required: false, trigger: 'blur' }
}

async function saveProfile() {
  if (!auth.user) return
  savingProfile.value = true
  try {
  const res = await apiService.updateProfile({ displayName: profile.value.displayName })
    if ((res as any).success !== false) {
      // 更新本地 store 显示名
      auth.user = { ...(auth.user as any), displayName: profile.value.displayName }
      message.success('资料已保存')
    } else {
      message.error((res as any).message || '保存失败')
    }
  } catch (e) {
    message.error('保存失败')
  } finally {
    savingProfile.value = false
  }
}

const pwd = ref({ oldPassword: '', newPassword: '', confirmPassword: '' })
const savingPwd = ref(false)

const pwdRules = {
  oldPassword: { required: true, message: '请输入旧密码', trigger: 'blur' },
  newPassword: { required: true, message: '请输入新密码', trigger: 'blur' },
  confirmPassword: {
    required: true,
    message: '请确认新密码',
    trigger: 'blur'
  }
}

async function changePassword() {
  if (!auth.user) return
  if (!pwd.value.oldPassword || !pwd.value.newPassword || pwd.value.newPassword !== pwd.value.confirmPassword) {
    message.warning('两次输入的密码不一致')
    return
  }
  savingPwd.value = true
  try {
  const res = await apiService.changeMyPassword(pwd.value.oldPassword, pwd.value.newPassword)
    if ((res as any).success !== false) {
      message.success('密码已更新')
  pwd.value = { oldPassword: '', newPassword: '', confirmPassword: '' }
    } else {
      message.error((res as any).message || '更新失败')
    }
  } catch (e) {
    message.error('更新失败')
  } finally {
    savingPwd.value = false
  }
}
</script>

<style scoped>
.profile-wrap { max-width: 720px; margin: 0 auto; padding: 24px; }
.mt { margin-top: 16px; }
.actions { margin-top: 8px; display: flex; gap: 12px; }
.tip { margin-top: 8px; font-size: 12px; color: #888; }
</style>
