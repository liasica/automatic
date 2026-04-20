<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { UserGroup, AttendanceRecord, UserInfo, DayFilter, OvertimeRecord, OvertimeSchema } from '@/types'

const props = defineProps<{
  groups: UserGroup[]
  users: UserInfo[]
  loading: boolean
  dayFilter: DayFilter
  overtimeRecords: OvertimeRecord[]
  overtimeSchema: OvertimeSchema | null
  overtimeSaving: boolean
}>()

const emit = defineEmits<{
  delete: [recordIds: string[]]
  overtimeUpdate: [recordId: string, fields: Record<string, unknown>]
  overtimeCreate: [fields: Record<string, unknown>]
}>()

interface DateGroup {
  date: string
  weekday: string
  dayType: string
  records: AttendanceRecord[]
  hasCheckin: boolean
  hasCheckout: boolean
  showMissingCheckin: boolean
  showMissingCheckout: boolean
}

interface Anomaly {
  date: string
  weekday: string
  label: string
}

interface ProcessedGroup {
  user_id: string
  totalRecords: number
  dateGroups: DateGroup[]
  anomalies: Anomaly[]
}

const todayStr = new Date().toISOString().slice(0, 10)

// 判断指定日期的指定时间点是否已过
function isPastTime(date: string, timeStr?: string): boolean {
  if (date !== todayStr)
    return true
  if (!timeStr)
    return true
  const parts = timeStr.split(':').map(Number)
  const h = parts[0] ?? 0
  const m = parts[1] ?? 0
  const now = new Date()
  return now.getHours() > h || (now.getHours() === h && now.getMinutes() >= m)
}

function matchFilter(dayType: string): boolean {
  switch (props.dayFilter) {
    case 'workday': return dayType === 'workday'
    case 'non_workday': return dayType === 'weekend' || dayType === 'holiday'
    case 'weekend': return dayType === 'weekend'
    case 'holiday': return dayType === 'holiday'
    default: return true
  }
}

// 从 schema 中找到日期字段名（type 5 = DateTime）
const dateFieldName = computed(() => {
  if (!props.overtimeSchema) return null
  const field = props.overtimeSchema.fields.find(f => f.type === 5)
  return field?.field_name ?? null
})

// 从 schema 中获取可编辑的文本字段（type 1 = Text）
const editableFields = computed(() => {
  if (!props.overtimeSchema) return []
  return props.overtimeSchema.fields.filter(f => f.type === 1)
})

// 按日期索引加班记录
const overtimeByDate = computed(() => {
  const map = new Map<string, OvertimeRecord[]>()
  const dField = dateFieldName.value
  if (!dField) return map

  for (const record of props.overtimeRecords) {
    const raw = record.fields[dField]
    const date = extractDate(raw)
    if (!date) continue
    if (!map.has(date)) map.set(date, [])
    map.get(date)!.push(record)
  }
  return map
})

// 从飞书日期字段值中提取本地 YYYY-MM-DD
function toLocalDateStr(d: Date): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

function extractDate(value: unknown): string | null {
  if (!value) return null
  // 毫秒时间戳
  if (typeof value === 'number') {
    return toLocalDateStr(new Date(value))
  }
  // 字符串时间戳或日期字符串
  if (typeof value === 'string') {
    if (/^\d{10,13}$/.test(value)) {
      const ts = Number(value)
      return toLocalDateStr(new Date(ts > 1e12 ? ts : ts * 1000))
    }
    const match = value.match(/^\d{4}-\d{2}-\d{2}/)
    if (match) return match[0]
  }
  return null
}

// 获取加班记录中某文本字段的值
function getFieldText(record: OvertimeRecord, fieldName: string): string {
  const val = record.fields[fieldName]
  if (val == null) return ''
  if (typeof val === 'string') return val
  // 飞书文本字段有时返回数组格式 [{type: "text", text: "..."}]
  if (Array.isArray(val)) {
    return val.map((item: unknown) => {
      if (typeof item === 'string') return item
      if (item && typeof item === 'object' && 'text' in item) return (item as { text: string }).text
      return ''
    }).join('')
  }
  return String(val)
}

// 内联编辑状态
const editingKey = ref<string | null>(null) // "recordId:fieldName"
const editingValue = ref('')

function startEdit(recordId: string, fieldName: string, currentValue: string) {
  editingKey.value = `${recordId}:${fieldName}`
  editingValue.value = currentValue
}

function cancelEdit() {
  if (props.overtimeSaving) return
  editingKey.value = null
  editingValue.value = ''
}

