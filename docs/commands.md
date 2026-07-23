# AEJ Commands

This file covers the most common `aej` commands, with quick usage instructions and minimal use cases.

## Authentication

### `aej login`
- Use case: configure the Jira URL, email address, and API token.
- Example:
  ```bash
  aej login
  ```

## Search and tracking

### `aej me`
- Use case: view information about the current user.
- Example:
  ```bash
  aej me
  ```

### `aej mine`
- Use case: list the latest issues assigned to you.
- Example:
  ```bash
  aej mine
  aej mine --status "In Progress"
  ```

### `aej board`
- Use case: list available boards or view the issues on a board.
- Example:
  ```bash
  aej board
  aej board 1712
  aej board 1712 --full
  ```

### `aej search [TERM]`
- Use case: search for issues by text, tag, or version.
- Example:
  ```bash
  aej search "login bug"
  aej search --tag backend
  aej search --version 2.1
  ```

### `aej issue <KEY>`
- Use case: view the details of a specific issue.
- Example:
  ```bash
  aej issue DEV-123
  ```

### `aej logs`
- Use case: view the time worked during a given period.
- Example:
  ```bash
  aej logs
  aej logs --days 15
  aej logs --date 16-07-2026
  ```

## Issue actions

### `aej assign <KEY>`
- Use case: assign an issue to yourself or another user, or remove the assignee.
- Example:
  ```bash
  aej assign DEV-123
  aej assign DEV-123 --to user@company.com
  aej assign DEV-123 --unassign
  ```

### `aej transition <KEY>`
- Use case: change an issue's status through an interactive selection.
- Example:
  ```bash
  aej transition DEV-123
  ```

### `aej comment <KEY> <COMMENT>`
- Use case: add a comment to an issue.
- Example:
  ```bash
  aej comment DEV-123 "Fix available for validation"
  ```

### `aej open <KEY>`
- Use case: open an issue directly in the browser.
- Example:
  ```bash
  aej open DEV-123
  ```

### `aej log <KEY> <TIME> [COMMENT]`
- Use case: log time spent on an issue.
- Example:
  ```bash
  aej log DEV-123 2h
  aej log DEV-123 30m "Code review"
  aej log DEV-123 "1h 30m" "Implementing feature"
  ```

## Global flags

- `--json`: returns the output as JSON.
- `--timing`: shows the total execution time.

Example:

```bash
aej issue DEV-123 --json
aej search "deploy" --timing
```
