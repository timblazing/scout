export type DeviceStatus = "online" | "seen" | "offline"

export interface Device {
  ip: string
  mac: string
  hostname: string
  vendor: string
  type?: string
  web_ui?: boolean
  risks?: string[]
  label: string
  notes: string
  group: string
  first_seen: string
  last_seen: string
  response_time?: number
  address_history?: { ip: string; changed_at: string }[]
  status: DeviceStatus
}

export interface Network {
  cidr: string
  interface: string
  friendly_name: string
  ip: string
  is_tailscale: boolean
  is_wireless: boolean
}

export interface Overview {
  stats: { total: number; online: number; offline: number }
  devices: Device[]
  networks: Network[]
  network_warning?: string
  flagged: number
  moved: number
  last_scan?: string
  continuous_scan: boolean
  scanning: boolean
}

interface Envelope<T> {
  success: boolean
  data?: T
  error?: string
}

async function request<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, init)
  const body = (await res.json().catch(() => null)) as Envelope<T> | null
  if (!res.ok || !body?.success) {
    throw new Error(body?.error ?? `Request failed (${res.status})`)
  }
  return body.data as T
}

export const getOverview = () => request<Overview>("/api/overview")

// The API rejects mutating requests that are not JSON, as CSRF protection.
export const startScan = () =>
  request<unknown>("/api/scan/start?network=all", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
  })
