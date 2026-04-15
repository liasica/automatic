import { ref } from 'vue'
import type { AddRecordParams, QueryParams, UserGroup, UserInfo } from '@/types'

const API_BASE = '/api'

export function useAttendance() {
  const groups = ref<UserGroup[]>([])
  const users = ref<UserInfo[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchUsers() {
    try {
      const res = await fetch(`${API_BASE}/users`)
      if (!res.ok) {
        throw new Error(`获取用户列表失败: ${res.status}`)
      }
      const data = await res.json() as { users: UserInfo[] }
      users.value = data.users ?? []
    }
    catch (e) {
      error.value = e instanceof Error ? e.message : '获取用户列表失败'
    }
  }

  async function queryRecords(params: QueryParams) {
    loading.value = true
    error.value = null
    try {
      const query = new URLSearchParams({ from: params.from })
      if (params.to) {
        query.set('to', params.to)
      }
      if (params.user_id) {
        query.set('user_id', params.user_id)
      }
      const res = await fetch(`${API_BASE}/attendance/records?${query.toString()}`)
      if (!res.ok) {
        throw new Error(`查询失败: ${res.status}`)
      }
      const data = await res.json() as { groups: UserGroup[] }
      groups.value = data.groups ?? []
    }
    catch (e) {
      error.value = e instanceof Error ? e.message : '查询失败'
      groups.value = []
    }
    finally {
      loading.value = false
    }
  }

  async function addRecord(params: AddRecordParams): Promise<boolean> {
    loading.value = true
    error.value = null
    try {
      const res = await fetch(`${API_BASE}/attendance/records`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(params),
      })
      if (!res.ok) {
        throw new Error(`添加失败: ${res.status}`)
      }
      return true
    }
    catch (e) {
      error.value = e instanceof Error ? e.message : '添加失败'
      return false
    }
    finally {
      loading.value = false
    }
  }

  async function deleteRecords(recordIds: string[]): Promise<{ successIds: string[], failIds: string[] } | null> {
    loading.value = true
    error.value = null
    try {
      const res = await fetch(`${API_BASE}/attendance/records`, {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ record_ids: recordIds }),
      })
      if (!res.ok) {
        throw new Error(`删除失败: ${res.status}`)
      }
      const data = await res.json() as { success_ids: string[], fail_ids: string[] }
      return { successIds: data.success_ids ?? [], failIds: data.fail_ids ?? [] }
    }
    catch (e) {
      error.value = e instanceof Error ? e.message : '删除失败'
      return null
    }
    finally {
      loading.value = false
    }
  }

  return { groups, users, loading, error, fetchUsers, queryRecords, addRecord, deleteRecords }
}
