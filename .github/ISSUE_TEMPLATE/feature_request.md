---
name: Feature request
description: Suggest an idea or improvement
title: "[feat]: "
labels: ["enhancement"]
body:
  - type: markdown
    attributes:
      value: |
        Before filing, please check the [roadmap](../blob/main/docs/TODO.md) —
        it may already be planned.
  - type: textarea
    id: problem
    attributes:
      label: What problem does this solve?
      description: Describe the need, not just the solution.
    validations:
      required: true
  - type: textarea
    id: proposal
    attributes:
      label: Proposed solution
    validations:
      required: true
  - type: textarea
    id: alternatives
    attributes:
      label: Alternatives considered
    validations:
      required: false
  - type: dropdown
    id: area
    attributes:
      label: Area
      options:
        - Terminal UI (TUI)
        - Web GUI
        - REST API
        - Agents / swarm mode
        - Verification
        - Models / providers
        - Packaging / install
        - Docs
        - Other
    validations:
      required: true
  - type: checkboxes
    id: constraints
    attributes:
      label: Constraints
      options:
        - label: This keeps Vajra fully functional offline
        - label: This does not add telemetry
