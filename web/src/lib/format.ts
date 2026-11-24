export function ipKey(ip: string): number {
  const parts = ip.split(".")
  if (parts.length !== 4) return Number.MAX_SAFE_INTEGER
  return parts.reduce((acc, p) => acc * 256 + (Number(p) || 0), 0)
}
