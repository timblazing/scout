export interface Device {
  id: string
  ip: string
  mac: string
  hostname: string
  vendor: string
  type: string
  label: string
  icon: string
  web_ui: boolean
  risks: string[]
}

export interface DeviceList {
  devices: Device[]
  warning?: string
  // scanning is true until the first scan has finished.
  scanning?: boolean
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

export const getDevices = () => request<DeviceList>("/api/devices")

// The API rejects mutating requests that are not JSON, as CSRF protection.
export const updateDevice = (id: string, fields: { label?: string; icon?: string }) =>
  request<unknown>(`/api/devices/${encodeURIComponent(id)}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(fields),
  })
