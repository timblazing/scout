import { useMemo, useRef, useState } from "react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { ArrowDown, ArrowUp, ShieldAlert, Globe } from "lucide-react"

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
import { updateDevice, type Device, type DeviceList } from "@/lib/api"
import { deviceName, vendorLabel } from "@/lib/device"
import { ipKey } from "@/lib/format"
import { cn } from "@/lib/utils"

type SortKey = "name" | "ip" | "mac"

function compare(a: Device, b: Device, key: SortKey): number {
  switch (key) {
    case "name":
      return deviceName(a).text.localeCompare(deviceName(b).text) || ipKey(a.ip) - ipKey(b.ip)
    case "ip":
      return ipKey(a.ip) - ipKey(b.ip) || a.ip.localeCompare(b.ip)
    case "mac":
      return a.mac.localeCompare(b.mac)
  }
}

interface DevicesTableProps {
  devices?: Device[]
  scanning: boolean
  unavailable: boolean
  query: string
  onReset: () => void
}

export function DevicesTable({ devices, scanning, unavailable, query, onReset }: DevicesTableProps) {
  const [sort, setSort] = useState<{ key: SortKey; desc: boolean }>({ key: "ip", desc: false })

  const rows = useMemo(() => {
    if (!devices) return undefined
    const q = query.trim().toLowerCase()
    const filtered = q
      ? devices.filter((d) =>
          [d.ip, d.mac, d.hostname, d.label, d.vendor, d.type, deviceName(d).text].some((v) =>
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
          aria-label={`Sort by ${label}`}
          onClick={() => setSort({ key, desc: active ? !sort.desc : false })}
          className={cn(
            "inline-flex min-h-10 items-center gap-1 rounded-sm text-xs font-medium transition-colors hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-foreground",
            active ? "text-foreground" : "text-muted-foreground"
          )}
        >
          {label}
          <Arrow className={cn("size-3", !active && "invisible")} />
        </button>
      </TableHead>
    )
  }

  return (
    <div className="overflow-hidden rounded-xl bg-card ring-1 ring-foreground/10">
      <Table aria-label="Network devices">
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            {header("name", "Device", "pl-4")}
            {header("ip", "IP")}
            {header("mac", "MAC", "hidden pr-4 md:table-cell")}
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows === undefined && !unavailable &&
            Array.from({ length: 6 }, (_, i) => (
              <TableRow key={i} className="hover:bg-transparent">
                <TableCell colSpan={3} className="px-4">
                  <Skeleton className="h-5 w-full" />
                </TableCell>
              </TableRow>
            ))}

          {(rows?.length === 0 || unavailable) && (
            <TableRow className="hover:bg-transparent">
              <TableCell colSpan={3} className="h-32 whitespace-normal px-4 text-center text-sm text-muted-foreground">
                {unavailable
                  ? "Devices will appear when Scout reconnects."
                  : query.trim()
                    ? "No devices match your search."
                    : scanning
                      ? "Scanning your network…"
                      : "No devices found on your network."}
                {!unavailable && query.trim() && <button type="button" onClick={onReset} className="mx-auto mt-2 block rounded-sm px-2 py-1 text-foreground underline underline-offset-4">Clear search</button>}
              </TableCell>
            </TableRow>
          )}

          {rows?.map((d) => (
            <TableRow key={d.id}>
              <TableCell className="max-w-0 w-full pl-4 md:w-auto md:max-w-96">
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

              <TableCell className="hidden pr-4 font-mono text-[13px] text-muted-foreground md:table-cell">
                {d.mac || "—"}
              </TableCell>
            </TableRow>
          ))}
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
  const vendor = vendorLabel(device)
  const detail = name.derived
    ? name.full
    : [vendor.muted ? "" : vendor.text, device.type].filter(Boolean).join(" · ")
  const save = useMutation({
    mutationFn: (fields: { label?: string; icon?: string }) => updateDevice(device.id, fields),
    onMutate: async (fields) => {
      setError("")
      await queryClient.cancelQueries({ queryKey: ["devices"] })
      const previous = queryClient.getQueryData<DeviceList>(["devices"])
      queryClient.setQueryData<DeviceList>(["devices"], (current) => current && ({
        ...current,
        devices: current.devices.map((row) => row.id === device.id ? { ...row, ...fields } : row),
      }))
      return { previous }
    },
    onError: (failure, _fields, context) => {
      if (context?.previous) queryClient.setQueryData(["devices"], context.previous)
      setError(failure.message)
    },
    onSettled: () => void queryClient.invalidateQueries({ queryKey: ["devices"] }),
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
    <div className="min-w-0 flex-1">
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
      {detail && <div className="truncate text-xs text-muted-foreground">{detail}</div>}
    </div>
    {error && <span role="alert" className="max-w-32 truncate text-xs text-red-400" title={error}>Could not save</span>}
    <IconPicker open={pickerOpen} onOpenChange={setPickerOpen} selected={device.icon} deviceName={name.text} onSelect={(icon) => { setPickerOpen(false); if (icon !== (device.icon ?? "")) save.mutate({ icon }) }} />
  </>
}

function RowFlags({ device: d }: { device: Device }) {
  if (!d.risks?.length && !d.web_ui) return null

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
    </div>
  )
}
