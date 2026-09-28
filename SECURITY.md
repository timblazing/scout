# Security Policy

Scout is a private-network discovery dashboard, forked from LAN Orangutan.
Security fixes target the latest Scout release.

## Reporting a vulnerability

Use [private vulnerability reporting](https://github.com/timblazing/scout/security/advisories/new)
when enabled. Include affected versions, reproduction steps, and impact. Do not
post credentials or private network inventories in public issues.

## Access and privileges

Scout has **no authentication** and listens on `0.0.0.0:291` by default. Anyone
who can reach the service can read device history and use the API, including
scanning, editing or deleting device records, and toggling continuous scanning.
Legacy password, session, and `allow_insecure` settings do not protect Scout.
Keep the port on a trusted network, or use a reverse proxy with authentication.
Scout serves HTTP; use a TLS proxy or private VPN for encrypted remote access.

Mutating API requests require a JSON content type or `X-Requested-With` header.
This browser request check is not authentication or authorization.

Scout can discover Tailscale peers but cannot connect or disconnect the host's
Tailscale session through its API. Settings are managed through configuration
files and environment variables, not a dashboard settings endpoint.

The Docker image runs as root for raw network discovery. Only scan networks you
own or are authorized to scan. Network probes contact devices on those networks;
MAC vendor lookup uses the database embedded in the binary.

Device history is unencrypted JSON on disk. Protect the data directory and its
backups. Scout does not add router port forwarding or provide user accounts,
roles, or per-user permissions.
