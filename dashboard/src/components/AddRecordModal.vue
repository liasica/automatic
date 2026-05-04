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

const hourOptions = Array.from({ length: 24 }, (_, i) => ({
  value: pad(i),
  label: pad(i),
}))

const minuteOptions = Array.from({ length: 60 }, (_, i) => ({
  value: pad(i),
  label: pad(i),
}))

watch(() => props.show, val => {
  if (val) {
    selectedUser.value = props.defaultUserId
    const n = new Date()
    pickedDate.value = `${n.getFullYear()}-${pad(n.getMonth() + 1)}-${pad(n.getDate())}`
    pickedHour.value = pad(n.getHours())
    pickedMinute.value = pad(n.getMinutes())
  }
})

const canSubmit = computed(() => !!selectedUser.value && !!pickedDate.value)

function handleSubmit() {
  if (!canSubmit.value) return
  // 随机化秒数，避免固定 :00 偏移
  const second = pad(Math.floor(Math.random() * 60))
  const formatted = `${pickedDate.value} ${pickedHour.value}:${pickedMinute.value}:${second}`
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
                <CustomSelect v-model="pickedHour" :options="hourOptions" />
                <span class="font-mono text-sm text-muted">:</span>
                <CustomSelect v-model="pickedMinute" :options="minuteOptions" />
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