function saveEdit(recordId: string, fieldName: string) {
  // 不立即清空编辑态，保存期间保留 UI；请求完成后 watch overtimeSaving 会清理
  emit('overtimeUpdate', recordId, { [fieldName]: editingValue.value })
}

function isEditing(recordId: string, fieldName: string): boolean {
  return editingKey.value === `${recordId}:${fieldName}`
}

// 新增加班记录状态
const creatingDate = ref<string | null>(null) // 正在添加加班的日期
const createFields = ref<Record<string, string>>({})

function startCreate(date: string) {
  creatingDate.value = date
  const fields: Record<string, string> = {}
  for (const f of editableFields.value) {
    fields[f.field_name] = ''
  }
  createFields.value = fields
}

function cancelCreate() {
  if (props.overtimeSaving) return
  creatingDate.value = null
  createFields.value = {}
}

function submitCreate(date: string) {
  const dField = dateFieldName.value
  if (!dField) return

  // 日期转为本地零点的毫秒时间戳
  const ts = new Date(`${date}T00:00:00`).getTime()
  const fields: Record<string, unknown> = { [dField]: ts }
  for (const [k, v] of Object.entries(createFields.value)) {
    if (v) fields[k] = v
  }

  // 不立即清空表单，保存期间保留 UI；请求完成后 watch overtimeSaving 会清理
  emit('overtimeCreate', fields)
}

// 保存完成（saving: true → false）后清理编辑/创建态
watch(() => props.overtimeSaving, (newVal, oldVal) => {
  if (oldVal === true && newVal === false) {
    editingKey.value = null
    editingValue.value = ''
    creatingDate.value = null
    createFields.value = {}
  }
})

const processedGroups = computed<ProcessedGroup[]>(() => {
  return props.groups.map(group => {
    const dateMap = new Map<string, AttendanceRecord[]>()
    for (const r of group.records) {
      const date = r.check_time.slice(0, 10)
      if (!dateMap.has(date))
        dateMap.set(date, [])
      dateMap.get(date)!.push(r)
    }

    const user = props.users.find(u => u.id === group.user_id)

    const allDateGroups: DateGroup[] = Array.from(dateMap.entries()).map(([date, records]) => {
      const hasCheckin = records.some(r => r.label === 'checkin')
      const hasCheckout = records.some(r => r.label === 'checkout')
      return {
        date,
        weekday: records[0]?.weekday ?? '',
        dayType: records[0]?.day_type ?? 'workday',
        records,
        hasCheckin,
        hasCheckout,
        showMissingCheckin: !hasCheckin && isPastTime(date, user?.check_in_latest),
        showMissingCheckout: !hasCheckout && isPastTime(date, user?.check_out_earliest),
      }
    })

    const dateGroups = allDateGroups.filter(dg => matchFilter(dg.dayType))

    const anomalies: Anomaly[] = dateGroups
      .filter(dg => dg.showMissingCheckin || dg.showMissingCheckout)
      .flatMap(dg => {
        const items: Anomaly[] = []
        if (dg.showMissingCheckin)
          items.push({ date: dg.date, weekday: dg.weekday, label: '缺上班' })
        if (dg.showMissingCheckout)
          items.push({ date: dg.date, weekday: dg.weekday, label: '缺下班' })
        return items
      })

    return {
      user_id: group.user_id,
      totalRecords: dateGroups.reduce((sum, dg) => sum + dg.records.length, 0),
      dateGroups,
      anomalies,
    }
  }).filter(pg => pg.dateGroups.length > 0)
})

const summaryUsers = computed(() => processedGroups.value.length)
const summaryRecords = computed(() => processedGroups.value.reduce((sum, pg) => sum + pg.totalRecords, 0))

function timeOnly(checkTime: string): string {
  return checkTime.split(' ')[1] || checkTime
}

function shortDate(date: string): string {
  const [, m, d] = date.split('-')
  return `${Number(m)}/${Number(d)}`
}

// 计算单日上班时长（最早上班 → 最晚下班）
interface DurationInfo {
  hours: number // 精确到 0.1 小时
  days: number // 0 / 0.5 / 1
}

