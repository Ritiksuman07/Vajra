---
name: Documentation issue
description: Report missing, wrong or unclear docs
title: "[docs]: "
labels: ["documentation"]
body:
  - type: input
    id: page
    attributes:
      label: Which page or file?
      placeholder: "README.md / docs/CLI.md"
    validations:
      required: true
  - type: textarea
    id: problem
    attributes:
      label: What's wrong or missing?
    validations:
      required: true
  - type: textarea
    id: suggestion
    attributes:
      label: Suggested wording
    validations:
      required: false
