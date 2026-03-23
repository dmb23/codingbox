<!--
Sync Impact Report
- Version change: N/A (template) → 1.0.0
- Added principles:
  - I. Verify Before Assuming Success (NEW)
- Added sections:
  - Verification Standards (NEW)
  - Development Workflow (NEW)
- Removed sections:
  - [SECTION_2_NAME] placeholder removed
  - [SECTION_3_NAME] placeholder removed
  - Principles 2–5 placeholders removed (user provided 1 principle)
- Templates requiring updates:
  - .specify/templates/plan-template.md ✅ no changes needed (Constitution Check section is generic)
  - .specify/templates/spec-template.md ✅ no changes needed
  - .specify/templates/tasks-template.md ✅ no changes needed
  - .specify/templates/checklist-template.md ✅ no changes needed
  - .specify/templates/commands/ ✅ no command files exist
- Follow-up TODOs: none
-->

# CodingBox Constitution

## Core Principles

### I. Verify Before Assuming Success (NON-NEGOTIABLE)

A compiling build is NOT proof of success. Every change MUST be
verified through actual execution before it is considered done.

- **CLI work**: Run the command and inspect its output. Verify
  that the behavior matches expectations, not just that it exits
  without error.
- **Non-CLI work**: Write tests that exercise the changed behavior
  and run them. Passing tests are the proof of success.
- **All work**: Never rely solely on compilation, type-checking, or
  linting as evidence that a change works. These catch categories
  of errors but do not prove correctness.

**Rationale**: Silent failures, wrong outputs, and regressions
slip through when verification is skipped. The cost of running
the code is always lower than the cost of debugging a false
assumption later.

## Verification Standards

- For every implementation task, the definition of done includes
  a verification step that exercises the feature end-to-end.
- When fixing a bug, reproduce the bug first, then verify the
  fix eliminates it.
- When adding a feature, demonstrate the feature working before
  marking it complete.
- Automated tests MUST be run, not just written. A test file
  that has never been executed proves nothing.

## Development Workflow

- Implement the change.
- Verify: run the relevant command or test suite.
- Review the output for correctness (not just zero exit code).
- Only then mark the work as done.

## Governance

This constitution is the authoritative source of project
principles. All development work MUST comply with these
principles.

- **Amendments**: Any change to this constitution MUST be
  documented with a version bump and rationale.
- **Versioning**: MAJOR.MINOR.PATCH semantic versioning.
  MAJOR for principle removals or redefinitions, MINOR for
  new principles or material expansions, PATCH for wording
  and clarification changes.
- **Compliance**: Every task checkpoint and PR review MUST
  verify that the verification principle was followed.

**Version**: 1.0.0 | **Ratified**: 2026-03-23 | **Last Amended**: 2026-03-23
