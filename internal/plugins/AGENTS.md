# Plugins Package Guide

## Scope

- Use this file for plugin registration, plugin file/data integration, plugin web hooks, and cross-module exported function behavior in `internal/plugins`.
- This package defines the module contract used across `modules/`.

## Working Rules

- Preserve plugin API stability unless the task explicitly changes module contracts.
- Be careful with registration order, exported functions, file overlays, and admin/web registration helpers. Changes here can break many modules at once.
- Keep module-facing helpers generic. Do not smuggle one module's policy into the shared plugin layer.
- When adjusting plugin web or file behavior, verify both the plugin API and at least one real module caller.

## Verification

- Run targeted package tests for plugin contract or file-integration changes.
- Use a module-level validation path if the change affects registration, exported functions, or web integration.
- Call out module compatibility risk when changing shared plugin behavior.

## Documentation

- Keep this file about module-contract guardrails, not a full plugin API guide.
