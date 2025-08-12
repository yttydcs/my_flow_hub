import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import HomeView from '@/views/HomeView.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue')
    },
    {
      path: '/',
      name: 'home',
      component: HomeView,
      children: [
        {
          path: '',
          name: 'dashboard',
          component: () => import('@/views/DashboardView.vue')
        },
        {
          path: 'manage',
          name: 'manage',
          component: () => import('@/views/ManageView.vue'),
          children: [
            {
              path: 'devices',
              name: 'manage-devices',
              component: () => import('@/views/manage/DevicesView.vue')
            },
            {
              path: 'variables',
              name: 'manage-variables', 
              component: () => import('@/views/manage/VariablesView.vue')
            },
            {
              path: 'users',
              name: 'manage-users',
              component: () => import('@/views/manage/UsersView.vue')
            },
            {
              path: 'logs',
              name: 'manage-logs',
              component: () => import('@/views/manage/LogsView.vue')
            }
          ]
        },
        {
          path: 'settings',
          name: 'settings',
          component: () => import('@/views/SettingsView.vue')
        }
      ]
    }
  ]
})

// 全局路由守卫：未登录只允许访问 /login
router.beforeEach((to) => {
  const auth = useAuthStore()
  if (!auth.isLoggedIn && to.name !== 'login') {
    return { name: 'login', replace: true }
  }
  if (auth.isLoggedIn && to.name === 'login') {
    return { name: 'dashboard', replace: true }
  }
})

export default router
