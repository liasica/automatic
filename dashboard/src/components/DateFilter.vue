<script setup lang="ts">
import { computed, ref } from 'vue'
import type { QueryParams, UserInfo } from '@/types'
import CustomSelect from '@/components/CustomSelect.vue'
import CalendarPicker from '@/components/CalendarPicker.vue'

const props = defineProps<{
  users: UserInfo[]
  loading: boolean
}>()

const emit = defineEmits<{
  query: [params: QueryParams]
}>()

const now = new Date()
const today = now.toISOString().slice(0, 10)
const monthStart = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-01`
const fromDate = ref(monthStart)
const toDate = ref(today)
const selectedUser = ref('all')

const userOptions = computed(() => [
  { value: 'all', label: '全部用户' },
  ...props.users.map(u => ({ value: u.id, label: u.id })),
])

function handleQuery() {
  emit('query', {
    user_id: selectedUser.value === 'all' ? undefined : selectedUser.value,
    from: fromDate.value,
    to: toDate.value || undefined,
  })
}
</script>

<template>
  <div class="flex items-center gap-3">
    <CustomSelect v-model="selectedUser" :options="userOptions" />
    <CalendarPicker v-model="fromDate" />
    <span class="text-sm text-muted">—</span>
    <CalendarPicker v-model="toDate" />

    <button
      class="h-8 rounded border border-off-black bg-white px-4 text-sm text-off-black transition-transform hover:scale-105 active:scale-95 disabled:opacity-40"
      :disabled="loading"
      @click="handleQuery"
    >
      {{ loading ? '查询中...' : '查询' }}
    </button>
  </div>
</template>
