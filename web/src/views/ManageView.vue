<template>
  <n-layout has-sider style="height: calc(100vh - 60px)">
    <!-- 左侧菜单 -->
    <n-layout-sider
      bordered
      collapse-mode="width"
      :collapsed-width="64"
      :width="240"
      show-trigger
      content-style="padding: 16px 0;"
    >
      <n-menu
        :value="currentMenuKey"
        :options="menuOptions"
        @update:value="handleMenuSelect"
      />
    </n-layout-sider>

    <!-- 主内容区域 -->
    <n-layout-content style="padding: 24px;">
      <router-view />
    </n-layout-content>
  </n-layout>
</template>

<script setup lang="ts">
import { computed, h } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useRouter, useRoute } from 'vue-router'
import { 
  NLayout, NLayoutSider, NLayoutContent, NMenu
} from 'naive-ui'
import type { MenuOption } from 'naive-ui'
import {
  Cube as DeviceIcon,
  Analytics as VariableIcon,
  People as UsersIcon,
  Document as LogsIcon,
  Key as KeyIcon
} from '@vicons/ionicons5'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

// 当前选中的菜单项
const currentMenuKey = computed(() => {
  return route.name as string
})

// 菜单选项
const menuOptions = computed<MenuOption[]>(() => {
  const base: MenuOption[] = [
    { label: '设备管理', key: 'manage-devices', icon: () => h(DeviceIcon) },
  { label: '变量管理', key: 'manage-variables', icon: () => h(VariableIcon) },
  { label: '密钥管理', key: 'manage-keys', icon: () => h(KeyIcon) }
  ]
  if (auth.isAdmin) base.push({ label: '用户管理', key: 'manage-users', icon: () => h(UsersIcon) })
  const canViewLogs = auth.isAdmin || auth.permissions.includes('log.read')
  if (canViewLogs) base.push({ label: '日志管理', key: 'manage-logs', icon: () => h(LogsIcon) })
  return base
})

// 处理菜单选择
const handleMenuSelect = (key: string) => {
  switch (key) {
    case 'manage-devices':
      router.push('/manage/devices')
      break
    case 'manage-variables':
      router.push('/manage/variables')
      break
    case 'manage-keys':
      router.push('/manage/keys')
      break
    case 'manage-users':
      router.push('/manage/users')
      break
    case 'manage-logs':
      router.push('/manage/logs')
      break
  }
}
</script>

<style scoped>
.n-layout-sider {
  /* 使用 Naive UI 主题变量，随明暗主题自动切换 */
  background-color: var(--n-color);
}
</style>
