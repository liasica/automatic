export interface AttendanceRecord {
  record_id: string
  user_id: string
  check_time: string
  timestamp: number
  label: 'checkin' | 'checkout'
  day_type: 'workday' | 'weekend' | 'holiday'
  weekday: string
}

export interface UserGroup {
  user_id: string
  records: AttendanceRecord[]
}

export interface UserInfo {
  id: string
  check_in_latest: string
  check_out_earliest: string
}

export interface QueryParams {
  user_id?: string
  from: string
  to?: string
}

export interface AddRecordParams {
  user_id: string
  check_time: string
}

export type DayFilter = 'all' | 'workday' | 'non_workday' | 'weekend' | 'holiday'
