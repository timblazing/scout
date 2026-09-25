import { useMemo, useState } from "react"
import { ArrowDown, ArrowUp, ShieldAlert, ArrowRightLeft, Globe } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { DeviceIcon } from "@/components/device-icon"
import type { Device, DeviceStatus } from "@/lib/api"
import { deviceName, vendorLabel } from "@/lib/device"
import { ipKey, timeAgo } from "@/lib/format"
import { cn } from "@/lib/utils"

type SortKey = "status" | "name" | "ip" | "mac" | "vendor" | "last_seen"

const statusRank: Record<DeviceStatus, number> = { online: 0, seen: 1, offline: 2 }

const statusDot: Record<DeviceStatus, string> = {
  online: "bg-emerald-500 shadow-[0_0_0_3px] shadow-emerald-500/15",
  seen: "bg-amber-400",
  offline: "bg-zinc-600",
}

const statusLabel: Record<DeviceStatus, string> = {
  online: "Online · seen in the last 5 minutes",
  seen: "Idle · seen in the last hour",
  offline: "Offline",
}

function compare(a: Device, b: Device, key: SortKey): number {
  switch (key) {
    case "status":
      return statusRank[a.status] - statusRank[b.status] || ipKey(a.ip) - ipKey(b.ip)
    case "name":
      return deviceName(a).text.localeCompare(deviceName(b).text) || ipKey(a.ip) - ipKey(b.ip)
    case "ip":
      return ipKey(a.ip) - ipKey(b.ip) || a.ip.localeCompare(b.ip)
    case "mac":
      return a.mac.localeCompare(b.mac)
    case "vendor":
      return vendorLabel(a).text.localeCompare(vendorLabel(b).text) || ipKey(a.ip) - ipKey(b.ip)
    case "last_seen":
      return new Date(b.last_seen).getTime() - new Date(a.last_seen).getTime()
  }
}

interface DevicesTableProps {
  devices?: Device[]
  query: string
  now: number
}