function calcDuration(dg: DateGroup): DurationInfo | null {
  const checkins = dg.records.filter(r => r.label === 'checkin')
  const checkouts = dg.records.filter(r => r.label === 'checkout')
  if (checkins.length === 0 || checkouts.length === 0) return null

  const earliest = Math.min(...checkins.map(r => r.timestamp))
  const latest = Math.max(...checkouts.map(r => r.timestamp))
  const hours = Math.round((latest - earliest) / 360) / 10 // 保留一位小数

  let days: number
  if (hours < 4) days = 0
  else if (hours < 8) days = 0.5
  else days = 1

  return { hours, days }
}

function formatDuration(d: DurationInfo): string {
  return `${d.days}天 · ${d.hours}h`
}

// 汇总某用户全部日期的总时长
function totalDuration(dateGroups: DateGroup[]): DurationInfo | null {
  let totalSeconds = 0
  let count = 0

  for (const dg of dateGroups) {
    const checkins = dg.records.filter(r => r.label === 'checkin')
    const checkouts = dg.records.filter(r => r.label === 'checkout')
    if (checkins.length === 0 || checkouts.length === 0) continue
    totalSeconds += Math.max(...checkouts.map(r => r.timestamp)) - Math.min(...checkins.map(r => r.timestamp))
    count++
  }

  if (count === 0) return null
  const hours = Math.round(totalSeconds / 360) / 10

  let days = 0
  for (const dg of dateGroups) {
    const d = calcDuration(dg)
    if (d) days += d.days
  }

  return { hours, days }
}
</script>

