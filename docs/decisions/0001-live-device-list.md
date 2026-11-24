# 0001: Live device list

## Context

Scout kept a persistent record of every device it had ever seen, with first/last seen times, address history, online/offline status, scan state and Tailscale peers. The result was a long table of mostly stale rows, duplicated devices (the same MAC at several addresses, or IPv4 and IPv6 entries), and a lot of code to manage it all. The owner only wants to know what is on the network now.

## Decision

- Scout shows only devices currently on the network, one row per device. The list lives in memory and is rebuilt from scans.
- A device's identity is its lowercase MAC address, or `ip:<ip>` when there is no MAC. IPv6 neighbors and mDNS results only enrich existing devices and never create rows.
- A scan every 60 seconds is the only way the list updates; there is no manual scan. A device missing from a scan gets a miss, and is removed after 2 misses in a row.
- Tailscale peers and /32 networks are excluded entirely: Scout does not scan 100.64.0.0/10 or any /32 network and drops results in that range.
- The only thing persisted is `names.json`, a map from device key to label and icon. When it does not exist, Scout imports names and icons once from the old `devices.json`, which is left untouched.
- The API is `GET /api/devices` and `PATCH /api/devices/{id}`. The CLI keeps only `serve` and `version`.

## Consequences

- The list is accurate and small, and the code is much smaller. Scout has no history, statistics, export or offline devices.
- A restart shows "Scanning your network" until the first scan finishes.
- A device with no MAC is named by IP. If it later appears with a MAC, its key changes and its saved name is not carried over.
- Names for devices that left the network stay in `names.json` and return when the device comes back.
- Removed commands and endpoints (`scan`, `list`, `status`, `export`, `/api/overview`, `/api/scan/*`) are gone, so scripts using them need updating.
