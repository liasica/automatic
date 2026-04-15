<script setup lang="ts">
defineProps<{
  show: boolean
  recordIds: string[]
}>()

const emit = defineEmits<{
  close: []
  confirm: []
}>()
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="show" class="fixed inset-0 z-50 flex items-center justify-center">
        <!-- 遮罩 -->
        <div class="absolute inset-0 bg-off-black/40" @click="emit('close')" />

        <!-- 弹窗 -->
        <div class="relative z-10 w-full max-w-sm rounded-lg border border-oat bg-white p-8 shadow-sm">
          <div class="mb-6">
            <h3 class="text-lg font-medium tracking-[-0.48px] text-off-black">
              确认删除
            </h3>
            <p class="mt-3 text-sm text-muted">
              {{ recordIds.length > 1
                ? `此操作将永久删除 ${recordIds.length} 条打卡记录，无法撤销`
                : '此操作将永久删除该打卡记录，无法撤销'
              }}
            </p>
          </div>

          <!-- 记录 ID -->
          <div class="mb-8 max-h-32 space-y-1 overflow-y-auto rounded border border-report-red/20 bg-report-red/5 px-4 py-3">
            <div v-for="id in recordIds" :key="id" class="font-mono text-xs text-report-red">
              {{ id }}
            </div>
          </div>

          <!-- 操作按钮 -->
          <div class="flex justify-end gap-3">
            <button
              class="rounded border border-oat bg-white px-5 py-2.5 text-sm text-off-black transition-transform hover:scale-105 active:scale-95"
              @click="emit('close')"
            >
              取消
            </button>
            <button
              class="rounded bg-report-red px-5 py-2.5 text-sm font-medium text-white transition-transform hover:scale-105 active:scale-95"
              @click="emit('confirm')"
            >
              确认删除
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
