# Security

Report a vulnerability privately through [GitHub private vulnerability reporting](https://github.com/alzubair-alojali/hova/security/advisories/new). Do not include real server inventories, keys, or credentials in public issues.

## Trust model

- Hova saves connection metadata locally. It never stores passwords or private-key contents. Key file paths, server addresses, usernames, tags and notes are still sensitive metadata.
- Files are created with mode `0600` and directories with `0700` on Unix. Windows access depends on the parent directory's ACLs. The inventory is not encrypted.
- Native OpenSSH handles keys, agents, host-key verification and SSH config. Hova never disables host-key verification. Interactive connections follow your SSH settings; `exec` explicitly enables BatchMode and strict host-key checking.
- SSH config is executable configuration: `ProxyCommand`, `LocalCommand` and `Match exec` can run local programs when OpenSSH connects. Only import and use SSH config you trust. Import itself reads text without evaluating those commands.
- `hova exec` runs the exact remote shell command you supply. Hova does not interpret it locally. Review agent-generated commands before allowing them access to your servers.
- Execution timeout terminates the local SSH client. It does not guarantee that a remote process is terminated.
- JSON remote output is buffered in memory; use streamed execution without `--json` for large or unbounded output.
- Inventory mutations use an exclusive lock and temporary-file replacement. If a process crashes while writing, a `.lock` file may remain. Confirm no Hova writer is running before removing that lock.

The initial `0.1.x` release line receives security fixes. Release archives include checksums but are not signed or notarized.
