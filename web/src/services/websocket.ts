import { ref } from 'vue'
import { useMessage } from 'naive-ui'
import { useHubStore } from '@/stores/hub'

const ws = ref<WebSocket | null>(null)
const isConnected = ref(false)

export function useWebSocket() {
  const message = useMessage()
  const hubStore = useHubStore()

  const connect = (url: string) => {
    if (ws.value) {
      message.warning('Already connected.')
      return
    }

    ws.value = new WebSocket(url)

    ws.value.onopen = () => {
      isConnected.value = true
      message.success('WebSocket Connected')
    }

    ws.value.onclose = () => {
      isConnected.value = false
      ws.value = null
      message.info('WebSocket Disconnected')
    }

    ws.value.onerror = (error) => {
      message.error('WebSocket Error')
      console.error('WebSocket Error:', error)
    }

    ws.value.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        hubStore.updateNodeData(data)
      } catch (e) {
        console.error("Failed to parse incoming message:", e)
      }
    }
  }

  const disconnect = () => {
    if (ws.value) {
      ws.value.close()
    }
  }

  const sendMessage = (data: any) => {
    if (ws.value && isConnected.value) {
      ws.value.send(JSON.stringify(data))
    } else {
      message.error('Not connected to WebSocket.')
    }
  }

  return {
    ws,
    isConnected,
    connect,
    disconnect,
    sendMessage
  }
}
