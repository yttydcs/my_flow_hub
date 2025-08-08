<template>
  <div style="max-width: 800px; margin: 0 auto;">
    <n-h2>系统设置</n-h2>
    
    <!-- 系统设置列表 -->
    <n-card>
      <n-list clickable>
        <!-- 连接设置 -->
        <n-list-item @click="showConnectionModal = true">
          <template #prefix>
            <n-icon size="20" color="#18a058">
              <LinkIcon />
            </n-icon>
          </template>
          <n-thing>
            <template #header>连接设置</template>
            <template #description>配置API和WebSocket连接地址</template>
          </n-thing>
          <template #suffix>
            <n-icon size="16" color="#999">
              <ChevronForwardIcon />
            </n-icon>
          </template>
        </n-list-item>

        <n-divider style="margin: 0;" />

        <!-- 界面设置 -->
        <n-list-item @click="showThemeModal = true">
          <template #prefix>
            <n-icon size="20" color="#722ed1">
              <ColorPaletteIcon />
            </n-icon>
          </template>
          <n-thing>
            <template #header>界面设置</template>
            <template #description>主题、语言和显示选项</template>
          </n-thing>
          <template #suffix>
            <n-icon size="16" color="#999">
              <ChevronForwardIcon />
            </n-icon>
          </template>
        </n-list-item>

        <n-divider style="margin: 0;" />

        <!-- 通知设置 -->
        <n-list-item @click="showNotificationModal = true">
          <template #prefix>
            <n-icon size="20" color="#fa8c16">
              <NotificationsIcon />
            </n-icon>
          </template>
          <n-thing>
            <template #header>通知设置</template>
            <template #description>设备状态和系统通知</template>
          </n-thing>
          <template #suffix>
            <n-icon size="16" color="#999">
              <ChevronForwardIcon />
            </n-icon>
          </template>
        </n-list-item>

        <n-divider style="margin: 0;" />

        <!-- 安全设置 -->
        <n-list-item @click="showSecurityModal = true">
          <template #prefix>
            <n-icon size="20" color="#f5222d">
              <ShieldIcon />
            </n-icon>
          </template>
          <n-thing>
            <template #header>安全设置</template>
            <template #description>账户安全和访问控制</template>
          </n-thing>
          <template #suffix>
            <n-icon size="16" color="#999">
              <ChevronForwardIcon />
            </n-icon>
          </template>
        </n-list-item>

        <n-divider style="margin: 0;" />

        <!-- 系统信息 -->
        <n-list-item @click="showSystemInfoModal = true">
          <template #prefix>
            <n-icon size="20" color="#1890ff">
              <InformationCircleIcon />
            </n-icon>
          </template>
          <n-thing>
            <template #header>系统信息</template>
            <template #description>版本信息和系统状态</template>
          </n-thing>
          <template #suffix>
            <n-icon size="16" color="#999">
              <ChevronForwardIcon />
            </n-icon>
          </template>
        </n-list-item>
      </n-list>
    </n-card>

    <!-- 连接设置弹窗 -->
    <n-modal v-model:show="showConnectionModal" preset="dialog" title="连接设置">
      <n-form :model="connectionSettings" label-placement="left" :label-width="120">
        <n-form-item label="API地址">
          <n-input v-model:value="connectionSettings.apiUrl" placeholder="http://localhost:8090" />
        </n-form-item>
        <n-form-item label="WebSocket地址">
          <n-input v-model:value="connectionSettings.wsUrl" placeholder="ws://localhost:8080/ws" />
        </n-form-item>
        <n-form-item label="连接超时(秒)">
          <n-input-number v-model:value="connectionSettings.timeout" :min="5" :max="60" />
        </n-form-item>
      </n-form>
      <template #action>
        <n-space>
          <n-button @click="showConnectionModal = false">取消</n-button>
          <n-button type="primary" @click="saveConnectionSettings">保存</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 界面设置弹窗 -->
    <n-modal v-model:show="showThemeModal" preset="dialog" title="界面设置">
      <n-form :model="themeSettings" label-placement="left" :label-width="120">
        <n-form-item label="主题">
          <n-select 
            v-model:value="themeSettings.theme" 
            :options="themeOptions"
          />
        </n-form-item>
        <n-form-item label="语言">
          <n-select 
            v-model:value="themeSettings.language" 
            :options="languageOptions"
          />
        </n-form-item>
        <n-form-item label="自动刷新间隔(秒)">
          <n-input-number v-model:value="themeSettings.refreshInterval" :min="10" :max="300" />
        </n-form-item>
      </n-form>
      <template #action>
        <n-space>
          <n-button @click="showThemeModal = false">取消</n-button>
          <n-button type="primary" @click="saveThemeSettings">保存</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 其他设置弹窗占位 -->
    <n-modal v-model:show="showNotificationModal" preset="dialog" title="通知设置">
      <p>通知设置功能开发中...</p>
      <template #action>
        <n-button @click="showNotificationModal = false">关闭</n-button>
      </template>
    </n-modal>

    <n-modal v-model:show="showSecurityModal" preset="dialog" title="安全设置">
      <p>安全设置功能开发中...</p>
      <template #action>
        <n-button @click="showSecurityModal = false">关闭</n-button>
      </template>
    </n-modal>

    <n-modal v-model:show="showSystemInfoModal" preset="dialog" title="系统信息">
      <n-descriptions label-placement="left" :column="1" bordered>
        <n-descriptions-item label="应用版本">1.0.0</n-descriptions-item>
        <n-descriptions-item label="构建时间">{{ new Date().toLocaleDateString() }}</n-descriptions-item>
        <n-descriptions-item label="运行环境">{{ import.meta.env.MODE }}</n-descriptions-item>
      </n-descriptions>
      <template #action>
        <n-button @click="showSystemInfoModal = false">关闭</n-button>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useMessage } from 'naive-ui'
