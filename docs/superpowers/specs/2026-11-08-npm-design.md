# npm installation

The user approved npm installation with the existing Go core. Package `hova` is owned by an unrelated project; use `hova-cli` (no registry entry found on 2026-11-08), exposing command `hova`.

Ship one package containing all six prebuilt executables: Windows, macOS and Linux, each x64 and arm64. This deliberately trades a larger download for no optional platform dependencies, lifecycle scripts, runtime downloads, or compiler requirements. Node 22 or newer is the runtime floor. OpenSSH remains required for connections only.

A dependency-free CommonJS launcher resolves the matching bundled binary, inherits terminal stdin/stdout/stderr and the environment, forwards literal arguments with no shell, preserves exit codes, and forwards termination signals. Missing/unsupported binaries produce actionable errors; --json errors stay machine-readable on stderr.

A build script cross-compiles the six native binaries with the package version, using Go only for maintainers and CI. npm's explicit files allowlist excludes source, tests, developer data and credentials. Publish a tested npm tarball on each GitHub release as well as npm when authenticated; GitHub tarball installation provides a usable npm route even when npm publishing credentials are unavailable.

No account login is required for users installing a public package. Publishing requires the owner's npm login and any npm-requested two-factor verification. Never expose or commit npm credentials. Keep standalone releases available.
