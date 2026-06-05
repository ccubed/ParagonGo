# GMCP Module Guide

## Scope

- Use this file for the `modules/gmcp` protocol module, including GMCP payload dispatch, connection capability tracking, and web-client text-prefix handling.
- This module bridges server events to telnet and web client protocol behavior.

## Working Rules

- Preserve existing namespace contracts unless the task explicitly changes a GMCP payload shape.
- Be careful with telnet negotiation versus WebSocket text-prefix behavior; those paths are related but not identical.
- If a change affects client capability tracking or Mudlet-specific behavior, verify the caller assumptions in web client or term code too.
- Prefer additive namespace changes over silently repurposing an existing payload.
- `GMCPCharModule_Payload_Inventory_Worn` is a `map[string]GMCPCharModule_Payload_Inventory_Item` keyed by slot name. It is populated by `buildWornPayload`, which iterates `items.AllEquipSlots()`. Adding or removing a slot in `internal/items/itemspec.go` is sufficient; no manual update to this module is needed.

## Verification

- Run targeted module/package tests for GMCP behavior changes.
- Verify the exact namespace or negotiation path changed, not just generic module load behavior.
- Call out client compatibility risk when changing payload shapes or negotiation rules.

## Documentation

- Keep this file about protocol compatibility and integration guardrails.
