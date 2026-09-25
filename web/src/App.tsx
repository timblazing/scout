import { useEffect, useMemo, useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { AlertTriangle, RefreshCw, Search } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { DevicesTable } from "@/components/devices-table"
import { StatCards, type StatusFilter } from "@/components/stat-cards"
import { getOverview, startScan } from "@/lib/api"
import { timeAgo } from "@/lib/format"
import { cn } from "@/lib/utils"

export default function App() {
  const queryClient = useQueryClient()
  const [query, setQuery] = useState("")
  const [filter, setFilter] = useState<StatusFilter>("all")
  const now = useNow(15_000)

  const overview = useQuery({
    queryKey: ["overview"],
    queryFn: getOverview,
    // Poll quickly while a scan is running so results stream in, and slowly
    // otherwise; the server keeps scanning on its own schedule.
    refetchInterval: (q) => (q.state.data?.scanning ? 2_000 : 15_000),
  })

  const scan = useMutation({
    mutationFn: startScan,
    onSettled: () => queryClient.invalidateQueries({ queryKey: ["overview"] }),
  })

  const data = overview.data
  const scanning = scan.isPending || !!data?.scanning

  const devices = useMemo(() => {
    if (!data) return undefined
    if (filter === "online") return data.devices.filter((d) => d.status !== "offline")
    if (filter === "offline") return data.devices.filter((d) => d.status === "offline")
    return data.devices
  }, [data, filter])

  return (
    <main className="mx-auto flex w-full max-w-6xl flex-col gap-4 px-4 py-8 sm:px-6 sm:py-12">
      {data?.network_warning && (
        <div className="flex gap-3 rounded-xl border border-amber-500/20 bg-amber-500/5 px-4 py-3 text-sm text-amber-200/90">
          <AlertTriangle className="mt-0.5 size-4 shrink-0 text-amber-400" />
          <p>{data.network_warning}</p>
        </div>
      )}

      {overview.isError && (
        <div className="rounded-xl border border-red-500/20 bg-red-500/5 px-4 py-3 text-sm text-red-300">
          Could not reach the Scout server: {overview.error.message}
        </div>
      )}

      <StatCards data={data} filter={filter} onFilter={setFilter} />

      <div className="flex flex-col gap-2 pt-4 sm:flex-row sm:items-center">
        <div className="relative flex-1 sm:max-w-xs">
          <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search devices…"
            aria-label="Search devices"
            className="pl-8"
          />
        </div>
        <div className="flex items-center justify-between gap-3 sm:ml-auto">
          <span className="text-xs text-muted-foreground tabular-nums" title={data?.last_scan && new Date(data.last_scan).toLocaleString()}>
            {scanning ? (
              "Scanning…"
            ) : (
              <>
                {data?.continuous_scan && (
                  <span className="mr-1.5 inline-block size-1.5 animate-pulse rounded-full bg-emerald-500 align-middle" />
                )}
                Last scan {timeAgo(data?.last_scan, now)}
              </>
            )}
          </span>
          <Button variant="outline" size="sm" onClick={() => scan.mutate()} disabled={scanning}>
            <RefreshCw className={cn(scanning && "animate-spin")} />
            Scan
          </Button>
        </div>
      </div>

      {scan.isError && <p className="text-sm text-red-400">{scan.error.message}</p>}

      <DevicesTable devices={devices} query={query} now={now} />
    </main>
  )
}

// useNow re-renders on an interval so relative times keep ticking between
// polls.
function useNow(intervalMs: number) {
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), intervalMs)
    return () => clearInterval(id)
  }, [intervalMs])
  return now
}
