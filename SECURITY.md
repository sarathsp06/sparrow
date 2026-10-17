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

## How a report is handled

1. **Acknowledge** within 14 days and confirm whether the issue reproduces.
2. **Assess** severity using the ratings in the
   [threat model](.oss-scanner/threat_model.md#severity), and agree a
   disclosure date with the reporter (normally within 90 days of the report).
3. **Fix** in private, through a GitHub security advisory and its temporary
   private fork, with a regression test.
4. **Release** a patch version. Its CHANGELOG.md section lists the fix under
   `### Security`, and the advisory is published with a CVE when one applies.
5. **Credit** the reporter in the advisory and the release notes, unless they
   ask not to be named.

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
