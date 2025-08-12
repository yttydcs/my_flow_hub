import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { apiService } from '@/services/api'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(null)
  const user = ref<{ id: number; username: string; displayName?: string } | null>(null)

  const isLoggedIn = computed(() => !!token.value)
  const isAdmin = computed(() => !!token.value) // TODO: 后端返回权限快照后再判断，如包含 admin.manage

  async function login(username: string, password: string) {
    const res = await apiService.login(username, password)
    if (res.success && (res as any).token) {
      token.value = (res as any).token
      ;(apiService as any).setToken(token.value)
    }
    return res
  }

  function logout() {
    token.value = null
    ;(apiService as any).setToken(null)
  }

  return { token, user, isLoggedIn, isAdmin, login, logout }
})
