<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { AddRecordParams, AttendanceRecord, UserInfo } from '@/types'
import { queryUserRecords } from '@/composables/useAttendance'
import CustomSelect from '@/components/CustomSelect.vue'
import CalendarPicker from '@/components/CalendarPicker.vue'
import TimeField from '@/components/TimeField.vue'

const props = defineProps<{
  show: boolean
  users: UserInfo[]
  defaultUserId: string
}>()

const emit = defineEmits<{
  close: []
  submit: [params: AddRecordParams[]]
}>()

const pad = (n: number) => String(n).padStart(2, '0')

const selectedUser = ref('')
const pickedDate = ref('')

// 补卡时间模式：random 按最近七天历史随机，normal 固定正常上下班时间
const timeMode = ref<'random' | 'normal'>('random')

// 正常上下班时间
const NORMAL_CHECK_IN = '09:00'
const NORMAL_CHECK_OUT = '18:00'

const timeModes = [
  { value: 'random', label: '随机' },
  { value: 'normal', label: '正常' },
] as const

const checkInTime = ref('09:00')
const checkOutTime = ref('18:00')
// 用户手动改过时间后，不再被预填覆盖
const checkInDirty = ref(false)
const checkOutDirty = ref(false)
// 是否添加对应记录，未勾选的不提交
const addCheckIn = ref(true)
const addCheckOut = ref(true)

// 选中日期的已有打卡记录
const dayRecords = ref<AttendanceRecord[]>([])
const dayLoading = ref(false)
const dayError = ref<string | null>(null)

const userOptions = computed(() =>
  props.users.map(u => ({ value: u.id, label: u.id })),
)

const currentUser = computed(() => props.users.find(u => u.id === selectedUser.value))

watch(() => props.show, val => {
  if (val) {
    selectedUser.value = props.defaultUserId
    addCheckIn.value = true
    addCheckOut.value = true
    // 直接赋值而非经 setTimeMode，避免与 prefillKey watch 重复触发预填
    timeMode.value = 'random'
    const n = new Date()
    pickedDate.value = `${n.getFullYear()}-${pad(n.getMonth() + 1)}-${pad(n.getDate())}`
  }
})

// 弹窗打开或切换用户时重新预填时间；computed 自带去重
const prefillKey = computed(() => (props.show && currentUser.value ? currentUser.value.id : ''))

let prefillToken = 0

watch(prefillKey, key => {
  if (key) prefillTimes()
})

// 预填补卡时间：先按用户配置兜底，再根据最近七天的打卡记录随机到常用时间
async function prefillTimes() {
  const u = currentUser.value
  if (!u) return
  checkInDirty.value = false
  checkOutDirty.value = false
  // 正常模式固定预填，不查询历史
  if (timeMode.value === 'normal') {
    prefillToken++
    checkInTime.value = NORMAL_CHECK_IN
    checkOutTime.value = NORMAL_CHECK_OUT
    return
  }
  const token = ++prefillToken
  checkInTime.value = u.check_in_latest || '09:00'
  checkOutTime.value = u.check_out_earliest || '18:00'

  const fmt = (d: Date) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
  const end = new Date()
  const start = new Date(end.getTime() - 6 * 86400000)
  let records: AttendanceRecord[]
  try {
    records = await queryUserRecords(u.id, fmt(start), fmt(end))
  }
  catch {
    // 查询失败时保持配置兜底值
    return
  }
  if (token !== prefillToken) return

  const toMinutes = (r: AttendanceRecord) =>
    Number(r.check_time.slice(11, 13)) * 60 + Number(r.check_time.slice(14, 16))
  const checkIns = records.filter(r => r.label === 'checkin').map(toMinutes)
  const checkOuts = records.filter(r => r.label === 'checkout').map(toMinutes)

  if (!checkInDirty.value && checkIns.length > 0) checkInTime.value = randomUsualTime(checkIns)
  if (!checkOutDirty.value && checkOuts.length > 0) checkOutTime.value = randomUsualTime(checkOuts)
}

