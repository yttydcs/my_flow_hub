<template>
  <div class="login-wrapper">
    <n-card style="width: 360px;">
      <template #header>登录</template>
      <n-form :model="form" label-placement="left" label-width="72">
        <n-form-item label="用户名">
          <n-input v-model:value="form.username" placeholder="admin" @keyup.enter="onLogin" />
        </n-form-item>
        <n-form-item label="密码">
          <n-input v-model:value="form.password" type="password" placeholder="******" @keyup.enter="onLogin" />
        </n-form-item>
        <n-button type="primary" block @click="onLogin" :loading="loading">登录</n-button>
      </n-form>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useMessage, NCard, NForm, NFormItem, NInput, NButton } from 'naive-ui'
import { useAuthStore } from '@/stores/auth'
import { useRouter } from 'vue-router'

const auth = useAuthStore()
const router = useRouter()
const message = useMessage()

const form = reactive({ username: 'admin', password: '' })
const loading = ref(false)

async function onLogin() {
  if (loading.value) return
  loading.value = true
  try {
    const res = await auth.login(form.username, form.password)
    if ((res as any).success) {
      message.success('登录成功')
      router.replace('/')
    } else {
      message.error((res as any).message || '登录失败')
    }
  } catch (e) {
    message.error('网络错误')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-wrapper {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f5f6f7;
  padding: 24px;
}
</style>
