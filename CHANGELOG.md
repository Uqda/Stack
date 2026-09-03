# Changelog

## 0.1.0 - Unreleased

- Derive UQDA Stack from Yggstack 1.0.5 with complete upstream attribution.
- Replace the upstream network engine with UQDA Core v0.1.4.
- Carry UQDA's patched Ironwood snapshot to preserve peer-debug race safety.
- Rebrand the executable, Go module, container and documentation as UQDA Stack.
- Restrict SOCKS5 to loopback unless `-allow-public-socks` is explicit.
- Create Unix SOCKS sockets with mode `0600` and safely clean stale sockets.
- Bind local TCP/UDP forwards to loopback by default.
- Reject local-forward targets outside the UQDA `0200::/7` address range.
- Add `.pk.uqda` public-key names while retaining `.pk.ygg` compatibility.
- Add automation-safe colors with `NO_COLOR` and redirected-output behavior.
- Correct IPv4-to-IPv4 four-part mapping parsing and validate port ranges.
- Deliver the first UDP datagram immediately and avoid copying synchronized
  gVisor connection values.
- Add unit, race, formatting, vulnerability, cross-build and release gates.
