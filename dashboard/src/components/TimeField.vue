<script setup lang="ts">
import { computed, ref, watch } from 'vue'

const props = defineProps<{
  // 时间值，格式为 HH:MM
  modelValue: string
  disabled?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const pad = (n: number) => String(n).padStart(2, '0')

const hour = ref('00')
const minute = ref('00')

watch(() => props.modelValue, v => {
  const [h = '', m = ''] = v.split(':')
  hour.value = h
  minute.value = m
}, { immediate: true })

// 仅保留数字字符
function sanitizeDigits(v: string, max = 2) {
  return v.replace(/\D/g, '').slice(0, max)
}

function emitValue() {
  emit('update:modelValue', `${hour.value}:${minute.value}`)
}

function onHourInput(e: Event) {
  hour.value = sanitizeDigits((e.target as HTMLInputElement).value)
  emitValue()
}

function onMinuteInput(e: Event) {
  minute.value = sanitizeDigits((e.target as HTMLInputElement).value)
  emitValue()
}

// 失焦时校验范围并补零
function normalizeHour() {
  const n = Number.parseInt(hour.value, 10)
  hour.value = Number.isNaN(n) ? '00' : pad(Math.min(23, Math.max(0, n)))
  emitValue()
}

function normalizeMinute() {
  const n = Number.parseInt(minute.value, 10)
  minute.value = Number.isNaN(n) ? '00' : pad(Math.min(59, Math.max(0, n)))
  emitValue()
}

const hourValid = computed(() => {
  const n = Number.parseInt(hour.value, 10)
  return !Number.isNaN(n) && n >= 0 && n <= 23
})

const minuteValid = computed(() => {
  const n = Number.parseInt(minute.value, 10)
  return !Number.isNaN(n) && n >= 0 && n <= 59
})
</script>

<template>
  <div class="flex items-center gap-1">
    <input
      :value="hour"
      type="text"
      inputmode="numeric"
      maxlength="2"
      placeholder="HH"
      :disabled="disabled"
      class="w-14 rounded border border-oat bg-white px-2 py-1.5 text-center font-mono text-sm text-off-black outline-none focus:border-off-black disabled:cursor-not-allowed disabled:opacity-40"
      :class="{ 'border-red-400': !disabled && !hourValid }"
      @input="onHourInput"
      @blur="normalizeHour"
    >
    <span class="font-mono text-sm text-muted" :class="{ 'opacity-40': disabled }">:</span>
    <input
      :value="minute"
      type="text"
      inputmode="numeric"
      maxlength="2"
      placeholder="MM"
      :disabled="disabled"
      class="w-14 rounded border border-oat bg-white px-2 py-1.5 text-center font-mono text-sm text-off-black outline-none focus:border-off-black disabled:cursor-not-allowed disabled:opacity-40"
      :class="{ 'border-red-400': !disabled && !minuteValid }"
      @input="onMinuteInput"
      @blur="normalizeMinute"
    >
  </div>
</template>
