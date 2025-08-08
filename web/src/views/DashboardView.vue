<template>
  <n-layout style="height: 100vh">
    <n-layout-header style="height: 64px; padding: 24px" bordered>
      MyFlowHub Dashboard
    </n-layout-header>
    <n-layout has-sider>
      <n-layout-sider
        bordered
        collapse-mode="width"
        :collapsed-width="64"
        :width="240"
        show-trigger
      >
        <n-menu
          :collapsed-width="64"
          :collapsed-icon-size="22"
          :options="menuOptions"
        />
      </n-layout-sider>
      <n-layout-content content-style="padding: 24px;">
        <p>Welcome to the Dashboard!</p>
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<script lang="ts" setup>
import { h, ref } from 'vue'
import type { Component } from 'vue'
import { NIcon } from 'naive-ui'
import type { MenuOption } from 'naive-ui'
import {
  BookOutline as BookIcon,
  PersonOutline as PersonIcon,
  WineOutline as WineIcon
} from '@vicons/ionicons5'

function renderIcon (icon: Component) {
  return () => h(NIcon, null, { default: () => h(icon) })
}

const menuOptions: MenuOption[] = [
  {
    label: 'Nodes',
    key: 'nodes',
    icon: renderIcon(BookIcon)
  },
  {
    label: 'Variables',
    key: 'variables',
    icon: renderIcon(PersonIcon)
  }
]
</script>