export function DevicesTable({ devices, query, now }: DevicesTableProps) {
  const [sort, setSort] = useState<{ key: SortKey; desc: boolean }>({ key: "ip", desc: false })

  const rows = useMemo(() => {
    if (!devices) return undefined
    const q = query.trim().toLowerCase()
    const filtered = q
      ? devices.filter((d) =>
          [d.ip, d.mac, d.hostname, d.label, d.vendor, d.type ?? "", deviceName(d).text].some((v) =>
            v.toLowerCase().includes(q)
          )
        )
      : devices
    const sorted = [...filtered].sort((a, b) => compare(a, b, sort.key))
    return sort.desc ? sorted.reverse() : sorted
  }, [devices, query, sort])

  const header = (key: SortKey, label: string, className?: string) => {
    const active = sort.key === key
    const Arrow = sort.desc ? ArrowDown : ArrowUp
    return (
      <TableHead className={className} aria-sort={active ? (sort.desc ? "descending" : "ascending") : "none"}>
        <button
          type="button"
          onClick={() => setSort({ key, desc: active ? !sort.desc : false })}
          className={cn(
            "inline-flex items-center gap-1 text-xs font-medium transition-colors hover:text-foreground",
            active ? "text-foreground" : "text-muted-foreground"
          )}
        >
          {label}
          <Arrow className={cn("size-3", !active && "invisible", !label && "hidden")} />
        </button>
      </TableHead>
    )
  }

  return (
    <div className="overflow-hidden rounded-xl bg-card ring-1 ring-foreground/10">
      <Table>
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            {header("status", "", "w-10 pl-4")}
            {header("name", "Device")}
            {header("ip", "IP address")}
            {header("mac", "MAC address", "hidden md:table-cell")}
            {header("vendor", "Vendor", "hidden lg:table-cell")}
            {header("last_seen", "Last seen", "hidden pr-4 text-right sm:table-cell")}
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows === undefined &&
            Array.from({ length: 6 }, (_, i) => (
              <TableRow key={i} className="hover:bg-transparent">
                <TableCell colSpan={6} className="px-4">
                  <Skeleton className="h-5 w-full" />
                </TableCell>
              </TableRow>
            ))}

          {rows?.length === 0 && (
            <TableRow className="hover:bg-transparent">
              <TableCell colSpan={6} className="h-32 text-center text-sm text-muted-foreground">
                {query ? "No devices match your search." : "No devices yet. Run a scan to discover your network."}
              </TableCell>
            </TableRow>
          )}

          {rows?.map((d) => {
            const name = deviceName(d)
            const vendor = vendorLabel(d)
            return (
              <TableRow key={d.ip} className={cn(d.status === "offline" && "text-muted-foreground")}>
                <TableCell className="pl-4">
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <span className="flex size-4 items-center justify-center" aria-label={statusLabel[d.status]}>
                        <span className={cn("size-2 rounded-full", statusDot[d.status])} />
                      </span>
                    </TooltipTrigger>
                    <TooltipContent>{statusLabel[d.status]}</TooltipContent>
                  </Tooltip>
                </TableCell>

                <TableCell>
                  <div className="flex items-center gap-2.5">
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <span className="shrink-0" aria-label={d.type || "Unknown type"}>
                          <DeviceIcon type={d.type} className="size-4 text-muted-foreground" />
                        </span>
                      </TooltipTrigger>
                      <TooltipContent>{d.type || "Unknown type"}</TooltipContent>
                    </Tooltip>
                    <span
                      className={cn("min-w-0 truncate", name.derived && "text-muted-foreground")}
                      title={name.full}
                    >
                      {name.text}
                    </span>
                    <RowFlags device={d} />
                  </div>
                </TableCell>

                <TableCell className="font-mono text-[13px] tabular-nums">
                  {d.web_ui ? (
                    <a
                      href={`http://${d.ip}`}
                      target="_blank"
                      rel="noreferrer"
                      className="underline-offset-4 hover:underline"
                    >
                      {d.ip}
                    </a>
                  ) : (
                    d.ip
                  )}
                </TableCell>

                <TableCell className="hidden font-mono text-[13px] text-muted-foreground md:table-cell">
                  {d.mac || "—"}
                </TableCell>

                <TableCell
                  className={cn(
                    "hidden max-w-56 truncate text-muted-foreground lg:table-cell",
                    vendor.muted && "text-muted-foreground/60"
                  )}
                  title={vendor.muted && d.mac ? "Randomized MAC address; the manufacturer can't be identified" : undefined}
                >
                  {vendor.text}
                </TableCell>

                <TableCell
                  className="hidden pr-4 text-right text-muted-foreground tabular-nums sm:table-cell"
                  title={new Date(d.last_seen).toLocaleString()}
                >
                  {timeAgo(d.last_seen, now)}
                </TableCell>
              </TableRow>
            )
          })}
        </TableBody>
      </Table>
    </div>
  )
}

function RowFlags({ device: d }: { device: Device }) {
  const moved = d.address_history?.at(-1)
  if (!d.risks?.length && !moved && !d.web_ui) return null

  return (
    <div className="ml-1 flex items-center gap-1">
      {d.web_ui && (
        <Tooltip>
          <TooltipTrigger asChild>
            <Badge variant="outline" className="size-5 p-0 text-muted-foreground">
              <Globe />
            </Badge>
          </TooltipTrigger>
          <TooltipContent>Serves a web interface</TooltipContent>
        </Tooltip>
      )}
      {!!d.risks?.length && (
        <Tooltip>
          <TooltipTrigger asChild>
            <Badge variant="outline" className="size-5 border-red-500/30 p-0 text-red-400">
              <ShieldAlert />
            </Badge>
          </TooltipTrigger>
          <TooltipContent>{d.risks.join(" · ")}</TooltipContent>
        </Tooltip>
      )}
      {moved && (
        <Tooltip>
          <TooltipTrigger asChild>
            <Badge variant="outline" className="size-5 border-amber-500/30 p-0 text-amber-400">
              <ArrowRightLeft />
            </Badge>
          </TooltipTrigger>
          <TooltipContent>Previously at {moved.ip}</TooltipContent>
        </Tooltip>
      )}
    </div>
  )
}
