export function timeAgo(iso: string | undefined, now: number): string {
  if (!iso) return "never"
  const t = new Date(iso).getTime()
  if (!t || t < 0) return "never"

  const s = Math.max(0, Math.round((now - t) / 1000))
  if (s < 60) return "just now"
  const m = Math.floor(s / 60)
  if (m < 60) return `${m}m ago`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h ago`
  const d = Math.floor(h / 24)
  if (d < 7) return `${d}d ago`
  return new Date(t).toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" })
}

export function ipKey(ip: string): number {
  const parts = ip.split(".")
  if (parts.length !== 4) return Number.MAX_SAFE_INTEGER
  return parts.reduce((acc, p) => acc * 256 + (Number(p) || 0), 0)
}

export function displayName(d: { label: string; hostname: string }): string {
  return d.label || d.hostname || ""
}
