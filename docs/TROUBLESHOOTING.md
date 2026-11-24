# Troubleshooting

## No MAC addresses or vendor info

MAC addresses and vendor information require elevated privileges to read the ARP table.

**Solution:** Run with sudo (or as Administrator on Windows):

```bash
# Linux/macOS
sudo orangutan serve

# Windows - run Command Prompt as Administrator
orangutan.exe serve
```

## Service won't start

```bash
# Check service status
sudo systemctl status lan-orangutan

# View logs
sudo journalctl -u lan-orangutan -n 50

# Check if binary exists and is executable
ls -la /usr/local/bin/orangutan
```

## Port already in use

```bash
# Check what's using the port
sudo ss -tulnp | grep :291

# Use a different port
orangutan serve --port 8080
```

Or change the port in your config file.

## No devices found

1. **Check nmap is installed:**
   ```bash
   which nmap
   nmap --version
   ```

2. **Test nmap directly:**
   ```bash
   sudo nmap -sn 192.168.1.0/24
   ```

3. **Check which networks Scout scans.** Scout scans the networks it detects, plus
   `ORANGUTAN_NETWORKS`, minus `ORANGUTAN_EXCLUDE_NETWORKS`. It skips Tailscale
   addresses (100.64.0.0/10) and single-address (/32) networks. A device must
   answer two scans in a row to drop off the list, so a quiet device can take a
   minute to disappear.

## Can't access web UI

1. **Check if server is running:**
   ```bash
   curl http://localhost:291/api/devices
   ```

2. **Check firewall:**
   ```bash
   # Linux (ufw)
   sudo ufw status
   sudo ufw allow 291/tcp

   # Linux (firewalld)
   sudo firewall-cmd --list-ports
   sudo firewall-cmd --add-port=291/tcp --permanent
   ```

3. **Check if binding to correct address:**
   ```bash
   # Allow external access
   orangutan serve --bind 0.0.0.0
   ```

   This is the default. Scout has no authentication: only expose this port to
   a trusted network, or use an authenticated reverse proxy.

## Data not persisting

Scout only saves device names and icons, in `names.json`. Check that the data directory exists and is writable:

```bash
# Linux
ls -la ~/.local/share/lan-orangutan/
# or as root
ls -la /var/lib/lan-orangutan/

# macOS
ls -la ~/Library/Application\ Support/lan-orangutan/
```

## View debug output

```bash
orangutan version
```

## Still having issues?

1. Check the [GitHub Issues](https://github.com/timblazing/scout/issues)
2. Open a new issue with:
   - Your OS and version
   - Output of `orangutan version`
   - Output of `curl http://localhost:291/api/devices`
   - Steps to reproduce the problem
