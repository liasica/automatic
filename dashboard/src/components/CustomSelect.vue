<script setup lang="ts">
import { computed, ref, onMounted, onBeforeUnmount } from 'vue'

const props = defineProps<{
  modelValue: string
  options: { value: string; label: string }[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const open = ref(false)
const wrapperRef = ref<HTMLElement>()

const selectedLabel = computed(() => {
  return props.options.find(o => o.value === props.modelValue)?.label ?? ''
})

function select(value: string) {
  emit('update:modelValue', value)
  open.value = false
}

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
      <span>{{ selectedLabel }}</span>
      <svg
        class="h-3.5 w-3.5 text-muted transition-transform duration-200"
        :class="{ 'rotate-180': open }"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        stroke-width="2"
      >
        <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 8.25l-7.5 7.5-7.5-7.5" />
      </svg>
    </button>

    <Transition name="dropdown">
      <div
        v-if="open"
        class="absolute left-0 top-full z-30 mt-1.5 min-w-full overflow-hidden rounded-lg border border-oat bg-white py-1 shadow-sm"
      >
        <button
          v-for="opt in options"
          :key="opt.value"
          class="flex w-full items-center px-3 py-2 text-left text-sm transition-colors"
          :class="opt.value === modelValue
            ? 'bg-cream text-off-black font-medium'
            : 'text-off-black hover:bg-cream/60'"
          @click="select(opt.value)"
        >
          <svg
            v-if="opt.value === modelValue"
            class="mr-2 h-3.5 w-3.5 text-fin"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2.5"
          >
            <path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" />
          </svg>
          <span :class="{ 'pl-5.5': opt.value !== modelValue }">{{ opt.label }}</span>
        </button>
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
