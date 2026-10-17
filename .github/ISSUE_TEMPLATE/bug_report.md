---
name: Bug report
description: Something isn't working as expected
title: "[bug]: "
labels: ["bug"]
body:
  - type: markdown
    attributes:
      value: |
        Thanks for the report! Please run `vajra doctor` and paste its output —
        it answers most questions up front.
  - type: textarea
    id: what-happened
    attributes:
      label: What happened?
      description: What did you expect, and what actually happened?
    validations:
      required: true
  - type: textarea
    id: reproduce
    attributes:
      label: Steps to reproduce
      placeholder: |
        1. run `vajra ...`
        2. ...
        3. see error
    validations:
      required: true
  - type: input
    id: version
    attributes:
      label: Vajra version
      description: Output of `vajra version`
      placeholder: "vajra 1.2.0"
    validations:
      required: true
  - type: input
    id: model
    attributes:
      label: Model
      placeholder: "llama3.1:8b"
  - type: dropdown
    id: os
    attributes:
      label: Operating system
      options:
        - Windows
        - Linux
        - macOS
        - Other
    validations:
      required: true
  - type: textarea
    id: doctor
    attributes:
      label: "`vajra doctor` output"
      render: shell
    validations:
      required: false
  - type: textarea
    id: logs
    attributes:
      label: Logs / screenshots
      render: shell
    validations:
      required: false
