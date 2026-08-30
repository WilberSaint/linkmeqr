import { apiClient } from './client'

export interface DailyCount {
  date: string
  count: number
}

export interface BreakdownRow {
  label: string
  count: number
}

export interface BlockClickRow {
  block_id: string
  title: string
  block_type: string
  count: number
}

export interface StatsSummary {
  total_views: number
  total_clicks: number
  views_7d: number
  views_30d: number
  timeseries: DailyCount[]
  devices: BreakdownRow[]
  os: BreakdownRow[]
  browsers: BreakdownRow[]
  block_clicks: BlockClickRow[]
}

export function myStatsSummary(range: '7d' | '30d' = '30d') {
  return apiClient.get<StatsSummary>('/me/stats/summary', { params: { range } }).then((r) => r.data)
}
