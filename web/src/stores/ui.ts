import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export const useUIStore = defineStore('ui', () => {
  const darkMode = ref<boolean>(
    localStorage.getItem('ui.darkMode')
      ? localStorage.getItem('ui.darkMode') === '1'
      : (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches)
  )

  watch(darkMode, (v) => {
    localStorage.setItem('ui.darkMode', v ? '1' : '0')
    document.documentElement.setAttribute('data-theme', v ? 'dark' : 'light')
  }, { immediate: true })

  return { darkMode }
})
