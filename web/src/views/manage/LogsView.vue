<template>
  <div>
    <n-h2>日志管理</n-h2>
    
    <n-space vertical size="large">
      <!-- 功能开发中提示 -->
      <n-alert type="info" title="功能开发中">
        日志管理功能正在开发中，敬请期待...
      </n-alert>

      <!-- 占位内容 -->
      <n-card>
        <n-empty description="日志管理功能开发中">
          <template #extra>
            <n-button size="small">返回主页</n-button>
          </template>
        </n-empty>
      </n-card>
    </n-space>
  </div>
</template>

<script setup lang="ts">
import {
  NH2, NSpace, NAlert, NCard, NEmpty, NButton
} from 'naive-ui'
</script>

<style scoped>
</style>
