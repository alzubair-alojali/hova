# Hova

A local-first SSH address book for people who forget server addresses and for agents that need explicit, scriptable access. The user selected the name Hova and authorized implementation and publication to a public GitHub repository.

## Experience

`hova` opens a full-screen terminal workspace: mint wordmark, server count, search field, selection list, connection detail panel, restrained borders, keyboard hints. Arrow keys navigate; enter connects; `/` searches; `a` adds; `e` edits; `d` opens a deletion confirmation. Guided forms collect name, host, user, port, identity path, tags, and notes. Empty states explain the next action. Narrow terminals use one column. Respect NO_COLOR and never put styling in JSON.

## Commands

`list`, `show`, `add`, `edit`, `remove`, `connect`, `exec`, `import ssh`, `config path`, and shell completion. CRUD and import support machine-readable JSON; mutations are noninteractive with explicit arguments. Removal requires --yes outside the UI. `exec NAME -- COMMAND` runs the supplied remote shell command explicitly; agents get BatchMode, strict host-key checking and a finite timeout. `connect --dry-run` exposes executable plus argument vector without executing anything. Native SSH owns authentication, host keys and SSH config. Never build a local shell command from server fields.

## Data and security

Versioned JSON under the OS user config directory, overridable by HOVA_CONFIG or --config. Private metadata stays out of the source repository. Store key paths only, never passwords or key contents. Validate all records on read and write, reject control characters and option-like hosts, enforce unique case-insensitive server names and ports 1–65535. Atomic replacement and an exclusive writer lock prevent corruption and lost concurrent updates. Missing files mean an empty inventory; invalid files produce actionable errors and are never overwritten.

SSH config import enumerates concrete aliases from a chosen file (no wildcard/negated hosts). Keep aliases and a source-config reference so native SSH resolves Include, Match, jump hosts and identities at connection time. Do not evaluate config commands during import. Duplicate aliases are skipped, never overwritten. Describe inventory addresses as configured addresses, not reachability status.

## Implementation choices

Builder review resolutions: `exec` accepts exactly one remote shell command string after `--`, avoiding ambiguous argument joining. Default execution timeout is 30 seconds; JSON execution returns stdout, stderr, exit_code, and timed_out while preserving SSH exit status. Imported aliases have separate alias/config fields; manual overrides are passed only when explicitly configured. Omitted edit flags preserve fields, empty optional flags clear them. Non-TTY bare invocation prints help. SSH handoff uses Bubble Tea ExecProcess and resumes search and selection. Locks wait at most two seconds; a stale lock produces manual recovery guidance. Forms retain values on validation errors and Escape cancels.

Builder choice: Go 1.24+, Cobra, Bubble Tea v1, Bubbles text inputs, Lip Gloss. One portable executable on Windows/macOS/Linux with OpenSSH installed. Tests use temp inventories and dry-run command plans, never real user servers. CI tests all three operating systems and builds release archives with checksums. MIT license. README uses original SVG graphics, real documented commands, and clearly illustrative server addresses.

## Boundaries

No credential vault, cloud account integration, background monitoring, sync, or autonomous server changes. Reading inventory never connects. Remote execution is opt-in and output goes directly to the caller; --json wraps remote output when requested. An optional exec timeout kills the local SSH process, not necessarily a remote job.
