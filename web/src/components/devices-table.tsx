import { useMemo, useRef, useState } from "react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { ArrowDown, ArrowUp, ShieldAlert, ArrowRightLeft, Globe, Circle } from "lucide-react"

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
import { IconPicker } from "@/components/icon-picker"
import { updateDevice, type Device, type DeviceStatus, type Overview } from "@/lib/api"
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

const statusLabel: Record<DeviceStatus, string> = { online: "Online", seen: "Idle", offline: "Offline" }

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
      return new Date(a.last_seen).getTime() - new Date(b.last_seen).getTime()
  }
}

interface DevicesTableProps {
  filter: "all" | "online" | "offline"
  unavailable: boolean
  onReset: () => void
  devices?: Device[]
  query: string
  now: number
}

export function DevicesTable({ devices, query, now, filter, unavailable, onReset }: DevicesTableProps) {
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
          aria-label={`Sort by ${label || "status"}`}
          onClick={() => setSort({ key, desc: active ? !sort.desc : false })}
          className={cn(
            "inline-flex min-h-10 items-center gap-1 rounded-sm text-xs font-medium transition-colors hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-foreground",
            active ? "text-foreground" : "text-muted-foreground"
          )}
        >
          {label || <Circle className="size-3" aria-hidden="true" />}
          <Arrow className={cn("size-3", !active && "invisible", !label && "hidden")} />
        </button>
      </TableHead>
    )
  }

  return (
    <div className="overflow-hidden rounded-xl bg-card ring-1 ring-foreground/10">
      <Table aria-label="Network devices">
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
          {rows === undefined && !unavailable &&
            Array.from({ length: 6 }, (_, i) => (
              <TableRow key={i} className="hover:bg-transparent">
                <TableCell colSpan={6} className="px-4">
                  <Skeleton className="h-5 w-full" />
                </TableCell>
              </TableRow>
            ))}

          {(rows?.length === 0 || unavailable) && (
            <TableRow className="hover:bg-transparent">
              <TableCell colSpan={6} className="h-32 whitespace-normal px-4 text-center text-sm text-muted-foreground">
                {unavailable ? "Devices will appear when Scout reconnects." : query.trim() ? "No devices match your search." : filter !== "all" ? `No ${filter} devices.` : "No devices yet. Run a scan to discover your network."}
                {!unavailable && (query.trim() || filter !== "all") && <button type="button" onClick={onReset} className="mx-auto mt-2 block rounded-sm px-2 py-1 text-foreground underline underline-offset-4">Clear filters</button>}
              </TableCell>
            </TableRow>
          )}

          {rows?.map((d) => {
            const vendor = vendorLabel(d)
            return (
              <TableRow key={d.ip} className={cn(d.status === "offline" && "text-muted-foreground")}>
                <TableCell className="pl-4">
                  <span role="img" className="flex size-4 items-center justify-center" aria-label={statusLabel[d.status]}>
                    <span className={cn("size-2 rounded-full", statusDot[d.status])} />
                  </span>
                </TableCell>

                <TableCell className="max-w-0 w-full md:w-auto md:max-w-80">
                  <div className="flex min-w-0 items-center gap-2.5">
                    <DeviceIdentity device={d} />
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

function DeviceIdentity({ device }: { device: Device }) {
  const queryClient = useQueryClient()
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState("")
  const [pickerOpen, setPickerOpen] = useState(false)
  const [error, setError] = useState("")
  const cancelled = useRef(false)
  const name = deviceName(device)
  const save = useMutation({
    mutationFn: (fields: { label?: string; icon?: string }) => updateDevice(device.ip, fields),
    onMutate: (fields) => {
      setError("")
      const previous = queryClient.getQueryData<Overview>(["overview"])
      queryClient.setQueryData<Overview>(["overview"], (current) => current && ({
        ...current,
        devices: current.devices.map((row) => row.ip === device.ip ? { ...row, ...fields } : row),
      }))
      return { previous }
    },
    onError: (failure, _fields, context) => {
      if (context?.previous) queryClient.setQueryData(["overview"], context.previous)
      setError(failure.message)
    },
    onSettled: () => void queryClient.invalidateQueries({ queryKey: ["overview"] }),
  })

  const commitName = () => {
    if (cancelled.current) { cancelled.current = false; return }
    setEditing(false)
    const label = draft.trim()
    if (label !== device.label) save.mutate({ label })
  }

  return <>
    <button type="button" aria-label={`Change icon for ${name.text}`} onClick={() => setPickerOpen(true)} className="flex size-7 shrink-0 items-center justify-center rounded-md hover:bg-accent focus-visible:outline-2 focus-visible:outline-ring">
      <DeviceIcon device={device} className="size-4 object-contain text-muted-foreground" />
    </button>
    {editing ? <input
      autoFocus
      value={draft}
      maxLength={120}
      aria-label={`Rename ${name.text}`}
      onFocus={(event) => event.currentTarget.select()}
      onChange={(event) => setDraft(event.target.value)}
      onBlur={commitName}
      onKeyDown={(event) => {
        if (event.key === "Enter") event.currentTarget.blur()
        if (event.key === "Escape") { cancelled.current = true; setEditing(false) }
      }}
      className="h-8 min-w-0 flex-1 rounded-md border border-ring bg-background px-2 text-sm text-foreground outline-none"
    /> : <button
      type="button"
      title={name.full}
      onClick={() => { cancelled.current = false; setDraft(name.text); setEditing(true) }}
      className={cn("min-w-0 truncate rounded-sm text-left hover:underline hover:underline-offset-2 focus-visible:outline-2 focus-visible:outline-ring", name.derived && "text-muted-foreground")}
    >{name.text}</button>}
    {error && <span role="alert" className="max-w-32 truncate text-xs text-red-400" title={error}>Could not save</span>}
    <IconPicker open={pickerOpen} onOpenChange={setPickerOpen} selected={device.icon} deviceName={name.text} onSelect={(icon) => { setPickerOpen(false); if (icon !== (device.icon ?? "")) save.mutate({ icon }) }} />
  </>
}

function RowFlags({ device: d }: { device: Device }) {
  const moved = d.address_history?.at(-1)
  if (!d.risks?.length && !moved && !d.web_ui) return null

  return (
    <div className="ml-1 flex shrink-0 items-center gap-1">
      {d.web_ui && (
        <Tooltip>
          <TooltipTrigger asChild>
            <Badge tabIndex={0} aria-label="Serves a web interface" variant="outline" className="size-5 p-0 text-muted-foreground">
              <Globe />
            </Badge>
          </TooltipTrigger>
          <TooltipContent>Serves a web interface</TooltipContent>
        </Tooltip>
      )}
      {!!d.risks?.length && (
        <Tooltip>
          <TooltipTrigger asChild>
            <Badge tabIndex={0} aria-label={d.risks.join(" · ")} variant="outline" className="size-5 border-red-500/30 p-0 text-red-400">
              <ShieldAlert />
            </Badge>
          </TooltipTrigger>
          <TooltipContent>{d.risks.join(" · ")}</TooltipContent>
        </Tooltip>
      )}
      {moved && (
        <Tooltip>
          <TooltipTrigger asChild>
            <Badge tabIndex={0} aria-label={`Previously at ${moved.ip}`} variant="outline" className="size-5 border-amber-500/30 p-0 text-amber-400">
              <ArrowRightLeft />
            </Badge>
          </TooltipTrigger>
          <TooltipContent>Previously at {moved.ip}</TooltipContent>
        </Tooltip>
      )}
    </div>
  )
}
