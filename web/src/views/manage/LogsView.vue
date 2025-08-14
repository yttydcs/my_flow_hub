<template>
  <div>
    <n-h2>日志管理</n-h2>

    <n-space vertical size="large">
      <n-card>
        <n-form inline :model="query">
          <n-form-item label="关键词">
            <n-input v-model:value="query.keyword" placeholder="message / 详情关键词" clearable style="width: 240px" />
          </n-form-item>
          <n-form-item label="级别">
            <n-select v-model:value="query.level" :options="levelOptions" clearable style="width: 140px" />
          </n-form-item>
          <n-form-item label="来源">
            <n-input v-model:value="query.source" placeholder="模块/组件名" clearable style="width: 160px" />
          </n-form-item>
          <n-form-item label="时间从">
            <n-date-picker v-model:value="query.startAtMs" type="datetime" clearable style="width: 220px" />
          </n-form-item>
          <n-form-item label="到">
            <n-date-picker v-model:value="query.endAtMs" type="datetime" clearable style="width: 220px" />
          </n-form-item>
          <n-form-item>
            <n-button type="primary" :loading="loading" @click="loadLogs(1)">查询</n-button>
          </n-form-item>
        </n-form>
      </n-card>

  <n-data-table
        :columns="columns"
        :data="rows"
        :loading="loading"
        :pagination="false"
        size="small"
        style="min-height: 200px"
      />

      <div style="display:flex; justify-content:flex-end">
        <n-pagination
          v-model:page="page"
          :page-count="pageCount"
          :page-size="pageSize"
          :page-sizes="[10,20,50,100]"
          show-size-picker
          @update:page="loadLogs"
          @update:page-size="(s:number)=>{ pageSize=s; loadLogs(1) }"
        />
      </div>
    </n-space>
  </div>
  
</template>

<script setup lang="ts">
import { ref, onMounted, h, computed } from 'vue'
import { apiService } from '@/services/api'
import type { SystemLog } from '@/types/api'
import { NH2, NSpace, NCard, NForm, NFormItem, NInput, NSelect, NButton, NDataTable, NPagination, NDatePicker, NTag } from 'naive-ui'

type Row = SystemLog & { key: number }

const levelOptions = [
  { label: 'info', value: 'info' },
  { label: 'warn', value: 'warn' },
  { label: 'error', value: 'error' },
]

const query = ref<{ keyword?: string; level?: string; source?: string; startAtMs?: number | null; endAtMs?: number | null }>({})
const loading = ref(false)
const rows = ref<Row[]>([])
const page = ref(1)
let total = 0
const pageSize = ref(20)

const columns = [
  { title: '时间', key: 'At', render(row: Row) { return new Date(row.At).toLocaleString() } },
  { title: '级别', key: 'Level', render(row: Row) { return h(NTag, { type: row.Level === 'error' ? 'error' : row.Level === 'warn' ? 'warning' : 'default', size: 'small' }, { default: () => row.Level }) } },
  { title: '来源', key: 'Source' },
  { title: '消息', key: 'Message' },
  { title: '详情', key: 'Details', render(row: Row) { return typeof row.Details === 'string' ? row.Details : JSON.stringify(row.Details) } },
]

const pageCount = computed(() => Math.max(1, Math.ceil(total / pageSize.value)))

async function loadLogs(p?: number) {
  if (p) page.value = p
  loading.value = true
  try {
    const params: any = {
      page: page.value,
      pageSize: pageSize.value,
      keyword: query.value.keyword?.trim() || undefined,
      level: query.value.level || undefined,
      source: query.value.source?.trim() || undefined,
      startAt: query.value.startAtMs ? Math.floor(query.value.startAtMs / 1000) : undefined,
      endAt: query.value.endAtMs ? Math.floor(query.value.endAtMs / 1000) : undefined,
    }
    const res = await apiService.getLogs(params)
    const pageData = res.data!
    total = pageData.Total
    rows.value = pageData.Items.map(it => ({ ...it, key: it.ID }))
  } finally {
    loading.value = false
  }
}

onMounted(() => { loadLogs(1) })
</script>

<style scoped>
</style>
