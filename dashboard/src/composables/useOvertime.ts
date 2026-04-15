import { ref } from 'vue'
import type { OvertimeRecord, OvertimeSchema } from '@/types'

const API_BASE = '/api'

export function useOvertime() {
  const schema = ref<OvertimeSchema | null>(null)
  const records = ref<OvertimeRecord[]>([])
  const loading = ref(false)
  const saving = ref(false) // 加班记录写入中（创建或更新）
  const error = ref<string | null>(null)
  const authorized = ref<boolean | null>(null) // null=未知, true=已授权, false=未授权

  async function fetchSchema() {
    try {
      const res = await fetch(`${API_BASE}/overtime/schema`)
      if (!res.ok) {
        throw new Error(`获取加班表结构失败: ${res.status}`)
      }
      schema.value = await res.json() as OvertimeSchema
    }
    catch (e) {
      error.value = e instanceof Error ? e.message : '获取加班表结构失败'
    }
  }

  async function queryRecords(filter?: string) {
    loading.value = true
    error.value = null
    try {
      const query = new URLSearchParams()
      if (filter) query.set('filter', filter)

      const res = await fetch(`${API_BASE}/overtime/records?${query.toString()}`)
      if (!res.ok) {
        const data = await res.json().catch(() => ({})) as { error?: string }
        const msg = data.error || ''
        // 飞书权限不足
        if (msg.includes('91403') || msg.includes('Forbidden')) {
          authorized.value = false
          records.value = []
          return
        }
        throw new Error(data.error || `查询加班记录失败: ${res.status}`)
      }
      const data = await res.json() as { records: OvertimeRecord[] }
      records.value = data.records ?? []
      authorized.value = true
    }
    catch (e) {
      error.value = e instanceof Error ? e.message : '查询加班记录失败'
      records.value = []
    }
    finally {
      loading.value = false
    }
  }

  async function updateRecord(recordId: string, fields: Record<string, unknown>): Promise<boolean> {
    saving.value = true
    error.value = null
    try {
      const res = await fetch(`${API_BASE}/overtime/records/${recordId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ fields }),
      })
      if (!res.ok) {
        const data = await res.json().catch(() => ({})) as { error?: string }
        const msg = data.error || ''
        // 飞书写权限不足：标记为未授权，让授权按钮重新出现
        if (msg.includes('91403') || msg.includes('Forbidden')) {
          authorized.value = false
          throw new Error('加班表格写权限不足，请点击右上角"授权加班表格"重新授权')
        }
        throw new Error(data.error || `更新失败: ${res.status}`)
      }
      return true
    }
    catch (e) {
      error.value = e instanceof Error ? e.message : '更新失败'
      return false
    }
    finally {
      saving.value = false
    }
  }

  async function createRecord(fields: Record<string, unknown>): Promise<OvertimeRecord | null> {
    saving.value = true
    error.value = null
    try {
      const res = await fetch(`${API_BASE}/overtime/records`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ fields }),
      })
      if (!res.ok) {
        const data = await res.json().catch(() => ({})) as { error?: string }
        const msg = data.error || ''
        // 飞书写权限不足：标记为未授权，让授权按钮重新出现
        if (msg.includes('91403') || msg.includes('Forbidden')) {
          authorized.value = false
          throw new Error('加班表格写权限不足，请点击右上角"授权加班表格"重新授权')
        }
        throw new Error(data.error || `创建失败: ${res.status}`)
      }
      const data = await res.json() as { record: OvertimeRecord }
      return data.record
    }
    catch (e) {
      error.value = e instanceof Error ? e.message : '创建失败'
      return null
    }
    finally {
      saving.value = false
    }
  }

  return { schema, records, loading, saving, error, authorized, fetchSchema, queryRecords, updateRecord, createRecord }
}
