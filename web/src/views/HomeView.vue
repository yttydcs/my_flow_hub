<template>
  <n-layout style="height: 100vh">
    <!-- 顶部导航栏 -->
    <n-layout-header 
      style="height: 60px; padding: 0 24px; display: flex; align-items: center; justify-content: space-between;" 
      bordered
    >
      <!-- 左侧Logo和标题 -->
      <div style="display: flex; align-items: center; gap: 12px;">
        <n-icon size="28" color="#18a058"><ServerIcon /></n-icon>
        <div>
          <div style="font-size: 1.4rem; font-weight: bold; color: #18a058;">MyFlowHub</div>
        </div>
      </div>
      
      <!-- 右侧导航和用户信息 -->
    <n-space size="large" v-if="isLoggedIn">
  <n-button-group>
          <n-button 
            :type="$route.name === 'dashboard' ? 'primary' : 'default'"
            @click="$router.push('/')"
          >
            <template #icon>
              <n-icon><HomeIcon /></n-icon>
            </template>
            主页
          </n-button>
          <n-button v-if="isAdmin"
            :type="$route.name?.toString().startsWith('manage') ? 'primary' : 'default'"
            @click="$router.push('/manage/devices')"
          >
            <template #icon>
              <n-icon><SettingsIcon /></n-icon>
            </template>
            管理
          </n-button>
          <n-button 
            :type="$route.name === 'settings' ? 'primary' : 'default'"
            @click="$router.push('/settings')"
          >
            <template #icon>
              <n-icon><CogIcon /></n-icon>
            </template>
            设置
          </n-button>
        </n-button-group>
        
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
  NIcon, NDropdown
} from 'naive-ui'
import { 
  Server as ServerIcon,
  Home as HomeIcon,
  Settings as SettingsIcon,
  Cog as CogIcon,
  Person as PersonIcon,
  LogOut as LogOutIcon,
  Person as ProfileIcon
} from '@vicons/ionicons5'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const auth = useAuthStore()
const isLoggedIn = computed(() => auth.isLoggedIn)
const isAdmin = computed(() => auth.isAdmin)

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
      // TODO: 打开个人资料页面
      console.log('打开个人资料')
      break
    case 'logout':
      // TODO: 退出登录逻辑
      console.log('退出登录')
      break
  }
}

function goLogin() {}
</script>

<style scoped>
.n-layout-header {
  background-color: white;
  border-bottom: 1px solid #e0e0e0;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05);
}

:deep(.n-button-group .n-button--primary) {
  background: linear-gradient(135deg, #18a058, #52c41a);
}
</style>
