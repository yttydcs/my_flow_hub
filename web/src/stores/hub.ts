import { ref, computed } from 'vue'
import { defineStore } from 'pinia'

export interface Node {
  key: string
  label: string
  isLeaf: boolean
  children?: Node[]
  type?: string
  variables?: any
}

export const useHubStore = defineStore('hub', () => {
  const nodes = ref<Node[]>([])
  const connectionStatus = ref('Disconnected')

  function updateNodeData(data: any) {
    // This is where we'll parse incoming messages and update the tree
    console.log('Updating store with data:', data)
  }

  return { nodes, connectionStatus, updateNodeData }
})
