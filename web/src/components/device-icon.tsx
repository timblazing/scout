import { useState } from "react"
import type { Device } from "@/lib/api"
import { automaticDashboardIcon, dashboardIconFor, dashboardIconUrl, type DeviceIconName } from "@/lib/icons"
import { lucideIcons } from "@/lib/lucide-icons"

const typeIcons: Record<string, DeviceIconName> = {
  Phone: "smartphone", Computer: "laptop", Printer: "printer", TV: "tv",
  "Media Player": "circle-play", Router: "router", Server: "server", IoT: "cpu",
  "Game Console": "gamepad-2", NAS: "hard-drive",
}

export function DeviceIcon({ device, className }: { device: Pick<Device, "icon" | "label" | "hostname" | "vendor" | "type">; className?: string }) {
  const [failedUrl, setFailedUrl] = useState("")
  const chosen = dashboardIconFor(device.icon) ?? (!device.icon ? automaticDashboardIcon(device) : undefined)
  const url = chosen && dashboardIconUrl(chosen)
  if (url && failedUrl !== url) {
    return <img src={url} alt="" className={className} loading="lazy" onError={() => setFailedUrl(url)} />
  }
  const isRobotVacuum = /roborock|robot.?vacuum|roomba/i.test(`${device.vendor} ${device.hostname} ${device.label}`)
  const name = device.icon?.startsWith("lucide:") ? device.icon.slice(7) : isRobotVacuum && !device.icon ? "bot" : typeIcons[device.type ?? ""]
  const Icon = lucideIcons[name as DeviceIconName] ?? lucideIcons.monitor
  return <Icon className={className} aria-hidden="true" />
}
