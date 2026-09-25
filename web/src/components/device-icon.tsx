import {
  Cpu,
  Gamepad2,
  HardDrive,
  Laptop,
  type LucideIcon,
  Monitor,
  Printer,
  Router,
  Server,
  Smartphone,
  Tv,
  CirclePlay,
} from "lucide-react"

const icons: Record<string, LucideIcon> = {
  Phone: Smartphone,
  Computer: Laptop,
  Printer: Printer,
  TV: Tv,
  "Media Player": CirclePlay,
  Router: Router,
  Server: Server,
  IoT: Cpu,
  "Game Console": Gamepad2,
  NAS: HardDrive,
}

export function DeviceIcon({ type, className }: { type?: string; className?: string }) {
  const Icon = (type && icons[type]) || Monitor
  return <Icon className={className} aria-hidden />
}
