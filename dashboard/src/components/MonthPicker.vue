<script setup lang="ts">
import { computed, ref, watch, onMounted, onBeforeUnmount } from 'vue'

const props = defineProps<{
  // 月份值，格式为 YYYY-MM
  modelValue: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const pad = (n: number) => String(n).padStart(2, '0')

const open = ref(false)
const wrapperRef = ref<HTMLElement>()

const parsed = computed(() => ({
  year: Number(props.modelValue.slice(0, 4)),
  month: Number(props.modelValue.slice(5, 7)),
}))

const viewYear = ref(parsed.value.year)

watch(() => props.modelValue, () => {
  viewYear.value = parsed.value.year
})

const n = new Date()
const currentYear = n.getFullYear()
const currentMonth = n.getMonth() + 1

const displayMonth = computed(() => props.modelValue.replace('-', '/'))

function monthStr(year: number, month: number) {
  return `${year}-${pad(month)}`
}

// 触发按钮两侧的快速切换
function stepMonth(delta: number) {
  let y = parsed.value.year
  let m = parsed.value.month + delta
  if (m < 1) { m = 12; y-- }
  if (m > 12) { m = 1; y++ }
  emit('update:modelValue', monthStr(y, m))
}

function selectMonth(m: number) {
  emit('update:modelValue', monthStr(viewYear.value, m))
  open.value = false
}

function goCurrentMonth() {
  emit('update:modelValue', monthStr(currentYear, currentMonth))
  viewYear.value = currentYear
  open.value = false
}

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
    <div class="flex h-8 items-center rounded border border-oat bg-white transition-colors hover:border-sand">
      <button
        class="flex h-full items-center rounded-l px-1.5 text-muted transition-colors hover:bg-cream hover:text-off-black"
        title="上一月"
        @click="stepMonth(-1)"
      >
        <svg
          class="h-3.5 w-3.5"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          stroke-width="2"
        >
          <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 19.5L8.25 12l7.5-7.5" />
        </svg>
      </button>
      <button
        class="flex h-full items-center gap-2 px-2 outline-none"
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
        <span class="font-mono text-xs tracking-tight text-off-black">{{ displayMonth }}</span>
      </button>
      <button
        class="flex h-full items-center rounded-r px-1.5 text-muted transition-colors hover:bg-cream hover:text-off-black"
        title="下一月"
        @click="stepMonth(1)"
      >
        <svg
          class="h-3.5 w-3.5"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          stroke-width="2"
        >
          <path stroke-linecap="round" stroke-linejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
        </svg>
      </button>
    </div>

    <Transition name="dropdown">
      <div
        v-if="open"
        class="absolute left-0 top-full z-30 mt-1.5 w-[232px] rounded-lg border border-oat bg-white p-3 shadow-sm"
      >
        <!-- 年份导航 -->
        <div class="mb-2 flex items-center justify-between">
          <button
            class="rounded p-1 text-muted transition-colors hover:bg-cream hover:text-off-black"
            @click="viewYear--"
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
          <span class="text-sm font-medium tracking-[-0.2px] text-off-black">{{ viewYear }}年</span>
          <button
            class="rounded p-1 text-muted transition-colors hover:bg-cream hover:text-off-black"
            @click="viewYear++"
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

        <!-- 月份网格 -->
        <div class="grid grid-cols-3 gap-1 text-center">
          <button
            v-for="m in 12"
            :key="m"
            class="rounded-full py-2 text-xs transition-colors"
            :class="[
              viewYear === parsed.year && m === parsed.month
                ? 'bg-fin text-white font-medium'
                : viewYear === currentYear && m === currentMonth
                  ? 'text-fin font-medium hover:bg-cream'
                  : 'text-off-black hover:bg-cream',
            ]"
            @click="selectMonth(m)"
          >
            {{ m }}月
          </button>
        </div>

        <!-- 本月快捷按钮 -->
        <div class="mt-2 border-t border-oat pt-2 text-center">
          <button
            class="font-mono text-[10px] uppercase tracking-wider text-fin transition-colors hover:text-off-black"
            @click="goCurrentMonth"
          >
            本月
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
