import type { Device } from "@/lib/api"

// Randomized ("private") MAC addresses set the locally administered bit, which
// makes the second hex digit 2, 6, A or E. Phones and laptops use them by
// default, and no vendor lookup can ever identify them.
export function isPrivateMac(mac: string): boolean {
  return /^[0-9a-f][26ae][:-]/i.test(mac)
}

const corpSuffix =
  /[,\s]+(inc\.?|llc|ltd\.?|limited|co\.?|corp\.?|corporation|company|gmbh|s\.?a\.?|b\.?v\.?|systems|technologies|technology|tech\.?|electronics|\(trading\)|incorporated)$/i

// shortVendor trims corporate suffixes: "TP-Link Systems" -> "TP-Link".
export function shortVendor(vendor: string): string {
  let v = vendor.trim()
  for (let prev = ""; prev !== v; ) {
    prev = v
    v = v.replace(corpSuffix, "").trim()
  }
  return v
}

export function vendorLabel(d: Pick<Device, "vendor" | "mac">): { text: string; muted: boolean } {
  const known = d.vendor && d.vendor !== "Unknown"
  if (known) return { text: d.vendor, muted: false }
  if (d.mac && isPrivateMac(d.mac)) return { text: "Private address", muted: true }
  return { text: "—", muted: true }
}

// Hostnames that say nothing about the device: bare OS names that embedded
// devices report (a Blink camera calls itself "Linux").
const generic = new Set(["linux", "android", "localhost", "unknown", "none", "espressif", "(none)"])

// Hostnames that are identifiers rather than names: UUIDs (Chromecast and
// other mDNS devices), router placeholders like "unknown98038eba160b", long
// runs of hex, and generic OS names.
function isUnreadable(name: string): boolean {
  return (
    generic.has(name.toLowerCase()) ||
    /^[0-9a-f]{8}-[0-9a-f]{4}-/i.test(name) ||
    /^unknown[0-9a-f]*$/i.test(name) ||
    /^[0-9a-f]{12,}$/i.test(name)
  )
}

const ipLike = /^[\d.]+$|:/

export interface DeviceName {
  text: string
  // derived is true when the name was built from vendor and type rather than
  // reported by the device, so it can be styled as secondary.
  derived: boolean
  // full is the untouched hostname, for a tooltip, when it differs from text.
  full?: string
}

export function deviceName(d: Pick<Device, "label" | "hostname" | "vendor" | "mac" | "type">): DeviceName {
  if (d.label) return { text: d.label, derived: false }

  const host = d.hostname.trim()
  if (host && !isUnreadable(host)) {
    // Drop the router's search domain: "iPhone.attlocal.net" -> "iPhone".
    const short = ipLike.test(host) ? host : host.split(".")[0]
    if (short && !isUnreadable(short)) {
      return { text: short, derived: false, full: short !== host ? host : undefined }
    }
  }

  const vendor = d.vendor && d.vendor !== "Unknown" ? shortVendor(d.vendor) : ""
  const text = [vendor, d.type].filter(Boolean).join(" ") || (isPrivateMac(d.mac) ? "Private device" : "Unknown device")
  return { text, derived: true, full: host || undefined }
}
