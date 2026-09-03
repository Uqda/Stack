# UQDA Stack

**Unprivileged SOCKS5 and TCP/UDP forwarding for the encrypted UQDA network**

[![CI](https://github.com/Uqda/Stack/actions/workflows/ci.yml/badge.svg)](https://github.com/Uqda/Stack/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.25.13%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License](https://img.shields.io/badge/License-LGPLv3-blue.svg)](LICENSE)

**English** · [العربية](README_AR.md)

UQDA Stack connects selected applications to UQDA without creating a TUN
interface and without requiring root or Administrator privileges. It embeds a
userspace IPv6/TCP/UDP stack and provides:

- a local SOCKS5 proxy;
- local TCP and UDP forwarding to remote UQDA services;
- remote TCP and UDP forwarding from UQDA to explicitly selected local services;
- built-in public-key names using `<public-key>.pk.uqda`;
- compatibility with the legacy `<public-key>.pk.ygg` suffix;
- the hardened protocol, group authentication and peer options from
  [UQDA Core](https://github.com/Uqda/Core).

UQDA Stack is derived from
[Yggstack](https://github.com/yggdrasil-network/yggstack). See [NOTICE.md](NOTICE.md)
for origin and attribution details.

## Security status

This project has not received an independent security audit and is not an
anonymity system. SOCKS5 binds are restricted to loopback by default. Do not
expose a proxy or forwarded service to an untrusted network without an external
firewall and access controls. See [SECURITY.md](SECURITY.md).

## Build

Requirements: Go 1.25.13 or newer and Git.

```sh
git clone https://github.com/Uqda/Stack.git
cd Stack
./build
```

The output is `uqda-stack` (`uqda-stack.exe` on Windows).

## Quick start

Generate a persistent configuration and add at least one peer:

```sh
mkdir -p "$HOME/.config/uqda-stack"
uqda-stack -genconf > "$HOME/.config/uqda-stack/uqda.conf"
$EDITOR "$HOME/.config/uqda-stack/uqda.conf"
```

Start a loopback-only SOCKS5 proxy:

```sh
uqda-stack \
  -useconffile "$HOME/.config/uqda-stack/uqda.conf" \
  -socks 127.0.0.1:1080
```

Use `socks5h` so hostname resolution also happens through the proxy:

```sh
curl --proxy socks5h://127.0.0.1:1080 \
  'http://[201:db8::1]:8080/'
```

The example address must be replaced with a real UQDA address.

For a temporary identity and local multicast discovery:

```sh
uqda-stack -autoconf -socks 127.0.0.1:1080
```

## Forwarding

Forward local TCP port 8080 to port 80 on a remote UQDA node:

```sh
uqda-stack -useconffile uqda.conf \
  -local-tcp '127.0.0.1:8080:[201:db8::1]:80'
```

Forward local UDP port 5353 to remote UDP port 53:

```sh
uqda-stack -useconffile uqda.conf \
  -local-udp '127.0.0.1:5353:[201:db8::1]:53'
```

Expose a local web service on UQDA port 80:

```sh
uqda-stack -useconffile uqda.conf -remote-tcp '80:127.0.0.1:8080'
```

Expose a local DNS service on UQDA UDP port 53:

```sh
uqda-stack -useconffile uqda.conf -remote-udp '53:127.0.0.1:53'
```

Local listeners default to `127.0.0.1` when no address is supplied. Remote
forwards expose a service on the node's UQDA address and should be treated as a
security boundary.

## DNS

Public-key names work without an external DNS server:

```text
<64-hex-character-public-key>.pk.uqda
```

To resolve other names, configure an IPv6 DNS server reachable over UQDA:

```sh
uqda-stack -useconffile uqda.conf \
  -nameserver '[UQDA-DNS-ADDRESS]:53' \
  -socks 127.0.0.1:1080
```

## Safe listener policy

These are accepted by default:

```text
127.0.0.1:1080
[::1]:1080
/tmp/uqda-stack.sock
```

Wildcard and LAN SOCKS listeners are rejected. `-allow-public-socks` overrides
that protection deliberately, but UQDA Stack does not currently provide SOCKS
username/password authentication. Unix sockets are created with mode `0600`.

## Configuration compatibility

UQDA Stack reads UQDA Core HJSON/JSON configuration and uses UQDA Core v0.1.4.
Its administration listener is always disabled so an unprivileged Stack process
cannot collide with the system UQDA daemon. Use a separate private key when Core
and Stack run simultaneously; reusing one live identity can cause session and
routing conflicts.

## Terminal output

Colors are enabled only for an interactive terminal. Redirected output,
`NO_COLOR`, and `TERM=dumb` remain free of ANSI sequences.

```sh
uqda-stack -color=always ...
uqda-stack -color=never ...
uqda-stack -no-color ...
```

## Development

```sh
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
go build ./...
```

Read [CONTRIBUTING.md](CONTRIBUTING.md) before submitting changes.

## License

UQDA Stack is distributed under GNU LGPLv3 with the additional exception in
[LICENSE](LICENSE). Third-party code remains under its respective licenses.
