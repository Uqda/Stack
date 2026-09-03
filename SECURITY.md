# Security policy

## Supported versions

Only the newest stable UQDA Stack release receives security fixes.

## Reporting

Do not open a public issue for a suspected vulnerability. Use GitHub's private
vulnerability reporting for `Uqda/Stack`, including reproduction steps, affected
versions and impact where possible.

## Security boundaries

- UQDA Stack provides encrypted overlay transport, not anonymity.
- SOCKS5 has no application-layer authentication; keep it on loopback.
- `-allow-public-socks` is an explicit dangerous override and requires an
  external firewall and access policy.
- A remote forward intentionally exposes the selected local service to nodes
  able to route to this UQDA address. The service still needs authentication.
- Protect configuration files and private keys from other local users.
- Prefer peer public-key pinning and `secure=required` on controlled links.
- Use `socks5h` in clients when names must be resolved through the proxy.
- Run Core and Stack with separate identities when both are active.

The project has not yet received an independent security audit.
