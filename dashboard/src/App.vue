<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useAttendance } from '@/composables/useAttendance'
import { useOvertime } from '@/composables/useOvertime'
import type { AddRecordParams, DayFilter, QueryParams } from '@/types'
import DateFilter from '@/components/DateFilter.vue'
import RecordTable from '@/components/RecordTable.vue'
import AddRecordModal from '@/components/AddRecordModal.vue'
import DeleteConfirmModal from '@/components/DeleteConfirmModal.vue'

const { groups, users, loading, error, fetchUsers, queryRecords, addRecord, deleteRecords } = useAttendance()
const overtime = useOvertime()

const showAddModal = ref(false)
const showDeleteModal = ref(false)
const deleteTargetIds = ref<string[]>([])
const errorMessage = ref<string | null>(null)
let errorTimer: ReturnType<typeof setTimeout> | null = null
// 加班写入 + 重查全过程都处于 saving 状态，用于 UI 展示保存中
const overtimeSaving = ref(false)

const now = new Date()
const today = now.toISOString().slice(0, 10)
const monthStart = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-01`
const lastQuery = ref<QueryParams>({ from: monthStart, to: today })
const dayFilter = ref<DayFilter>('all')

const defaultUserId = computed(() => users.value[0]?.id ?? '')
const overtimeAuthorized = computed(() => overtime.authorized.value === true)

const dayFilterOptions: { value: DayFilter; label: string }[] = [
  { value: 'all', label: '全部' },
  { value: 'workday', label: '工作日' },
  { value: 'non_workday', label: '非工作日' },
  { value: 'weekend', label: '周末' },
  { value: 'holiday', label: '节假日' },
]

function showError(msg: string) {
  errorMessage.value = msg
  if (errorTimer) clearTimeout(errorTimer)
  errorTimer = setTimeout(() => {
    errorMessage.value = null
  }, 5000)
}

async function handleQuery(params: QueryParams) {
  lastQuery.value = params
  await Promise.all([
    queryRecords(params),
    overtime.queryRecords(),
  ])
  if (error.value) showError(error.value)
  if (overtime.error.value) showError(overtime.error.value)
}

async function handleOvertimeUpdate(recordId: string, fields: Record<string, unknown>) {
  overtimeSaving.value = true
  try {
    const success = await overtime.updateRecord(recordId, fields)
    if (success) {
      await overtime.queryRecords()
    }
    else if (overtime.error.value) showError(overtime.error.value)
  }
  finally {
    overtimeSaving.value = false
  }
}

async function handleOvertimeCreate(fields: Record<string, unknown>) {
  overtimeSaving.value = true
  try {
    const record = await overtime.createRecord(fields)
    if (record) {
      await overtime.queryRecords()
    }
    else if (overtime.error.value) showError(overtime.error.value)
  }
  finally {
    overtimeSaving.value = false
  }
}

async function handleAdd(params: AddRecordParams) {
  const success = await addRecord(params)
  if (success) {
    showAddModal.value = false
    await queryRecords(lastQuery.value)
  }
  else if (error.value) showError(error.value)
}

function handleDeleteClick(recordIds: string[]) {
  deleteTargetIds.value = recordIds
  showDeleteModal.value = true
}

async function handleDeleteConfirm() {
  const result = await deleteRecords(deleteTargetIds.value)
  showDeleteModal.value = false
  if (result) {
    if (result.failIds.length > 0) showError(`部分记录删除失败: ${result.failIds.join(', ')}`)
    await queryRecords(lastQuery.value)
  }
  else if (error.value) showError(error.value)
}

onMounted(async () => {
  await fetchUsers()
  await Promise.all([
    queryRecords(lastQuery.value),
    overtime.fetchSchema(),
    overtime.queryRecords(),
  ])
})
</script>

<template>
  <div class="min-h-screen">
    <!-- 错误提示 -->
    <Transition name="fade">
      <div
        v-if="errorMessage"
        class="fixed left-1/2 top-5 z-50 -translate-x-1/2 rounded-lg border border-report-red/20 bg-white px-5 py-2.5 text-sm text-report-red shadow-sm"
      >
        {{ errorMessage }}
      </div>
    </Transition>

    <!-- 顶部导航栏 -->
    <header class="border-b border-oat bg-white">
      <div class="mx-auto flex max-w-5xl items-center justify-between px-8 py-4">
        <div class="flex items-baseline gap-2">
          <h1 class="text-xl font-medium tracking-[-0.48px] text-off-black">
            Automatic
          </h1>
          <span class="text-sm text-muted">Dashboard</span>
        </div>
        <div class="flex items-center gap-3">
          <!-- 加班授权状态 -->
          <a
            v-if="!overtimeAuthorized"
            href="/api/auth/feishu"
            class="inline-flex items-center gap-1.5 rounded border border-fin/20 bg-fin/5 px-3 py-1 text-xs text-fin transition-colors hover:bg-fin/10"
          >
            <svg
              class="h-3 w-3"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="2"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M13.5 10.5V6.75a4.5 4.5 0 119 0v3.75M3.75 21.75h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H3.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z"
              />
            </svg>
            授权加班表格
          </a>
          <span
            v-else
            class="inline-flex items-center gap-1 rounded border border-report-green/20 bg-report-green/5 px-3 py-1 text-xs text-report-green"
          >
            <svg
              class="h-3 w-3"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="2"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M4.5 12.75l6 6 9-13.5"
              />
            </svg>
            加班已关联
          </span>
          <div class="rounded bg-cream px-3 py-1 font-mono text-xs text-muted">
            {{ today }}
          </div>
        </div>
      </div>
    </header>

    <!-- 主内容 -->
    <main class="mx-auto max-w-5xl px-8 py-8">
      <!-- 筛选 + 操作区 -->
      <div class="mb-8 space-y-3">
        <div class="flex items-center justify-between">
          <DateFilter :users="users" :loading="loading" @query="handleQuery" />
          <button
            class="h-8 rounded bg-off-black px-4 text-sm text-white transition-transform hover:scale-105 active:scale-95"
            @click="showAddModal = true"
          >
            + 添加记录
          </button>
        </div>

        <!-- 日期类型筛选 -->
        <div class="flex items-center gap-1.5">
          <button
            v-for="opt in dayFilterOptions"
            :key="opt.value"
            class="rounded px-2.5 py-1 font-mono text-xs transition-colors"
            :class="dayFilter === opt.value
              ? 'bg-off-black text-white'
              : 'text-muted hover:bg-cream hover:text-off-black'"
            @click="dayFilter = opt.value"
          >
            {{ opt.label }}
          </button>
        </div>
      </div>

      <!-- 记录表格 -->
      <RecordTable
        :groups="groups"
        :users="users"
        :loading="loading"
        :day-filter="dayFilter"
        :overtime-records="overtime.records.value"
        :overtime-schema="overtime.schema.value"
        :overtime-saving="overtimeSaving"
        @delete="handleDeleteClick"
        @overtime-update="handleOvertimeUpdate"
        @overtime-create="handleOvertimeCreate"
      />
    </main>

    <!-- 弹窗 -->
    <AddRecordModal
      :show="showAddModal"
      :users="users"
      :default-user-id="defaultUserId"
      @close="showAddModal = false"
      @submit="handleAdd"
    />
    <DeleteConfirmModal
      :show="showDeleteModal"
      :record-ids="deleteTargetIds"
      @close="showDeleteModal = false"
      @confirm="handleDeleteConfirm"
    />
  </div>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease, transform 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translate(-50%, -10px);
}
</style>
