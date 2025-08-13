<script setup lang="ts">
import { RouterView } from 'vue-router'
import { computed } from 'vue'
import { NMessageProvider, NDialogProvider, NConfigProvider, NGlobalStyle, darkTheme } from 'naive-ui'
import { useUIStore } from '@/stores/ui'

const ui = useUIStore()
const theme = computed(() => (ui.darkMode ? darkTheme : undefined))
</script>

<template>
  <n-config-provider :theme="theme">
    <n-message-provider>
      <n-dialog-provider>
        <RouterView />
      </n-dialog-provider>
    </n-message-provider>
    <n-global-style />
  </n-config-provider>
</template>

<style>
html, body {
  height: 100%;
  margin: 0;
  padding: 0;
  background-color: #f5f5f5;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', sans-serif;
}

#app {
  height: 100vh;
}
</style>
