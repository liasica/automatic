<script setup lang="ts">
import { computed } from 'vue'
import type { UserGroup, AttendanceRecord, UserInfo, DayFilter } from '@/types'

const props = defineProps<{
  groups: UserGroup[]
  users: UserInfo[]
  loading: boolean
  dayFilter: DayFilter
}>()

const emit = defineEmits<{
  delete: [recordIds: string[]]
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

const processedGroups = computed<ProcessedGroup[]>(() => {
  return props.groups.map((group) => {
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
      .flatMap((dg) => {
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
</script>

<template>
  <!-- 加载中 -->
  <div v-if="loading" class="rounded-lg border border-oat bg-white px-5 py-16 text-center">
    <div class="flex items-center justify-center gap-3 text-muted">
      <svg class="h-4 w-4 animate-spin text-fin" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3" />
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
          </div>

          <!-- 缺卡汇总 -->
          <div v-if="pg.anomalies.length > 0" class="mt-2.5 flex flex-wrap gap-1.5 pl-[38px]">
            <span
              v-for="(a, i) in pg.anomalies"
              :key="`${a.date}-${i}`"
              class="inline-flex items-center gap-1 rounded border border-report-red/20 bg-report-red/5 px-2 py-0.5 font-mono text-[10px] tracking-wide text-report-red"
            >
              <svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
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
