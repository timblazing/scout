import {
  Bot, Box, Camera, Car, CirclePlay, Cpu, Gamepad2, HardDrive, House, Laptop, Lightbulb,
  Monitor, Network, Plug, Printer, Radio, Router, Server, Shield, Smartphone, Speaker,
  Tablet, Thermometer, Tv, Watch, Wifi, type LucideIcon,
} from "lucide-react"
import type { DeviceIconName } from "@/lib/icons"

export const lucideIcons: Record<DeviceIconName, LucideIcon> = {
  monitor: Monitor, laptop: Laptop, smartphone: Smartphone, tablet: Tablet, router: Router,
  server: Server, "hard-drive": HardDrive, printer: Printer, tv: Tv, "circle-play": CirclePlay,
  "gamepad-2": Gamepad2, cpu: Cpu, camera: Camera, speaker: Speaker, wifi: Wifi,
  network: Network, radio: Radio, plug: Plug, lightbulb: Lightbulb, thermometer: Thermometer,
  watch: Watch, car: Car, bot: Bot, shield: Shield, house: House, box: Box,
}
