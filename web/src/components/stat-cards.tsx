import { Card } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import type { Overview } from "@/lib/api"
import { cn } from "@/lib/utils"

export type StatusFilter = "all" | "online" | "offline"

interface StatCardsProps {
  data?: Overview
  filter: StatusFilter
  onFilter: (f: StatusFilter) => void
}

export function StatCards({ data, filter, onFilter }: StatCardsProps) {
  const networks = data?.networks.filter((n) => !n.is_tailscale) ?? []

  const cards: {
    key: string
    label: string
    value?: number
    hint?: string
    dot?: string
    filter?: StatusFilter
  }[] = [
    {
      key: "all",
      label: "Devices",
      value: data?.stats.total,
      filter: "all",
      hint: data ? `${data.flagged} flagged · ${data.moved} moved` : undefined,
    },
    {
      key: "online",
      label: "Online",
      value: data?.stats.online,
      dot: "bg-emerald-500",
      filter: "online",
      hint: data ? "seen in the last hour" : undefined,
    },
    {
      key: "offline",
      label: "Offline",
      value: data?.stats.offline,
      dot: "bg-zinc-600",
      filter: "offline",
      hint: data ? "not seen for an hour" : undefined,
    },
    {
      key: "networks",
      label: "Networks",
      value: data ? networks.length : undefined,
      hint: networks.map((n) => n.cidr).join(", "),
    },
  ]

  return (
    <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
      {cards.map((c) => {
        const interactive = c.filter !== undefined
        const active = interactive && filter === c.filter && c.filter !== "all"
        return (
          <Card
            key={c.key}
            role={interactive ? "button" : undefined}
            tabIndex={interactive ? 0 : undefined}
            aria-pressed={interactive ? active : undefined}
            onClick={() => interactive && onFilter(active ? "all" : c.filter!)}
            onKeyDown={(e) => {
              if (interactive && (e.key === "Enter" || e.key === " ")) {
                e.preventDefault()
                onFilter(active ? "all" : c.filter!)
              }
            }}
            className={cn(
              "gap-1.5 px-4 py-4 transition-colors",
              interactive && "cursor-pointer select-none hover:bg-accent/40",
              active && "ring-foreground/30"
            )}
          >
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              {c.dot && <span className={cn("size-1.5 rounded-full", c.dot)} />}
              {c.label}
            </div>
            {c.value === undefined ? (
              <Skeleton className="h-8 w-12" />
            ) : (
              <div className="text-2xl font-semibold tracking-tight tabular-nums">{c.value}</div>
            )}
            {data ? (
              <div className="truncate text-xs text-muted-foreground" title={c.hint}>
                {c.hint || "none detected"}
              </div>
            ) : (
              <Skeleton className="h-4 w-24" />
            )}
          </Card>
        )
      })}
    </div>
  )
}
