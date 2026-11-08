# Contributing to Hova

Small, focused pull requests are welcome. Open an issue first for a substantial feature so we can agree on its behavior.

## Development

Use Go 1.24 or newer and OpenSSH. Clone this repository, then run:

```sh
go mod download
go test ./...
go vet ./...
go build -o bin/hova ./cmd/hova
```

On Windows, build with `go build -o bin/hova.exe ./cmd/hova`.

Use `--config` to point development runs at a disposable inventory. Never commit your real inventory, SSH config, private keys, hostnames, or credentials. Documentation examples use reserved addresses.

## Before a pull request

- Add a behavior test for a bug fix or a new command contract.
- Run `go fmt ./...`, `go vet ./...`, and `go test ./...`.
- Check terminal changes at wide and narrow sizes, with an empty inventory and with `NO_COLOR` set.
- Update the README when a command or its behavior changes.

The inventory, SSH command builder, CLI, and TUI live in separate packages under `internal/`. Keep connections explicit and let OpenSSH handle authentication. A feature that reads metadata must not initiate a connection.

## Releases

Maintainers push a semantic version tag such as `v0.1.0`. The release workflow tests the code and uses `.goreleaser.yaml` to publish archives for Windows, macOS, and Linux, plus SHA-256 checksums. Release binaries are not code-signed.
