<template>
  <n-layout style="height: 100vh">
    <n-layout-header style="height: 64px; padding: 0 24px; display: flex; align-items: center; justify-content: space-between;" bordered>
      <div style="font-size: 1.5rem; font-weight: bold;">MyFlowHub Dashboard</div>
      <n-space>
        <n-input v-model:value="serverUrl" placeholder="ws://localhost:8080/ws" style="width: 200px;" />
        <n-button @click="handleConnect" :type="isConnected ? 'error' : 'primary'">
          {{ isConnected ? 'Disconnect' : 'Connect' }}
        </n-button>
      </n-space>
    </n-layout-header>
    <n-layout has-sider>
      <n-layout-sider
        bordered
        collapse-mode="width"
        :collapsed-width="64"
        :width="240"
        show-trigger
        content-style="padding: 12px;"
      >
        <n-h3>Node Tree</n-h3>
        <n-tree
          block-line
          :data="treeData"
          :node-props="nodeProps"
        />
      </n-layout-sider>
      <n-layout-content content-style="padding: 24px;">
        <h2>Node Details</h2>
        <n-card :title="selectedNode?.label || 'No Node Selected'">
           <n-descriptions label-placement="left" bordered :column="1">
            <n-descriptions-item label="Key">
              {{ selectedNode?.key || 'N/A' }}
            </n-descriptions-item>
             <n-descriptions-item label="Type">
              {{ selectedNode?.type || 'N/A' }}
            </n-descriptions-item>
            <n-descriptions-item label="Variables">
              <pre>{{ JSON.stringify(selectedNode?.variables, null, 2) }}</pre>
            </n-descriptions-item>
          </n-descriptions>
        </n-card>
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<script lang="ts" setup>
import { ref, computed } from 'vue'
import { NLayout, NLayoutHeader, NLayoutSider, NLayoutContent, NButton, NInput, NSpace, NTree, NCard, NDescriptions, NDescriptionsItem, NH3 } from 'naive-ui'
import { useWebSocket } from '@/services/websocket'
import { useHubStore } from '@/stores/hub'
import type { Node } from '@/stores/hub'
import type { TreeOption } from 'naive-ui'

const serverUrl = ref('ws://localhost:8080/ws')
const { isConnected, connect, disconnect } = useWebSocket()
const hubStore = useHubStore()

const selectedNode = ref<Node | null>(null)

const treeData = computed(() => hubStore.nodes)

const handleConnect = () => {
  if (isConnected.value) {
    disconnect()
  } else {
    connect(serverUrl.value)
  }
}

const nodeProps = ({ option }: { option: TreeOption }) => {
  return {
    onClick() {
      selectedNode.value = option as Node
    }
  }
}
</script>
