# Contributing to Nexos

Thanks for wanting to help. This project stays healthy because contributors
respect the architectural invariants documented in
[ARCHITECTURE.md](ARCHITECTURE.md#architecture-invariants). Please skim that section
before proposing changes.

## Before you open an issue

- **Bug reports:** include the Nexos version (git tag or commit), the
  `docker compose ps` output, and the relevant logs from
  `docker compose logs -f <service>`.
- **Feature requests:** explain the use case, not just the feature. The
  maintainers will push back hard on anything that breaks an invariant
  (Redis, multi-tenancy, device control, binary payloads, etc.) and
  accept features that compose cleanly with the existing surface.

## Before you open a PR

Run the full checks locally:

```bash
make ci
```

That runs Go tests with `-race`, `golangci-lint`, `svelte-check`, and the
dashboard build. The CI workflow mirrors it exactly, so a green local run
should translate to green CI.

### Commit style

We use Conventional Commits so release notes generate cleanly:

```
feat(alert): support slack webhook format
fix(mqtt): drop invalid payloads silently
docs: clarify Auto-Discovery constraints
test(auth): cover refresh rotation edge case
```

Types: `feat`, `fix`, `perf`, `refactor`, `docs`, `test`. `chore` and `ci`
are allowed but stay out of the changelog.

### Scope discipline

- **Keep PRs small.** One focused change is reviewed in hours; a 2000-line
  mega-PR sits for a week. Split into "foundation" and "usage" if needed.
- **No drive-by refactors.** If you're fixing a bug, the PR should change the
  minimum necessary. Propose refactors in separate issues/PRs.
- **No comments telling us the PR fixed something.** Commit messages and the
  PR description carry that context. Code comments should explain *why*, not
  *when*.

### Tests

Match the testing posture documented in ARCHITECTURE.md:

- Pure logic (MQTT parsing, threshold evaluation, JWT) → table-driven unit tests.
- DB / HTTP / WebSocket → integration tests against real Docker services.
- Do **not** test Fiber middleware or `main.go` wiring.

## Architecture changes

If your PR touches or challenges an ADR in `ARCHITECTURE.md`, open a discussion
first. Specifically, any of the following warrants a pre-PR conversation:

- Adding a service to `docker-compose.yml`.
- Changing the `devices/{id}/{sensor}` topic contract.
- Introducing Redis, message queues, or any server-side telemetry.
- Multi-user / role-based access control.
- Device control endpoints.

These aren't off-limits forever, but they break the current design contract
and need a clear motivation shared with the community before code lands.

## Code of Conduct

Be direct, be kind, be specific. No harassment, no gatekeeping, no
nitpicking that isn't actionable. Maintainers reserve the right to lock or
close threads that don't follow this.

## Good first issues

Check the [`good first issue`](https://github.com/OWNER/REPO/labels/good%20first%20issue)
label. If the project is brand-new, maintainers seed a few of these on each
release to help newcomers ship their first PR.
