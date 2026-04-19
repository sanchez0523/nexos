## What

<!-- One paragraph: what does this PR change, and which component? -->

## Why

<!-- Link an issue or explain the motivation. If this is a bug fix, describe
     the root cause, not just the symptom. -->

## Scope & invariants

- [ ] I've read [ARCHITECTURE.md → Architecture Invariants](../ARCHITECTURE.md#architecture-invariants)
      and this PR does not violate any of them.
- [ ] This PR is focused on one change. Unrelated refactors are split out.

## Verification

- [ ] `make ci` passes locally
- [ ] New code has tests matching [ARCHITECTURE.md → Testing Philosophy](../ARCHITECTURE.md#testing-philosophy)
- [ ] For UI changes: tested in a browser against a running `docker compose up -d`
