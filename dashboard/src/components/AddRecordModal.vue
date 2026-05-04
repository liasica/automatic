<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { AddRecordParams, UserInfo } from '@/types'
import CustomSelect from '@/components/CustomSelect.vue'
import CalendarPicker from '@/components/CalendarPicker.vue'

const props = defineProps<{
  show: boolean
  users: UserInfo[]
  defaultUserId: string
}>()

const emit = defineEmits<{
  close: []
  submit: [params: AddRecordParams]
}>()

const pad = (n: number) => String(n).padStart(2, '0')

const selectedUser = ref('')
const pickedDate = ref('')
const pickedHour = ref('09')
const pickedMinute = ref('00')

const userOptions = computed(() =>
  props.users.map(u => ({ value: u.id, label: u.id })),
)

watch(() => props.show, val => {
  if (val) {
    selectedUser.value = props.defaultUserId
    const n = new Date()
    pickedDate.value = `${n.getFullYear()}-${pad(n.getMonth() + 1)}-${pad(n.getDate())}`
    pickedHour.value = pad(n.getHours())
    pickedMinute.value = pad(n.getMinutes())
  }
})

// 仅保留数字字符
function sanitizeDigits(v: string, max = 2) {
  return v.replace(/\D/g, '').slice(0, max)
}

function onHourInput(e: Event) {
  pickedHour.value = sanitizeDigits((e.target as HTMLInputElement).value)
}

function onMinuteInput(e: Event) {
  pickedMinute.value = sanitizeDigits((e.target as HTMLInputElement).value)
}

// 失焦时校验范围并补零
function normalizeHour() {
  const n = Number.parseInt(pickedHour.value, 10)
  if (Number.isNaN(n)) {
    pickedHour.value = '00'
    return
  }
  pickedHour.value = pad(Math.min(23, Math.max(0, n)))
}

function normalizeMinute() {
  const n = Number.parseInt(pickedMinute.value, 10)
  if (Number.isNaN(n)) {
    pickedMinute.value = '00'
    return
  }
  pickedMinute.value = pad(Math.min(59, Math.max(0, n)))
}

const hourValid = computed(() => {
  const n = Number.parseInt(pickedHour.value, 10)
  return !Number.isNaN(n) && n >= 0 && n <= 23
})

const minuteValid = computed(() => {
  const n = Number.parseInt(pickedMinute.value, 10)
  return !Number.isNaN(n) && n >= 0 && n <= 59
})

const canSubmit = computed(
  () => !!selectedUser.value && !!pickedDate.value && hourValid.value && minuteValid.value,
)

function handleSubmit() {
  if (!canSubmit.value) return
  // 随机化秒数，避免固定 :00 偏移
  const second = pad(Math.floor(Math.random() * 60))
  const hh = pad(Number.parseInt(pickedHour.value, 10))
  const mm = pad(Number.parseInt(pickedMinute.value, 10))
  const formatted = `${pickedDate.value} ${hh}:${mm}:${second}`
  emit('submit', {
    user_id: selectedUser.value,
    check_time: formatted,
  })
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="show" class="fixed inset-0 z-50 flex items-center justify-center">
        <div class="absolute inset-0 bg-off-black/40" @click="emit('close')" />
        <div class="relative z-10 w-full max-w-md rounded-lg border border-oat bg-white p-8 shadow-sm">
          <div class="mb-8">
            <h3 class="text-lg font-medium tracking-[-0.48px] text-off-black">
              添加打卡记录
            </h3>
            <p class="mt-1 text-xs text-muted">
              手动创建一条打卡流水记录
            </p>
          </div>

          <!-- 用户选择 -->
          <div class="mb-5">
            <label class="mb-2 block font-mono text-xs uppercase tracking-wider text-muted">用户</label>
            <CustomSelect v-model="selectedUser" :options="userOptions" class="w-full [&>button]:w-full [&>button]:justify-between" />
          </div>

          <!-- 打卡时间 -->
          <div class="mb-8">
            <label class="mb-2 block font-mono text-xs uppercase tracking-wider text-muted">打卡时间</label>
            <div class="flex flex-wrap items-center gap-2">
              <CalendarPicker v-model="pickedDate" />
              <div class="flex items-center gap-1">
                <input
                  :value="pickedHour"
                  type="text"
                  inputmode="numeric"
                  maxlength="2"
                  placeholder="HH"
                  class="w-14 rounded border border-oat bg-white px-2 py-1.5 text-center font-mono text-sm text-off-black outline-none focus:border-off-black"
                  :class="{ 'border-red-400': !hourValid }"
                  @input="onHourInput"
                  @blur="normalizeHour"
                >
                <span class="font-mono text-sm text-muted">:</span>
                <input
                  :value="pickedMinute"
                  type="text"
                  inputmode="numeric"
                  maxlength="2"
                  placeholder="MM"
                  class="w-14 rounded border border-oat bg-white px-2 py-1.5 text-center font-mono text-sm text-off-black outline-none focus:border-off-black"
                  :class="{ 'border-red-400': !minuteValid }"
                  @input="onMinuteInput"
                  @blur="normalizeMinute"
                >
              </div>
            </div>
          </div>

          <div class="flex justify-end gap-3">
            <button
              class="rounded border border-oat bg-white px-5 py-2.5 text-sm text-off-black transition-transform hover:scale-105 active:scale-95"
              @click="emit('close')"
            >
              取消
            </button>
            <button
              class="rounded bg-off-black px-5 py-2.5 text-sm text-white transition-transform hover:scale-105 active:scale-95 disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:scale-100"
              :disabled="!canSubmit"
              @click="handleSubmit"
            >
              确认添加
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.2s ease;
}

.modal-enter-active > div:last-child,
.modal-leave-active > div:last-child {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from > div:last-child,
.modal-leave-to > div:last-child {
  transform: scale(0.96);
}
</style>
