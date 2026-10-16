# Governance

Sparrow is a single-maintainer open source project. This page says who decides
what, and how that can change.

## Roles

**Maintainer**: [Sarath Sadasivan Pillai](https://github.com/sarathsp06)
(`@sarathsp06`). The maintainer:

- reviews and merges pull requests, and owns `main`;
- cuts releases (see [CONTRIBUTING.md](CONTRIBUTING.md#release-process));
- triages issues and responds to vulnerability reports
  (see [SECURITY.md](SECURITY.md));
- decides on the design principles in [AGENTS.md](AGENTS.md#design-principles)
  and the [compatibility policy](https://sarathsp06.github.io/sparrow/reference/compatibility/);
- enforces the [Code of Conduct](CODE_OF_CONDUCT.md).

**Contributors**: anyone who opens an issue, a discussion or a pull request.
Contributions are accepted under the requirements in
[CONTRIBUTING.md](CONTRIBUTING.md#requirements-for-acceptable-contributions).

## How decisions are made

- Day-to-day changes are decided in their pull request.
- Larger changes (a new component, a data model or API change, anything that
  touches the compatibility policy) start as an issue so the approach can be
  discussed before code is written. Lasting architectural decisions are
  recorded as ADRs in [`docs/adr/`](docs/adr/).
- When contributors disagree, the maintainer makes the final call and explains
  it in the issue or pull request.

## Becoming a maintainer

A regular contributor whose pull requests consistently meet the contribution
requirements can be invited to become a maintainer. New maintainers are added
to this file and get merge rights; release and security responsibilities are
shared once they have reviewed a release with an existing maintainer.

## Continuity

Sparrow currently has one maintainer, so the project depends on one person.
Everything needed to carry it on is public: the source, issues, CI and release
workflows (a release is one tag push), and the documentation. If the
maintainer becomes unavailable, the project can be forked and continued under
its MIT license. Adding a second maintainer with admin access to the
repository and the container registry is a goal; see
[Becoming a maintainer](#becoming-a-maintainer).

## Changing this document

Changes to governance are made by pull request and approved by the
maintainers.
