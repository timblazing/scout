import { useEffect, useRef, useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { AlertTriangle, Search } from "lucide-react"

import { Input } from "@/components/ui/input"
import { DevicesTable } from "@/components/devices-table"
import { getDevices } from "@/lib/api"
import { loadCatalog } from "@/lib/icons"

export default function App() {
  const [query, setQuery] = useState("")
  const searchRef = useRef<HTMLInputElement>(null)

  const list = useQuery({
    queryKey: ["devices"],
    queryFn: getDevices,
    refetchInterval: 10_000,
  })
  const data = list.data

  // Load the brand-icon catalog once the first paint is done; rows use it to
  // match vendors to icons.
  useEffect(() => {
    void loadCatalog()
  }, [])

  // "/" jumps to the search box, unless the user is already typing somewhere.
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "/" || event.metaKey || event.ctrlKey || event.altKey) return
      const target = event.target as HTMLElement | null
      if (target?.closest("input, textarea, select, [contenteditable='true']")) return
      event.preventDefault()
      searchRef.current?.focus()
    }
    window.addEventListener("keydown", onKeyDown)
    return () => window.removeEventListener("keydown", onKeyDown)
  }, [])

  const count = data?.devices.length

  return (
    <div className="min-h-dvh">
      <main className="mx-auto flex w-full max-w-5xl flex-col gap-6 px-4 py-10 sm:px-6 sm:py-14">
        <header className="flex flex-col gap-5 sm:flex-row sm:items-end sm:justify-between">
          <div className="flex flex-col gap-1.5">
            <h1 className="font-display text-3xl sm:text-4xl">Scout</h1>
            <p role="status" className="text-sm text-muted-foreground tabular-nums">
              {count === undefined
                ? "Looking for devices on your network…"
                : <><span className="text-foreground">{count}</span> {count === 1 ? "device" : "devices"} online</>}
            </p>
          </div>
          <div className="relative sm:w-72">
            <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              ref={searchRef}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Search name, IP, MAC, vendor…"
              aria-label="Search devices"
              className="h-9 pr-9 pl-9"
            />
            <kbd className="pointer-events-none absolute top-1/2 right-2 hidden -translate-y-1/2 rounded border bg-muted/50 px-1.5 text-[11px] leading-5 text-muted-foreground sm:block">/</kbd>
          </div>
        </header>

        {data?.warning && (
          <div className="flex gap-3 rounded-lg border border-warning/20 bg-warning/5 px-4 py-3 text-sm text-warning">
            <AlertTriangle className="mt-0.5 size-4 shrink-0" />
            <p>{data.warning}</p>
          </div>
        )}

        {list.isError && (
          <div role="alert" className="flex items-center justify-between gap-3 rounded-lg border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">
            <span>Could not reach Scout. {data ? "Showing the last available devices." : "Check that the server is running."}</span>
            <button type="button" className="shrink-0 rounded-md border border-destructive/30 px-2.5 py-1 text-xs font-medium hover:bg-destructive/10" onClick={() => void list.refetch()}>Retry</button>
          </div>
        )}

        <DevicesTable
          devices={data?.devices}
          scanning={!!data?.scanning}
          query={query}
          unavailable={list.isError && !data}
          onReset={() => setQuery("")}
        />
      </main>
    </div>
  )
}
