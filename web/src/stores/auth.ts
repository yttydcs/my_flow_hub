import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { apiService } from '@/services/api'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(null)
  const user = ref<{ id: number; username: string; displayName?: string } | null>(null)
  const permissions = ref<string[]>([])

  const isLoggedIn = computed(() => !!token.value)
  const isAdmin = computed(() => permissions.value.includes('admin.manage') || permissions.value.includes('**'))

  async function login(username: string, password: string) {
    const res = await apiService.login(username, password)
    if (res.success) {
      const t = (res as any).token as string | undefined
      const u = (res as any).user as any
      const perms = ((res as any).permissions as string[]) || []
      if (t) {
        token.value = t
        ;(apiService as any).setToken(token.value)
      }
      if (u) {
        user.value = { id: u.id, username: u.username, displayName: u.displayName }
      }
      permissions.value = perms
    }
    return res
  }

  function logout() {
    token.value = null
    user.value = null
    permissions.value = []
    ;(apiService as any).setToken(null)
  }

  return { token, user, permissions, isLoggedIn, isAdmin, login, logout }
})
