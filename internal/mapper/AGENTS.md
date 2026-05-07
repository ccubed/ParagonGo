# Mapper Package Guide

## Scope

- Use this file for map generation, pathfinding, and room-graph traversal logic in `internal/mapper`.
- This package is sensitive to room topology assumptions and user-facing navigation behavior.

## Working Rules

- Preserve pathfinding correctness before optimizing output shape or performance.
- Be careful with room graph assumptions, exit handling, and distance/radius logic. Several systems depend on those helpers indirectly.
- If you change map output formatting, keep rendering concerns separate from traversal logic where possible.
- When the mapper is used by higher-level features such as automation or navigation commands, inspect the caller's assumptions too.

## Verification

- Run targeted mapper tests for pathfinding or map-shape changes.
- If the change affects radius or route behavior, verify at least one real caller path in addition to package tests.
- Call out any edge cases not exercised, such as disconnected graphs or temporary exits.

## Documentation

- Keep this file about graph and pathfinding guardrails, not a mapping feature reference.