<template>
  <!-- 加载中 -->
  <div v-if="loading" class="rounded-lg border border-oat bg-white px-5 py-16 text-center">
    <div class="flex items-center justify-center gap-3 text-muted">
      <svg
        class="h-4 w-4 animate-spin text-fin"
        xmlns="http://www.w3.org/2000/svg"
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
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
      </svg>
      <span class="text-sm">加载中...</span>
    </div>
  </div>

  <!-- 空状态 -->
  <div v-else-if="summaryRecords === 0" class="rounded-lg border border-oat bg-white px-5 py-16 text-center">
    <div class="text-3xl">
      📋
    </div>
    <p class="mt-2 text-sm text-muted">
      暂无打卡记录
    </p>
  </div>

  <!-- 按用户分组 -->
  <template v-else>
    <div class="space-y-10">
      <div v-for="pg in processedGroups" :key="pg.user_id">
        <!-- 用户标题 -->
        <div class="mb-4">
          <div class="flex items-center gap-2.5">
            <div class="flex h-7 w-7 items-center justify-center rounded-full bg-fin text-xs font-medium text-white">
              {{ pg.user_id.charAt(0).toUpperCase() }}
            </div>
            <span class="text-sm font-medium tracking-[-0.2px] text-off-black">{{ pg.user_id }}</span>
            <span class="rounded bg-cream px-2 py-0.5 font-mono text-xs text-muted">{{ pg.totalRecords }}条</span>
            <span
              v-if="totalDuration(pg.dateGroups)"
              class="rounded bg-fin/10 px-2 py-0.5 font-mono text-xs text-fin"
            >
              合计 {{ totalDuration(pg.dateGroups)!.days }}天 · {{ totalDuration(pg.dateGroups)!.hours }}h
            </span>
          </div>

          <!-- 缺卡汇总 -->
          <div v-if="pg.anomalies.length > 0" class="mt-2.5 flex flex-wrap gap-1.5 pl-[38px]">
            <span
              v-for="(a, i) in pg.anomalies"
              :key="`${a.date}-${i}`"
              class="inline-flex items-center gap-1 rounded border border-report-red/20 bg-report-red/5 px-2 py-0.5 font-mono text-[10px] tracking-wide text-report-red"
            >
              <svg
                class="h-3 w-3"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="2"
              >
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
              </svg>
              {{ shortDate(a.date) }}({{ a.weekday }}) {{ a.label }}
            </span>
          </div>
        </div>

        <!-- 日期分组 -->
        <div class="space-y-3">
          <div v-for="dg in pg.dateGroups" :key="dg.date" class="overflow-hidden rounded-lg border border-oat bg-white">
            <!-- 日期标题栏 -->
            <div class="flex items-center justify-between border-b border-oat bg-cream/60 px-4 py-2.5">
              <div class="flex items-center gap-2">
                <span class="text-sm font-medium text-off-black">{{ dg.date }}</span>
                <span class="text-xs text-muted">{{ dg.weekday }}</span>
                <span
                  v-if="dg.dayType === 'holiday'"
                  class="rounded border border-report-red/20 bg-report-red/10 px-1.5 py-px font-mono text-[10px] font-medium text-report-red"
                >
                  节
                </span>
                <span
                  v-else-if="dg.dayType === 'weekend'"
                  class="rounded border border-sand bg-sand/20 px-1.5 py-px font-mono text-[10px] font-medium text-mid"
                >
                  休
                </span>
                <span
                  v-if="dg.showMissingCheckin"
                  class="rounded border border-report-red/20 bg-report-red/5 px-1.5 py-px font-mono text-[10px] text-report-red"
                >
                  缺上班
                </span>
                <span
                  v-if="dg.showMissingCheckout"
                  class="rounded border border-report-red/20 bg-report-red/5 px-1.5 py-px font-mono text-[10px] text-report-red"
                >
                  缺下班
                </span>
              </div>
              <div class="flex items-center gap-2">
                <span
                  v-if="calcDuration(dg)"
                  class="font-mono text-xs text-fin"
                >
                  {{ formatDuration(calcDuration(dg)!) }}
                </span>
                <span class="font-mono text-xs text-muted">{{ dg.records.length }}条</span>
                <button
                  v-if="dg.records.length > 1"
                  class="rounded border border-transparent px-2 py-0.5 font-mono text-[10px] text-report-red transition-all hover:border-report-red/20 hover:bg-report-red/5"
                  @click="emit('delete', dg.records.map(r => r.record_id))"
                >
                  删除当日
                </button>
              </div>
            </div>

            <!-- 记录行 -->
            <div>
              <div
                v-for="(record, i) in dg.records"
                :key="record.record_id"
                class="flex items-center gap-3 border-b border-oat/60 px-4 py-2.5 transition-colors last:border-b-0 hover:bg-cream/40"
                :class="{ 'last:border-b': overtimeByDate.has(dg.date) }"
              >
                <!-- 序号 -->
                <span class="w-6 shrink-0 font-mono text-xs text-muted">
                  {{ String(i + 1).padStart(2, '0') }}
                </span>

                <!-- 标签 -->
                <span
                  class="inline-block shrink-0 rounded px-2 py-0.5 font-mono text-[10px] font-medium uppercase tracking-wider"
                  :class="record.label === 'checkin'
                    ? 'bg-report-green/10 text-report-green border border-report-green/20'
                    : 'bg-fin/10 text-fin border border-fin/20'"
                >
                  {{ record.label === 'checkin' ? '上班' : '下班' }}
                </span>

                <!-- 时间 -->
                <span class="w-20 shrink-0 font-mono text-sm text-off-black">
                  {{ timeOnly(record.check_time) }}
                </span>

                <!-- 记录 ID -->
                <span class="min-w-0 flex-1 truncate font-mono text-xs text-muted">
                  {{ record.record_id }}
                </span>

                <!-- 删除 -->
                <button
                  class="shrink-0 rounded border border-transparent px-2.5 py-1 font-mono text-xs text-report-red transition-all hover:border-report-red/20 hover:bg-report-red/5"
                  @click="emit('delete', [record.record_id])"
                >
                  删除
                </button>
              </div>
            </div>

            <!-- 加班记录 -->
            <template v-if="overtimeByDate.has(dg.date)">
              <div
                v-for="otRecord in overtimeByDate.get(dg.date)"
                :key="otRecord.record_id"
                class="border-t border-dashed border-fin/30 bg-fin/5 px-4 py-2.5"
              >
                <div class="mb-1.5 flex items-center gap-2">
                  <span class="rounded border border-fin/20 bg-fin/10 px-1.5 py-px font-mono text-[10px] font-medium text-fin">
                    加班
                  </span>
                  <span class="font-mono text-[10px] text-muted">{{ otRecord.record_id }}</span>
                </div>
                <div class="space-y-1.5 pl-0.5">
                  <div
                    v-for="field in editableFields"
                    :key="field.field_id"
                    class="flex items-start gap-2 text-sm"
                  >
                    <span class="w-12 shrink-0 pt-px text-xs text-muted">{{ field.field_name }}</span>
                    <!-- 编辑模式 -->
                    <div
                      v-if="isEditing(otRecord.record_id, field.field_name)"
                      class="flex flex-1 items-center gap-1.5"
                    >
                      <input
                        v-model="editingValue"
                        class="flex-1 rounded border border-fin/30 bg-white px-2 py-0.5 text-xs text-off-black outline-none focus:border-fin disabled:opacity-60"
                        :disabled="overtimeSaving"
                        @keydown.enter="saveEdit(otRecord.record_id, field.field_name)"
                        @keydown.escape="cancelEdit"
                      >
                      <button
                        :class="[
                          'inline-flex items-center gap-1 rounded border border-fin/20 px-1.5 py-0.5 font-mono text-[10px] text-fin transition-colors hover:bg-fin/10',
                          'disabled:cursor-not-allowed disabled:opacity-60 disabled:hover:bg-transparent',
                        ]"
                        :disabled="overtimeSaving"
                        @click="saveEdit(otRecord.record_id, field.field_name)"
                      >
                        <svg
                          v-if="overtimeSaving"
                          class="h-2.5 w-2.5 animate-spin"
                          xmlns="http://www.w3.org/2000/svg"
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
                          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                        </svg>
                        {{ overtimeSaving ? '保存中' : '保存' }}
                      </button>
                      <button
                        class="rounded px-1.5 py-0.5 font-mono text-[10px] text-muted transition-colors hover:bg-cream disabled:cursor-not-allowed disabled:opacity-60 disabled:hover:bg-transparent"
                        :disabled="overtimeSaving"
                        @click="cancelEdit"
                      >
                        取消
                      </button>
                    </div>
                    <!-- 显示模式 -->
                    <span
                      v-else
                      class="min-w-0 flex-1 cursor-pointer truncate text-xs text-off-black transition-colors hover:text-fin"
                      :title="`点击编辑 ${field.field_name}`"
                      @click="startEdit(otRecord.record_id, field.field_name, getFieldText(otRecord, field.field_name))"
                    >
                      {{ getFieldText(otRecord, field.field_name) || '—' }}
                    </span>
                  </div>
                </div>
              </div>
            </template>

            <!-- 新增加班记录表单 -->
            <div
              v-if="creatingDate === dg.date"
              class="border-t border-dashed border-fin/30 bg-fin/5 px-4 py-2.5"
            >
              <div class="mb-1.5 flex items-center gap-2">
                <span class="rounded border border-fin/20 bg-fin/10 px-1.5 py-px font-mono text-[10px] font-medium text-fin">
                  新增加班
                </span>
              </div>
              <div class="space-y-1.5 pl-0.5">
                <div
                  v-for="field in editableFields"
                  :key="field.field_id"
                  class="flex items-center gap-2 text-sm"
                >
                  <span class="w-12 shrink-0 text-xs text-muted">{{ field.field_name }}</span>
                  <input
                    v-model="createFields[field.field_name]"
                    class="flex-1 rounded border border-fin/30 bg-white px-2 py-0.5 text-xs text-off-black outline-none focus:border-fin disabled:opacity-60"
                    :disabled="overtimeSaving"
                    :placeholder="field.field_name"
                  >
                </div>
                <div class="flex justify-end gap-1.5 pt-1">
                  <button
                    class="rounded px-2 py-0.5 font-mono text-[10px] text-muted transition-colors hover:bg-cream disabled:cursor-not-allowed disabled:opacity-60 disabled:hover:bg-transparent"
                    :disabled="overtimeSaving"
                    @click="cancelCreate"
                  >
                    取消
                  </button>
                  <button
                    :class="[
                      'inline-flex items-center gap-1 rounded border border-fin/20 bg-fin/10 px-2 py-0.5 font-mono text-[10px] text-fin transition-colors hover:bg-fin/20',
                      'disabled:cursor-not-allowed disabled:opacity-60 disabled:hover:bg-fin/10',
                    ]"
                    :disabled="overtimeSaving"
                    @click="submitCreate(dg.date)"
                  >
                    <svg
                      v-if="overtimeSaving"
                      class="h-2.5 w-2.5 animate-spin"
                      xmlns="http://www.w3.org/2000/svg"
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
                      <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                    </svg>
                    {{ overtimeSaving ? '保存中' : '保存' }}
                  </button>
                </div>
              </div>
            </div>

            <!-- 添加加班按钮 -->
            <div
              v-else-if="dateFieldName && !overtimeByDate.has(dg.date) && (dg.dayType === 'holiday' || dg.dayType === 'weekend')"
              class="border-t border-dashed border-oat/60 px-4 py-1.5"
            >
              <button
                class="font-mono text-[10px] text-muted transition-colors hover:text-fin"
                @click="startCreate(dg.date)"
              >
                + 添加加班记录
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 底部信息 -->
    <div class="mt-6 text-center font-mono text-xs uppercase tracking-wider text-muted">
      {{ summaryUsers }} 个用户 · 共 {{ summaryRecords }} 条记录
    </div>
  </template>
</template>
