import { useEffect, useRef, useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { AlertTriangle, Radar, Search } from "lucide-react"

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
    <main className="mx-auto flex w-full max-w-4xl flex-col gap-4 px-4 py-8 sm:px-6 sm:py-10">
      <header className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-baseline gap-2.5">
          <Radar className="size-5 self-center" strokeWidth={1.5} aria-hidden="true" />
          <h1 className="text-lg font-semibold tracking-tight">Scout</h1>
          {count !== undefined && (
            <span role="status" className="text-sm text-muted-foreground tabular-nums">
              {count} {count === 1 ? "device" : "devices"}
            </span>
          )}
        </div>
        <div className="relative sm:w-64">
          <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            ref={searchRef}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search devices…"
            aria-label="Search devices"
            className="pl-8"
          />
        </div>
      </header>

      {data?.warning && (
        <div className="flex gap-3 rounded-xl border border-amber-500/20 bg-amber-500/5 px-4 py-3 text-sm text-amber-200/90">
          <AlertTriangle className="mt-0.5 size-4 shrink-0 text-amber-400" />
          <p>{data.warning}</p>
        </div>
      )}

      {list.isError && (
        <div role="alert" className="rounded-xl border border-red-500/20 bg-red-500/5 px-4 py-3 text-sm text-red-300">
          Could not reach Scout. {data ? "Showing the last available devices." : "Check that the server is running."}
          <button type="button" className="ml-2 underline underline-offset-4" onClick={() => void list.refetch()}>Retry</button>
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
  )
}
