<script setup lang="ts">
import { RouterView } from 'vue-router'
import { NMessageProvider, NDialogProvider, NConfigProvider, NGlobalStyle } from 'naive-ui'
</script>

<template>
  <n-config-provider>
    <n-message-provider>
      <n-dialog-provider>
        <RouterView />
      </n-dialog-provider>
    </n-message-provider>
    <n-global-style />
  </n-config-provider>
</template>
