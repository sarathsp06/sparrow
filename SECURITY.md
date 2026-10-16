# Security policy

## Reporting a vulnerability

Please do not open a public issue for a security problem. Report it
privately in either of these ways:

- **GitHub private vulnerability reporting**: open a draft advisory at
  <https://github.com/sarathsp06/sparrow/security/advisories/new>. Only the
  maintainers can see it.
- **Email** **sarath.sp06@gmail.com**.

Include:

- the affected version or commit,
- what an attacker needs to start with (no credentials, a consumer-scoped
  token, a tenant-wide token, the master key),
- steps or a reproducer, and the impact you observed.

You will get an acknowledgement within 14 days, usually within a few days. Fixes ship in a patch
release, and the release notes credit the reporter unless you ask otherwise.

## Supported versions

Only the latest release receives security fixes. Upgrade to it before
reporting, if you can.

## Scope

The server (`cmd/server`), the CLI and satellites under `satellites/`, and the
libraries under `pkg/` are in scope. [`.oss-scanner/threat_model.md`](.oss-scanner/threat_model.md)
describes the trust boundaries and how findings are rated. Behaviour documented
as a known gap in [AGENTS.md](AGENTS.md) (for example, no inbound rate limiting
on `/v1`) is not a vulnerability by itself. See
[Securing Sparrow](https://sarathsp06.github.io/sparrow/deployment/security/)
for deployment guidance.