// 切换补卡时间模式：覆盖已填时间（含手动修改过的）
function setTimeMode(mode: 'random' | 'normal') {
  if (timeMode.value === mode) return
  timeMode.value = mode
  if (mode === 'normal') {
    // 使进行中的随机预填请求失效，避免异步结果覆盖固定值
    prefillToken++
    checkInDirty.value = false
    checkOutDirty.value = false
    checkInTime.value = NORMAL_CHECK_IN
    checkOutTime.value = NORMAL_CHECK_OUT
  }
  else {
    prefillTimes()
  }
}

// 在历史打卡时刻的 [最早, 最晚] 区间内随机取值；仅一个时刻时做 ±5 分钟抖动
function randomUsualTime(minutes: number[]) {
  const min = Math.min(...minutes)
  const max = Math.max(...minutes)
  let v: number
  if (min === max) {
    v = min + Math.floor(Math.random() * 11) - 5
  }
  else {
    v = min + Math.floor(Math.random() * (max - min + 1))
  }
  v = Math.min(23 * 60 + 59, Math.max(0, v))
  return `${pad(Math.floor(v / 60))}:${pad(v % 60)}`
}

function onCheckInInput(v: string) {
  checkInTime.value = v
  checkInDirty.value = true
}

function onCheckOutInput(v: string) {
  checkOutTime.value = v
  checkOutDirty.value = true
}

// 用户 + 日期变化时查询该日已有记录；computed 自带去重，弹窗关闭时为空串不触发
const fetchKey = computed(() =>
  props.show && selectedUser.value && pickedDate.value
    ? `${selectedUser.value}|${pickedDate.value}`
    : '',
)

let fetchToken = 0

watch(fetchKey, key => {
  if (key) loadDayRecords()
})

async function loadDayRecords() {
  const token = ++fetchToken
  dayLoading.value = true
  dayError.value = null
  dayRecords.value = []
  try {
    const records = await queryUserRecords(selectedUser.value, pickedDate.value)
    if (token !== fetchToken) return
    dayRecords.value = records
  }
  catch (e) {
    if (token !== fetchToken) return
    dayError.value = e instanceof Error ? e.message : '查询打卡记录失败'
  }
  finally {
    if (token === fetchToken) dayLoading.value = false
  }
}

const hasCheckIn = computed(() => dayRecords.value.some(r => r.label === 'checkin'))
const hasCheckOut = computed(() => dayRecords.value.some(r => r.label === 'checkout'))

// 查询完成后才决定需要补哪些时间输入
const dayReady = computed(() => !dayLoading.value && !dayError.value)
const needCheckIn = computed(() => dayReady.value && !hasCheckIn.value)
const needCheckOut = computed(() => dayReady.value && !hasCheckOut.value)
const dayComplete = computed(() => dayReady.value && hasCheckIn.value && hasCheckOut.value)

function validTime(v: string) {
  const m = /^(\d{1,2}):(\d{1,2})$/.exec(v)
  if (!m) return false
  return Number(m[1]) <= 23 && Number(m[2]) <= 59
}

// 实际要提交的项 = 缺失且已勾选
const wantCheckIn = computed(() => needCheckIn.value && addCheckIn.value)
const wantCheckOut = computed(() => needCheckOut.value && addCheckOut.value)

const canSubmit = computed(() => {
  if (!selectedUser.value || !pickedDate.value) return false
  if (!dayReady.value) return false
  if (!wantCheckIn.value && !wantCheckOut.value) return false
  if (wantCheckIn.value && !validTime(checkInTime.value)) return false
  if (wantCheckOut.value && !validTime(checkOutTime.value)) return false
  return true
})

// 随机模式随机化秒数避免固定 :00 偏移；正常模式固定整分
function buildDateTime(t: string) {
  const [h, m] = t.split(':')
  const second = timeMode.value === 'normal' ? '00' : pad(Math.floor(Math.random() * 60))
  return `${pickedDate.value} ${pad(Number(h))}:${pad(Number(m))}:${second}`
}

