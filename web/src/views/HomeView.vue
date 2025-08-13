<template>
  <n-layout style="height: 100vh">
    <!-- 顶部导航栏 -->
    <n-layout-header class="topbar" bordered>
      <!-- 左侧Logo和标题 -->
      <div class="brand">
        <n-icon size="26" color="#18a058"><ServerIcon /></n-icon>
        <div class="brand-title">MyFlowHub</div>
      </div>
      
      <!-- 右侧导航和用户信息 -->
      <n-space size="large" v-if="isLoggedIn" align="center">
        <!-- 图标按钮：主页 -->
        <n-button circle quaternary :type="$route.name === 'dashboard' ? 'primary' : 'default'" @click="$router.push('/')">
          <template #icon>
            <n-icon><HomeIcon /></n-icon>
          </template>
        </n-button>
        <!-- 图标按钮：管理（仅管理员显示） -->
        <n-button v-if="isAdmin" circle quaternary :type="$route.name?.toString().startsWith('manage') ? 'primary' : 'default'" @click="$router.push('/manage/devices')">
          <template #icon>
            <n-icon><SettingsIcon /></n-icon>
          </template>
        </n-button>
        <!-- 图标按钮：设置 -->
        <n-button circle quaternary :type="$route.name === 'settings' ? 'primary' : 'default'" @click="$router.push('/settings')">
          <template #icon>
            <n-icon><CogIcon /></n-icon>
          </template>
        </n-button>
        <!-- 图标按钮：暗黑模式切换（默认用图标，亦可改用图片） -->
        <n-button circle quaternary @click="toggleDark" :title="dark ? '切换到亮色' : '切换到暗色'">
          <template #icon>
            <img v-if="useImageIcons" :src="dark ? darkIconSrc : lightIconSrc" alt="theme" class="theme-icon" />
            <n-icon v-else>
              <component :is="dark ? MoonIcon : SunnyIcon" />
            </n-icon>
          </template>
        </n-button>
        <!-- 当前用户占位 -->
        <n-dropdown trigger="hover" :options="userMenuOptions" @select="handleUserMenuSelect">
          <n-button circle quaternary>
            <template #icon>
              <n-icon size="20"><PersonIcon /></n-icon>
            </template>
          </n-button>
        </n-dropdown>
      </n-space>
    </n-layout-header>

    <!-- 主内容区域 -->
    <n-layout>
      <router-view />
    </n-layout>
  </n-layout>
</template>

<script setup lang="ts">
import { computed, h } from 'vue'
import { useRouter } from 'vue-router'
import { 
  NLayout, NLayoutHeader, NSpace, NButton, NButtonGroup, 
  NIcon, NDropdown, NSwitch
} from 'naive-ui'
import { 
  Server as ServerIcon,
  Home as HomeIcon,
  Settings as SettingsIcon,
  Cog as CogIcon,
  Person as PersonIcon,
  LogOut as LogOutIcon,
  Person as ProfileIcon,
  Moon as MoonIcon,
  Sunny as SunnyIcon
} from '@vicons/ionicons5'
import { useAuthStore } from '@/stores/auth'
import { useUIStore } from '@/stores/ui'

const router = useRouter()
const auth = useAuthStore()
const isLoggedIn = computed(() => auth.isLoggedIn)
const isAdmin = computed(() => auth.isAdmin)
const ui = useUIStore()
const dark = computed({ get: () => ui.darkMode, set: (v: boolean) => (ui.darkMode = v) })

// 图标图片切换（如果你想用自定义图片，设 useImageIcons = true 并提供图片地址）
const useImageIcons = false
const lightIconSrc = '/favicon.ico'
const darkIconSrc = '/favicon.ico'

function toggleDark() {
  ui.darkMode = !ui.darkMode
}

// 用户菜单选项
const userMenuOptions = [
  {
    label: '个人资料',
    key: 'profile',
    icon: () => h(NIcon, null, { default: () => h(ProfileIcon) })
  },
  {
    label: '退出登录',
    key: 'logout', 
    icon: () => h(NIcon, null, { default: () => h(LogOutIcon) })
  }
]

// 处理用户菜单选择
const handleUserMenuSelect = (key: string) => {
  switch (key) {
    case 'profile':
      router.push('/profile')
      break
    case 'logout':
      auth.logout()
      router.replace('/login')
      break
  }
}

function goLogin() {}
</script>

<style scoped>
.topbar {
  height: 60px;
  padding: 0 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  background-color: var(--n-color);
  border-bottom: 1px solid #e0e0e0;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05);
}

.brand { display: flex; align-items: center; gap: 10px; }
.brand-title { font-size: 1.25rem; font-weight: 700; color: #18a058; letter-spacing: .2px; }

:deep(.n-button-group .n-button--primary) {
  background: linear-gradient(135deg, #18a058, #52c41a);
}

/* 图标按钮激活态高亮（primary） */
:deep(.n-button.n-button--primary.n-button--circle) {
  background: linear-gradient(135deg, #18a058, #52c41a);
  color: #fff;
}
.theme-icon { width: 18px; height: 18px; display: block; }
</style>
