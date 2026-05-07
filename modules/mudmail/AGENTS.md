# Mudmail Module Guide

## Scope

- Use this file for inbox persistence, admin mudmail APIs/UI, and cross-module inbox delivery in `modules/mudmail`.
- This module touches persistence, admin web flows, and player inventory/gold side effects.

## Working Rules

- Preserve inbox storage compatibility unless the task explicitly includes a migration.
- Be careful with offline delivery and legacy inbox migration. Those paths are easy to break without immediate local symptoms.
- Keep admin API behavior, admin UI behavior, and inbox side effects aligned when changing message create/read/delete flow.
- Other modules should continue using the exported inbox function rather than bypassing the module's storage path.

## Verification

- Run targeted module tests for inbox persistence or migration changes.
- If the change affects admin APIs or UI, verify the exact admin route path involved.
- If reading a message changes player gold or item state, verify that side effect explicitly.

## Documentation

- Keep only persistence and integration guardrails here.
