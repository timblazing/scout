import { useMemo, useState } from "react"
import { Dialog } from "radix-ui"
import { Search, X } from "lucide-react"
import { lucideIcons } from "@/lib/lucide-icons"
import { dashboardIcons, dashboardIconUrl, deviceIconNames } from "@/lib/icons"

const featured = ["apple", "raspberry-pi", "tp-link", "ubiquiti", "samsung", "synology", "home-assistant", "docker", "plex", "sonos", "amazon", "google"]

export function IconPicker({ open, onOpenChange, selected, onSelect, deviceName }: {
  open: boolean
  onOpenChange: (open: boolean) => void
  selected?: string
  onSelect: (icon: string) => void
  deviceName: string
}) {
  const [query, setQuery] = useState("")
  const [tab, setTab] = useState<"devices" | "brands">("devices")
  const search = query.trim().toLowerCase()
  const matches = useMemo(() => {
    if (!search) {
      return featured.map((slug) => dashboardIcons.find((icon) => icon.slug === slug)).filter((icon) => icon !== undefined)
    }
    return dashboardIcons.filter((icon) => [icon.slug, ...icon.aliases].some((name) => name.toLowerCase().includes(search))).slice(0, 120)
  }, [search])
  const devices = deviceIconNames.filter((name) => !search || name.includes(search))

  return (
    <Dialog.Root open={open} onOpenChange={(next) => { onOpenChange(next); if (!next) setQuery("") }}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-black/70 backdrop-blur-[2px]" />
        <Dialog.Content className="fixed top-1/2 left-1/2 z-50 flex max-h-[min(80vh,700px)] w-[min(94vw,560px)] -translate-x-1/2 -translate-y-1/2 flex-col overflow-hidden rounded-xl border border-border bg-popover shadow-2xl focus:outline-none">
          <div className="flex items-start justify-between gap-4 border-b border-border px-5 py-4">
            <div>
              <Dialog.Title className="text-base font-semibold">Choose an icon</Dialog.Title>
              <Dialog.Description className="mt-0.5 truncate text-xs text-muted-foreground">{deviceName}</Dialog.Description>
            </div>
            <Dialog.Close className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground focus-visible:outline-2 focus-visible:outline-ring" aria-label="Close icon picker"><X className="size-4" /></Dialog.Close>
          </div>
          <div className="border-b border-border px-5 pt-3">
            <div className="flex gap-5" role="tablist" aria-label="Icon collections">
              <button type="button" role="tab" aria-selected={tab === "devices"} onClick={() => setTab("devices")} className={`border-b-2 pb-2 text-sm ${tab === "devices" ? "border-foreground text-foreground" : "border-transparent text-muted-foreground hover:text-foreground"}`}>Device icons</button>
              <button type="button" role="tab" aria-selected={tab === "brands"} onClick={() => setTab("brands")} className={`border-b-2 pb-2 text-sm ${tab === "brands" ? "border-foreground text-foreground" : "border-transparent text-muted-foreground hover:text-foreground"}`}>Brands & apps</button>
            </div>
          </div>
          <div className="relative mx-5 mt-4">
            <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
            <input autoFocus value={query} onChange={(event) => setQuery(event.target.value)} placeholder={tab === "brands" ? "Search brands and apps…" : "Search device icons…"} aria-label="Search icons" className="h-10 w-full rounded-lg border border-input bg-background pr-3 pl-9 text-sm outline-none placeholder:text-muted-foreground focus-visible:border-ring" />
          </div>
          <div className="min-h-0 flex-1 overflow-y-auto px-5 py-4">
            {tab === "devices" ? (
              <div role="tabpanel" className="grid grid-cols-5 gap-2 sm:grid-cols-7">
                {devices.map((name) => {
                  const Icon = lucideIcons[name]
                  return <button key={name} type="button" title={name.replaceAll("-", " ")} aria-label={name.replaceAll("-", " ")} aria-pressed={selected === `lucide:${name}`} onClick={() => onSelect(`lucide:${name}`)} className="flex aspect-square items-center justify-center rounded-lg border border-transparent bg-background/60 text-muted-foreground hover:border-border hover:bg-accent hover:text-foreground focus-visible:outline-2 focus-visible:outline-ring aria-pressed:border-ring aria-pressed:bg-accent aria-pressed:text-foreground"><Icon className="size-6" strokeWidth={1.7} /></button>
                })}
              </div>
            ) : (
              <div role="tabpanel">
                {!search && <p className="mb-3 text-xs text-muted-foreground">Popular icons · search {dashboardIcons.length.toLocaleString()} available icons</p>}
                <div className="grid grid-cols-4 gap-2 sm:grid-cols-6">
                  {matches.map((icon) => <button key={icon.slug} type="button" title={icon.slug.replaceAll("-", " ")} aria-label={icon.slug.replaceAll("-", " ")} aria-pressed={selected === `dashboard:${icon.slug}`} onClick={() => onSelect(`dashboard:${icon.slug}`)} className="flex aspect-square items-center justify-center rounded-lg border border-transparent bg-background/60 p-2.5 hover:border-border hover:bg-accent focus-visible:outline-2 focus-visible:outline-ring aria-pressed:border-ring aria-pressed:bg-accent"><img src={dashboardIconUrl(icon)} alt="" loading="lazy" className="size-8 object-contain" /></button>)}
                </div>
                {search && matches.length === 120 && <p className="mt-3 text-xs text-muted-foreground">Showing the first 120 matches. Refine your search to see more.</p>}
              </div>
            )}
            {(tab === "devices" ? devices.length === 0 : matches.length === 0) && <p className="py-8 text-center text-sm text-muted-foreground">No matching icons. Try another search.</p>}
          </div>
          <div className="flex justify-between border-t border-border px-5 py-3">
            <button type="button" onClick={() => onSelect("")} className="rounded-md px-2 py-1.5 text-xs text-muted-foreground hover:bg-accent hover:text-foreground focus-visible:outline-2 focus-visible:outline-ring">Use automatic icon</button>
            <span className="self-center text-xs text-muted-foreground">Icons by <a href="https://github.com/homarr-labs/dashboard-icons" target="_blank" rel="noreferrer" className="underline underline-offset-2">Dashboard Icons</a></span>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
