# Install Scout

## Docker on Linux

Use the Compose example in [README.md](../README.md). Host networking lets nmap
see the LAN directly. Keep the data volume between container upgrades.

Scout has no authentication. Only expose the port to a trusted network or an
authenticated reverse proxy.

## Native binary

Install nmap, then download your platform's archive from
[Scout releases](https://github.com/timblazing/scout/releases). Extract it and run
the included binary, for example `./orangutan-linux-amd64 serve` on Linux or
`./orangutan-darwin-arm64 serve` on Apple Silicon. Elevated privileges enable
raw ARP discovery and MAC/vendor identification.

The binary name, `ORANGUTAN_*` environment variables, and configuration/data
paths remain compatible with upstream. See [config.example.ini](../config.example.ini).

Linux releases also include a `scout-<version>.tar.gz` bundle with native
binaries and `install.sh` for systemd installation. Review the script before
running it with sudo. It uses the existing `lan-orangutan` service and paths.
Scout does not publish distro packages, Homebrew/Scoop manifests, or a Snap.

## Build from source

Install Go 1.25+, Node 22+, and nmap, then:

```sh
git clone https://github.com/timblazing/scout.git
cd scout
make build
./bin/orangutan serve
```

`make build` builds and embeds the dashboard before compiling Go.

## Verify

```sh
./bin/orangutan version
curl http://localhost:291/api/status
```

For downloaded binaries, substitute the extracted filename in the version command.