function handleSubmit() {
  if (!canSubmit.value) return
  const items: AddRecordParams[] = []
  if (wantCheckIn.value) {
    items.push({ user_id: selectedUser.value, check_time: buildDateTime(checkInTime.value) })
  }
  if (wantCheckOut.value) {
    items.push({ user_id: selectedUser.value, check_time: buildDateTime(checkOutTime.value) })
  }
  emit('submit', items)
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="show" class="fixed inset-0 z-50 flex items-center justify-center">
        <div class="absolute inset-0 bg-off-black/40" @click="emit('close')" />
        <div class="relative z-10 w-full max-w-md rounded-lg border border-oat bg-white p-8 shadow-sm">
          <div class="mb-6">
            <h3 class="text-lg font-medium tracking-[-0.48px] text-off-black">
              添加打卡记录
            </h3>
            <p class="mt-1 text-xs text-muted">
              选择日期后自动检测缺失的上下班打卡
            </p>
          </div>

          <!-- 用户选择 -->
          <div class="mb-5">
            <label class="mb-2 block font-mono text-xs uppercase tracking-wider text-muted">用户</label>
            <CustomSelect v-model="selectedUser" :options="userOptions" class="w-full [&>button]:w-full [&>button]:justify-between" />
          </div>

          <!-- 日期选择 -->
          <div class="mb-5">
            <label class="mb-2 block font-mono text-xs uppercase tracking-wider text-muted">日期</label>
            <CalendarPicker v-model="pickedDate" inline />
          </div>

          <!-- 当日已有记录 -->
          <div class="mb-5">
            <label class="mb-2 block font-mono text-xs uppercase tracking-wider text-muted">当日记录</label>
            <p v-if="dayLoading" class="text-xs text-muted">
              查询中……
            </p>
            <p v-else-if="dayError" class="text-xs text-report-red">
              {{ dayError }}
            </p>
            <p v-else-if="dayRecords.length === 0" class="text-xs text-muted">
              暂无打卡记录
            </p>
            <div v-else class="flex flex-wrap gap-2">
              <span
                v-for="r in dayRecords"
                :key="r.record_id"
                class="inline-flex items-center gap-1.5 rounded border px-2 py-0.5 font-mono text-[10px] font-medium uppercase tracking-wider"
                :class="r.label === 'checkin'
                  ? 'bg-report-green/10 text-report-green border-report-green/20'
                  : 'bg-fin/10 text-fin border-fin/20'"
              >
                {{ r.label === 'checkin' ? '上班' : '下班' }}
                <span class="normal-case">{{ r.check_time.slice(11, 16) }}</span>
              </span>
            </div>
          </div>

          <!-- 补卡时间输入：按缺失情况动态显示 -->
          <div v-if="dayReady" class="mb-8">
            <template v-if="needCheckIn || needCheckOut">
              <div class="mb-2 flex items-center justify-between">
                <label class="font-mono text-xs uppercase tracking-wider text-muted">补卡时间</label>
                <div class="flex overflow-hidden rounded border border-oat">
                  <button
                    v-for="m in timeModes"
                    :key="m.value"
                    class="px-2.5 py-1 text-xs transition-colors"
                    :class="timeMode === m.value ? 'bg-off-black text-white' : 'bg-white text-muted hover:text-off-black'"
                    @click="setTimeMode(m.value)"
                  >
                    {{ m.label }}
                  </button>
                </div>
              </div>
              <div class="space-y-3">
                <div v-if="needCheckIn" class="flex items-center gap-3">
                  <label class="flex w-16 shrink-0 cursor-pointer select-none items-center gap-2 text-sm text-off-black">
                    <input
                      v-model="addCheckIn"
                      type="checkbox"
                      class="h-3.5 w-3.5 accent-fin"
                    >
                    上班
                  </label>
                  <TimeField :model-value="checkInTime" :disabled="!addCheckIn" @update:model-value="onCheckInInput" />
                </div>
                <div v-if="needCheckOut" class="flex items-center gap-3">
                  <label class="flex w-16 shrink-0 cursor-pointer select-none items-center gap-2 text-sm text-off-black">
                    <input
                      v-model="addCheckOut"
                      type="checkbox"
                      class="h-3.5 w-3.5 accent-fin"
                    >
                    下班
                  </label>
                  <TimeField :model-value="checkOutTime" :disabled="!addCheckOut" @update:model-value="onCheckOutInput" />
                </div>
              </div>
            </template>
            <p v-else-if="dayComplete" class="rounded border border-report-green/20 bg-report-green/5 px-3 py-2 text-xs text-report-green">
              该日期上下班打卡已齐全，无需补卡
            </p>
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
