import catalog from "@/lib/dashboard-icons.json"
import type { Device } from "@/lib/api"

type DashboardIcon = { slug: string; format: string; aliases: string[]; light: boolean }
export const dashboardIcons = catalog as DashboardIcon[]
export const deviceIconNames = [
  "monitor", "laptop", "smartphone", "tablet", "router", "server", "hard-drive", "printer",
  "tv", "circle-play", "gamepad-2", "cpu", "camera", "speaker", "wifi", "network",
  "radio", "plug", "lightbulb", "thermometer", "watch", "car", "bot", "shield", "house", "box",
] as const
export type DeviceIconName = typeof deviceIconNames[number]

const bySlug = new Map(dashboardIcons.map((icon) => [icon.slug, icon]))
const normalized = (value: string) => value.normalize("NFKD").toLowerCase().replace(/[^a-z0-9]+/g, " ").trim()
const byName = new Map<string, DashboardIcon>()
for (const icon of dashboardIcons) {
  for (const name of [icon.slug, ...icon.aliases]) {
    const key = normalized(name)
    if (key && !byName.has(key)) byName.set(key, icon)
    const compact = key.replaceAll(" ", "")
    if (compact.length > 5 && !byName.has(compact)) byName.set(compact, icon)
  }
}

const companySuffix = /(?:\s+(?:systems?|technolog(?:y|ies)|electronics|inc|incorporated|corp|corporation|company|co|ltd|limited|llc|gmbh|foundation|group))+$/
const regions = /^(?:beijing|shenzhen|shanghai|hangzhou|guangdong|hong kong)\s+/
function matchName(raw: string): DashboardIcon | undefined {
  let name = normalized(raw.replace(/\([^)]*\)/g, ""))
  for (let i = 0; i < 3 && name; i++) {
    const match = byName.get(name) ?? byName.get(name.replaceAll(" ", ""))
    if (match) return match
    const shorter = name.replace(companySuffix, "").replace(regions, "")
    if (shorter === name) break
    name = shorter
  }
}

export function automaticDashboardIcon(device: Pick<Device, "label" | "hostname" | "vendor">): DashboardIcon | undefined {
  for (const value of [device.label, device.vendor !== "Unknown" ? device.vendor : "", device.hostname]) {
    if (value) {
      const icon = matchName(value)
      if (icon) return icon
    }
  }
}

export function dashboardIconFor(value?: string): DashboardIcon | undefined {
  if (!value?.startsWith("dashboard:")) return undefined
  return bySlug.get(value.slice(10))
}

export function dashboardIconUrl(icon: DashboardIcon): string {
  const filename = icon.light ? `${icon.slug}-light` : icon.slug
  return `https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons/${icon.format}/${filename}.${icon.format}`
}
