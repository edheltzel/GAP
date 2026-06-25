# packages — DOX Instructions

## Purpose

Installable agent packages that can be loaded independently from the stowed global config.

## Ownership

- Each package owns its package manifest, runtime entry points, bundled skills, and human-facing README.
- Package-local AGENTS.md files override this document for their subtrees without weakening root DOX rules.

## Local Contracts

- Prefer agent-native APIs over direct ports from another agent runtime.
- Keep package manifests aligned with actual shipped files and entry points.
- Keep README usage examples aligned with command behavior.
- Do not add runtime state, credentials, generated caches, or local-only install artifacts to packages.

## Work Guidance

- For OMP packages, use `package.json#omp.extensions` and OMP extension APIs.
- For Pi packages, use `package.json#pi` metadata and Pi extension APIs.
- Any meaningful behavior change should update both README and any bundled skill guidance.

## Verification

Run the narrow package checks documented in the package-local AGENTS.md when present. If no child verification exists, at minimum parse the package manifest and run the runtime's extension load smoke test when available.

## Child DOX Index

- `pi-themis/AGENTS.md` — Pi-native Themis package rules.
- `omp-themis/AGENTS.md` — OMP-native Themis package rules.
