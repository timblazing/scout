import { useSyncExternalStore } from "react"
import type { Device } from "@/lib/api"

export type DashboardIcon = { slug: string; format: string; aliases: string[]; light: boolean }
export const deviceIconNames = [
  "monitor", "laptop", "smartphone", "tablet", "router", "server", "hard-drive", "printer",
  "tv", "circle-play", "gamepad-2", "cpu", "camera", "speaker", "wifi", "network",
  "radio", "plug", "lightbulb", "thermometer", "watch", "car", "bot", "shield", "house", "box",
] as const
export type DeviceIconName = typeof deviceIconNames[number]

// The Dashboard Icons catalog is large, so it loads on its own after the first
// paint instead of living in the main bundle. Until it arrives, rows fall back
// to device-type icons.
export interface Catalog {
  icons: DashboardIcon[]
  bySlug: Map<string, DashboardIcon>
  byName: Map<string, DashboardIcon>
}

const normalized = (value: string) => value.normalize("NFKD").toLowerCase().replace(/[^a-z0-9]+/g, " ").trim()

function buildCatalog(icons: DashboardIcon[]): Catalog {
  const bySlug = new Map(icons.map((icon) => [icon.slug, icon]))
  const byName = new Map<string, DashboardIcon>()
  for (const icon of icons) {
    for (const name of [icon.slug, ...icon.aliases]) {
      const key = normalized(name)
      if (key && !byName.has(key)) byName.set(key, icon)
      const compact = key.replaceAll(" ", "")
      if (compact.length > 5 && !byName.has(compact)) byName.set(compact, icon)
    }
  }
  return { icons, bySlug, byName }
}

let loaded: Catalog | undefined
let pending: Promise<Catalog | undefined> | undefined
const listeners = new Set<() => void>()

export function loadCatalog(): Promise<Catalog | undefined> {
  pending ??= import("@/lib/dashboard-icons.json").then(
    (module) => {
      loaded = buildCatalog(module.default as DashboardIcon[])
      listeners.forEach((listener) => listener())
      return loaded
    },
    () => {
      pending = undefined // let the next caller retry
      return undefined
    }
  )
  return pending
}

function subscribe(listener: () => void) {
  listeners.add(listener)
  return () => void listeners.delete(listener)
}

// useCatalog returns the catalog once it has loaded, and undefined before.
export function useCatalog(): Catalog | undefined {
  return useSyncExternalStore(subscribe, () => loaded)
}

const companySuffix = /(?:\s+(?:systems?|technolog(?:y|ies)|electronics|inc|incorporated|corp|corporation|company|co|ltd|limited|llc|gmbh|foundation|group))+$/
const regions = /^(?:beijing|shenzhen|shanghai|hangzhou|guangdong|hong kong)\s+/
function matchName(catalog: Catalog, raw: string): DashboardIcon | undefined {
  let name = normalized(raw.replace(/\([^)]*\)/g, ""))
  for (let i = 0; i < 3 && name; i++) {
    const match = catalog.byName.get(name) ?? catalog.byName.get(name.replaceAll(" ", ""))
    if (match) return match
    const shorter = name.replace(companySuffix, "").replace(regions, "")
    if (shorter === name) break
    name = shorter
  }
}

export function automaticDashboardIcon(catalog: Catalog, device: Pick<Device, "label" | "hostname" | "vendor">): DashboardIcon | undefined {
  for (const value of [device.label, device.vendor !== "Unknown" ? device.vendor : "", device.hostname]) {
    if (value) {
      const icon = matchName(catalog, value)
      if (icon) return icon
    }
  }
}

export function dashboardIconFor(catalog: Catalog, value?: string): DashboardIcon | undefined {
  if (!value?.startsWith("dashboard:")) return undefined
  return catalog.bySlug.get(value.slice(10))
}

export function dashboardIconUrl(icon: DashboardIcon): string {
  const filename = icon.light ? `${icon.slug}-light` : icon.slug
  return `https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons/${icon.format}/${filename}.${icon.format}`
}
