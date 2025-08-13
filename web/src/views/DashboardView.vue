<template>
  <div class="hero">
    <div class="hero-inner">
      <div class="hero-left">
        <div class="title">MyFlowHub 中枢</div>
        <div class="subtitle">统一管理设备、变量与访问权限</div>
        <div class="ctas">
          <n-button type="primary" size="large" @click="$router.push('/manage/devices')">开始管理设备</n-button>
          <n-button size="large" tertiary @click="$router.push('/manage/keys')">发放用户密钥</n-button>
        </div>
        <div class="stats">
          <n-statistic label="设备数量" :value="deviceCount" />
          <n-statistic label="变量数量" :value="variables.length" />
        </div>
      </div>
      <div class="hero-right">
        <div class="blob">
          <div class="blob-inner">
            <div class="blob-title">所有设备 ({{ allDevices.length }})</div>
            <div class="device-list">
              <div v-for="d in allDevices" :key="d.DeviceUID" class="device-item">
                <div class="name">{{ d.Name || d.HardwareID || d.DeviceUID }}</div>
                <div class="meta">{{ d.Role }} • 最后在线：{{ formatLastSeen(d.LastSeen) }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { NButton, NStatistic } from 'naive-ui'
import { useHubStore } from '@/stores/hub'
import type { Device, DeviceTreeNode } from '@/types/api'

const hub = useHubStore()
const deviceCount = computed(() => hub.deviceCount)
const variables = computed(() => hub.variables)

onMounted(() => {
  // 首屏只拉一次，避免旧版轮询带来的页面抖动
  hub.fetchDeviceTree()
  hub.fetchVariables()
})

// 将设备树拍平成数组
function flattenDevices(nodes: DeviceTreeNode[]): Device[] {
  const list: Device[] = []
  const walk = (arr: DeviceTreeNode[]) => {
    for (const n of arr) {
      list.push(n)
      if (n.children && n.children.length) walk(n.children)
    }
  }
  walk(nodes)
  return list
}

const allDevices = computed(() => flattenDevices(hub.devices))

function formatLastSeen(ls?: string | null): string {
  if (!ls) return '未知'
  const t = new Date(ls)
  return t.toLocaleString()
}
</script>

<style scoped>
.hero {
  height: calc(100vh - 60px);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: linear-gradient(135deg, rgba(24,160,88,.06), rgba(24,160,88,.02));
}
.hero-inner {
  width: 100%;
  max-width: 1080px;
  display: grid;
  grid-template-columns: 1.2fr 1fr;
  gap: 32px;
  align-items: center;
}
.hero-left .title {
  font-size: 40px;
  font-weight: 800;
  letter-spacing: .2px;
}
.hero-left .subtitle {
  margin-top: 8px;
  color: #666;
}
.ctas { margin-top: 20px; display: flex; gap: 12px; }
.stats { margin-top: 24px; display: flex; gap: 24px; }
.hero-right { position: relative; }
.blob {
  width: 100%;
  height: 320px;
  border-radius: 8px;
  overflow: hidden; /* 不显示滚动条 */
  background: radial-gradient(120px 80px at 60% 40%, rgba(24,160,88,.25), rgba(24,160,88,0)),
              radial-gradient(140px 100px at 20% 70%, rgba(24,160,88,.2), rgba(24,160,88,0)),
              linear-gradient(135deg, rgba(24,160,88,.08), rgba(24,160,88,.02));
}
.blob-inner { padding: 16px; height: 100%; box-sizing: border-box; display: flex; flex-direction: column; user-select: none; }
.blob-title { font-weight: 700; margin-bottom: 8px; }
.device-list { /* 可滚动但隐藏滚动条 */ flex: 1; overflow: auto; display: flex; flex-direction: column; gap: 8px; scrollbar-width: none; -ms-overflow-style: none; user-select: none; }
.device-list::-webkit-scrollbar { width: 0; height: 0; }
.device-item { padding: 6px 6px; border-bottom: 1px solid rgba(0,0,0,.06); border-radius: 10px; transition: background .2s ease, box-shadow .2s ease, transform .2s ease; cursor: default; user-select: none; }
.device-item:hover { background: rgba(24,160,88,.06); box-shadow: 0 2px 8px -4px rgba(0,0,0,.12); transform: translateY(-1px); border-bottom-color: transparent; }
.device-item:last-child { border-bottom: none; }
.device-item .name { font-weight: 600; }
.device-item .meta { margin-top: 2px; font-size: 12px; color: #666; }
@media (max-width: 900px) {
  .hero-inner { grid-template-columns: 1fr; }
  .hero-right { display: none; }
}
</style>
