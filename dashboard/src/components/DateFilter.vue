<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { QueryParams, UserInfo } from '@/types'
import CustomSelect from '@/components/CustomSelect.vue'
import MonthPicker from '@/components/MonthPicker.vue'

const props = defineProps<{
  users: UserInfo[]
  loading: boolean
}>()

const emit = defineEmits<{
  query: [params: QueryParams]
}>()

const pad = (n: number) => String(n).padStart(2, '0')

const now = new Date()
const month = ref(`${now.getFullYear()}-${pad(now.getMonth() + 1)}`)
const selectedUser = ref('all')

const userOptions = computed(() => [
  { value: 'all', label: '全部用户' },
  ...props.users.map(u => ({ value: u.id, label: u.id })),
])

// 由所选月份计算查询范围（月首 ~ 月末）
function monthRange(m: string) {
  const year = Number(m.slice(0, 4))
  const mon = Number(m.slice(5, 7))
  const lastDay = new Date(year, mon, 0).getDate()
  return { from: `${m}-01`, to: `${m}-${pad(lastDay)}` }
}

// 月份或用户变化时自动查询
watch([month, selectedUser], () => {
  const { from, to } = monthRange(month.value)
  emit('query', {
    user_id: selectedUser.value === 'all' ? undefined : selectedUser.value,
    from,
    to,
  })
})
</script>

<template>
  <div class="flex items-center gap-3">
    <CustomSelect v-model="selectedUser" :options="userOptions" />
    <MonthPicker v-model="month" />

    <!-- 查询中指示 -->
    <Transition name="fade">
      <span v-if="loading" class="flex items-center gap-1.5 font-mono text-xs text-muted">
        <svg
          class="h-3.5 w-3.5 animate-spin"
          fill="none"
          viewBox="0 0 24 24"
        >
          <circle
            class="opacity-25"
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            stroke-width="3"
          />
          <path
            class="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8v3a5 5 0 00-5 5H4z"
          />
        </svg>
        查询中
      </span>
    </Transition>
  </div>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
