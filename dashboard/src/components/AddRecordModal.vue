<script setup lang="ts">
import { ref, watch } from 'vue'
import type { AddRecordParams, UserInfo } from '@/types'

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
const checkTime = ref('')

watch(() => props.show, (val) => {
  if (val) {
    selectedUser.value = props.defaultUserId
    const n = new Date()
    checkTime.value = `${n.getFullYear()}-${pad(n.getMonth() + 1)}-${pad(n.getDate())}T${pad(n.getHours())}:${pad(n.getMinutes())}`
  }
})

function handleSubmit() {
  const formatted = checkTime.value.replace('T', ' ') + ':00'
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
            <select
              v-model="selectedUser"
              class="w-full rounded border border-oat bg-white px-4 py-3 text-sm text-off-black outline-none transition-colors focus:border-off-black"
            >
              <option v-for="user in users" :key="user.id" :value="user.id">
                {{ user.id }}
              </option>
            </select>
          </div>

          <!-- 时间选择 -->
          <div class="mb-8">
            <label class="mb-2 block font-mono text-xs uppercase tracking-wider text-muted">打卡时间</label>
            <input
              v-model="checkTime"
              type="datetime-local"
              class="w-full rounded border border-oat bg-white px-4 py-3 text-sm text-off-black outline-none transition-colors focus:border-off-black"
            >
          </div>

          <div class="flex justify-end gap-3">
            <button
              class="rounded border border-oat bg-white px-5 py-2.5 text-sm text-off-black transition-transform hover:scale-105 active:scale-95"
              @click="emit('close')"
            >
              取消
            </button>
            <button
              class="rounded bg-off-black px-5 py-2.5 text-sm text-white transition-transform hover:scale-105 active:scale-95"
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
