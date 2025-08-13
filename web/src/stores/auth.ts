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
        try { localStorage.setItem('mf_token', token.value) } catch {}
      }
      if (u) {
        user.value = { id: u.id, username: u.username, displayName: u.displayName }
      }
      permissions.value = perms
    }
    return res
  }

  function logout() {
    // 主动让后端撤销当前 token
    const t = token.value
    if (t) { apiService.logout().catch(() => {}) }
    token.value = null
    user.value = null
    permissions.value = []
    ;(apiService as any).setToken(null)
    try { localStorage.removeItem('mf_token') } catch {}
  }

  // 启动时尝试恢复 token 并拉取用户信息
  ;(() => {
    try {
      const t = localStorage.getItem('mf_token')
      if (t) {
        token.value = t
        ;(apiService as any).setToken(t)
        // 同步获取用户信息
        apiService.me().then((res: any) => {
          if (res && res.user) {
            user.value = { id: res.user.id, username: res.user.username, displayName: res.user.displayName }
            permissions.value = res.permissions || []
          } else {
            // 失效则清理
            logout()
          }
        }).catch(() => logout())
      }
    } catch {}
  })()

  return { token, user, permissions, isLoggedIn, isAdmin, login, logout }
})
