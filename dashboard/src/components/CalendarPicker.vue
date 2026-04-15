<script setup lang="ts">
import { computed, ref, watch, onMounted, onBeforeUnmount } from 'vue'

const props = defineProps<{
  modelValue: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const open = ref(false)
const wrapperRef = ref<HTMLElement>()

const parsed = computed(() => ({
  year: Number(props.modelValue.slice(0, 4)),
  month: Number(props.modelValue.slice(5, 7)),
}))

const viewYear = ref(parsed.value.year)
const viewMonth = ref(parsed.value.month)

watch(() => props.modelValue, () => {
  viewYear.value = parsed.value.year
  viewMonth.value = parsed.value.month
})

const weekdays = ['一', '二', '三', '四', '五', '六', '日']

const monthLabel = computed(() => `${viewYear.value}年${viewMonth.value}月`)

const daysInMonth = computed(() => new Date(viewYear.value, viewMonth.value, 0).getDate())

const firstDayOffset = computed(() => {
  const dow = new Date(viewYear.value, viewMonth.value - 1, 1).getDay()
  return dow === 0 ? 6 : dow - 1
})

interface CalDay {
  day: number
  dateStr: string
  isCurrentMonth: boolean
  isToday: boolean
  isSelected: boolean
}

const todayStr = new Date().toISOString().slice(0, 10)

const calendarDays = computed<CalDay[]>(() => {
  const days: CalDay[] = []

  const prevYear = viewMonth.value === 1 ? viewYear.value - 1 : viewYear.value
  const prevMonth = viewMonth.value === 1 ? 12 : viewMonth.value - 1
  const prevDays = new Date(prevYear, prevMonth, 0).getDate()

  for (let i = firstDayOffset.value - 1; i >= 0; i--) {
    const d = prevDays - i
    const ds = `${prevYear}-${String(prevMonth).padStart(2, '0')}-${String(d).padStart(2, '0')}`
    days.push({ day: d, dateStr: ds, isCurrentMonth: false, isToday: ds === todayStr, isSelected: ds === props.modelValue })
  }

  for (let d = 1; d <= daysInMonth.value; d++) {
    const ds = `${viewYear.value}-${String(viewMonth.value).padStart(2, '0')}-${String(d).padStart(2, '0')}`
    days.push({ day: d, dateStr: ds, isCurrentMonth: true, isToday: ds === todayStr, isSelected: ds === props.modelValue })
  }

  const nextYear = viewMonth.value === 12 ? viewYear.value + 1 : viewYear.value
  const nextMonth = viewMonth.value === 12 ? 1 : viewMonth.value + 1
  const totalCells = Math.ceil(days.length / 7) * 7
  for (let d = 1; days.length < totalCells; d++) {
    const ds = `${nextYear}-${String(nextMonth).padStart(2, '0')}-${String(d).padStart(2, '0')}`
    days.push({ day: d, dateStr: ds, isCurrentMonth: false, isToday: ds === todayStr, isSelected: ds === props.modelValue })
  }

  return days
})

function prevMonthNav() {
  if (viewMonth.value === 1) { viewMonth.value = 12; viewYear.value-- }
  else viewMonth.value--
}

function nextMonthNav() {
  if (viewMonth.value === 12) { viewMonth.value = 1; viewYear.value++ }
  else viewMonth.value++
}

function selectDate(d: CalDay) {
  emit('update:modelValue', d.dateStr)
  open.value = false
}

function goToday() {
  emit('update:modelValue', todayStr)
  viewYear.value = new Date().getFullYear()
  viewMonth.value = new Date().getMonth() + 1
  open.value = false
}

const displayDate = computed(() => props.modelValue.replace(/-/g, '/'))

// 日历图标 path（放在脚本里避免模板行超长）
const calendarIconPath = 'M6.75 3v2.25M17.25 3v2.25M3 18.75V7.5a2.25 2.25 0 012.25-2.25h13.5A2.25 2.25 0 0121 7.5v11.25m-18 0A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75m-18 0v-7.5A2.25 2.25 0 015.25 9h13.5A2.25 2.25 0 0121 11.25v7.5'

function onClickOutside(e: MouseEvent) {
  if (!wrapperRef.value?.contains(e.target as Node)) {
    open.value = false
  }
}

onMounted(() => document.addEventListener('click', onClickOutside, true))
onBeforeUnmount(() => document.removeEventListener('click', onClickOutside, true))
</script>

<template>
  <div ref="wrapperRef" class="relative">
    <button
      class="flex h-8 items-center gap-2 rounded border border-oat bg-white px-3 text-sm text-off-black outline-none transition-colors hover:border-sand focus:border-off-black"
      @click="open = !open"
    >
      <svg
        class="h-4 w-4 text-muted"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        stroke-width="1.5"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          :d="calendarIconPath"
        />
      </svg>
      <span class="font-mono text-xs tracking-tight">{{ displayDate }}</span>
    </button>

    <Transition name="dropdown">
      <div
        v-if="open"
        class="absolute left-0 top-full z-30 mt-1.5 w-[272px] rounded-lg border border-oat bg-white p-3 shadow-sm"
      >
        <!-- 月份导航 -->
        <div class="mb-2 flex items-center justify-between">
          <button
            class="rounded p-1 text-muted transition-colors hover:bg-cream hover:text-off-black"
            @click="prevMonthNav"
          >
            <svg
              class="h-4 w-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="2"
            >
              <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 19.5L8.25 12l7.5-7.5" />
            </svg>
          </button>
          <span class="text-sm font-medium tracking-[-0.2px] text-off-black">{{ monthLabel }}</span>
          <button
            class="rounded p-1 text-muted transition-colors hover:bg-cream hover:text-off-black"
            @click="nextMonthNav"
          >
            <svg
              class="h-4 w-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="2"
            >
              <path stroke-linecap="round" stroke-linejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
            </svg>
          </button>
        </div>

        <!-- 星期标题 -->
        <div class="mb-1 grid grid-cols-7 text-center">
          <span v-for="wd in weekdays" :key="wd" class="py-1 font-mono text-[10px] uppercase tracking-wider text-muted">
            {{ wd }}
          </span>
        </div>

        <!-- 日期网格 -->
        <div class="grid grid-cols-7 gap-0.5 text-center">
          <button
            v-for="(d, i) in calendarDays"
            :key="i"
            class="flex h-[34px] w-full items-center justify-center rounded-full text-xs transition-colors"
            :class="[
              d.isSelected
                ? 'bg-fin text-white font-medium'
                : d.isToday
                  ? 'text-fin font-medium hover:bg-cream'
                  : d.isCurrentMonth
                    ? 'text-off-black hover:bg-cream'
                    : 'text-oat hover:bg-cream/60',
            ]"
            @click="selectDate(d)"
          >
            {{ d.day }}
          </button>
        </div>

        <!-- 今天快捷按钮 -->
        <div class="mt-2 border-t border-oat pt-2 text-center">
          <button
            class="font-mono text-[10px] uppercase tracking-wider text-fin transition-colors hover:text-off-black"
            @click="goToday"
          >
            今天
          </button>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.dropdown-enter-active,
.dropdown-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>
