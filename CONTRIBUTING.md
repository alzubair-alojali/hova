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

For the npm launcher, use Node.js 22 or newer. Run `npm test` for launcher behavior. `npm run build:npm` cross-compiles all six binaries into the ignored `native/` directory; maintainers need Go on PATH or an absolute compiler path in `HOVA_GO`. Then run `npm run test:package` to inspect the real tarball and install it into a temporary prefix with lifecycle scripts disabled. No npm dependencies need installing for these tests.

The package has an explicit files allowlist. Keep credentials, server inventories, tests and Go source out of the npm tarball. It deliberately bundles all six binaries in one download, avoiding optional dependencies and install scripts.

Maintainers update `package.json` and `package-lock.json`, then push a matching semantic version tag such as `v0.1.1`. The release workflow tests the code, publishes native archives with `.goreleaser.yaml`, and attaches the npm tarball and its checksum. Release binaries are not code-signed.

GitHub tarballs can be installed directly through npm. Public packages must be packed by the Linux release workflow, which verifies Unix executable permissions. A tarball packed on Windows is suitable for local Windows testing but loses Unix executable modes; do not publish it.

For npm registry publication, log in to the authorized npm account, download the tested release tarball, verify it against `npm-checksums.txt`, and run `npm publish ./hova-cli-VERSION.tgz --access public` on that original archive. Complete any npm-requested two-factor verification. Publishing the release archive preserves its executable modes even when the publisher uses Windows. The registry package name is `hova-cli`; the installed command is `hova`. Do not commit npm tokens or registry configuration containing credentials.