import {
  NH2, NCard, NList, NListItem, NDivider, NThing, NIcon,
  NModal, NForm, NFormItem, NInput, NInputNumber, NSelect,
  NSpace, NButton, NDescriptions, NDescriptionsItem
} from 'naive-ui'
import {
  Link as LinkIcon,
  ColorPalette as ColorPaletteIcon,
  Notifications as NotificationsIcon,
  Shield as ShieldIcon,
  InformationCircle as InformationCircleIcon,
  ChevronForward as ChevronForwardIcon
} from '@vicons/ionicons5'

const message = useMessage()

// 弹窗显示状态
const showConnectionModal = ref(false)
const showThemeModal = ref(false)
const showNotificationModal = ref(false)
const showSecurityModal = ref(false)
const showSystemInfoModal = ref(false)

// 连接设置
const connectionSettings = ref({
  apiUrl: 'http://localhost:8090/api',
  wsUrl: 'ws://localhost:8080/ws',
  timeout: 30
})

// 界面设置
const themeSettings = ref({
  theme: 'light',
  language: 'zh-CN',
  refreshInterval: 30
})

// 主题选项
const themeOptions = [
  { label: '浅色主题', value: 'light' },
  { label: '深色主题', value: 'dark' },
  { label: '跟随系统', value: 'auto' }
]

// 语言选项
const languageOptions = [
  { label: '简体中文', value: 'zh-CN' },
  { label: 'English', value: 'en-US' }
]

// 保存连接设置
const saveConnectionSettings = () => {
  // TODO: 保存到本地存储或发送到服务器
  localStorage.setItem('connectionSettings', JSON.stringify(connectionSettings.value))
  message.success('连接设置已保存')
  showConnectionModal.value = false
}

// 保存界面设置
const saveThemeSettings = () => {
  // TODO: 应用主题设置
  localStorage.setItem('themeSettings', JSON.stringify(themeSettings.value))
  message.success('界面设置已保存')
  showThemeModal.value = false
}
</script>

<style scoped>
.n-list-item {
  padding: 16px !important;
  transition: background-color 0.3s ease;
}

.n-list-item:hover {
  background-color: #f5f5f5;
}

:deep(.n-thing-header) {
  font-weight: 500;
  font-size: 16px;
}

:deep(.n-thing-description) {
  color: #666;
  font-size: 14px;
}
</style>
