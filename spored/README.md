# spore-os

`spored` is the central IPC hub daemon of the Spore OS ecosystem. It runs as a dedicated system user via launchd (macOS), systemd (Linux), or the Windows SCM.

For installation and service management, see the [installation guide](https://github.com/sporeos-dev/spore-os/wiki).
For the wire protocol, see the [Spore Protocol SPEC](https://github.com/sporeos-dev/spore-protocol).

## Development

```bash
cd spored

# Build
go build -o spored .

# Stop → copy → start (macOS)
sudo launchctl bootout system /Library/LaunchDaemons/dev.sporeos.spored.plist
sudo cp spored /usr/local/bin/spored
sudo cp spored.manifest.spore.yaml "/Library/Application Support/spore-os/hub/"
sudo launchctl bootstrap system /Library/LaunchDaemons/dev.sporeos.spored.plist
```

Run interactively without touching system directories:

```bash
SPORE_DATA_DIR=/tmp/spore-dev go run .
```

## Testing

Tests are SPEC-driven — they verify protocol behavior rather than implementation details.

```bash
cd spored
go test ./... -count=1
```

| Package | SPEC Sections | What's tested |
|---------|--------------|---------------|
| `message/` | §6 Wire Grammar | Message dispatch (Cast/Capture/Spore/Error/Cancelled by prefix), `~handle` extraction |
| `message/` (cast) | §6.1, §6.2, §6.3, §6.6 | Cast parsing, `cast=` injection, handle requirement, args, flags, `json` flag |
| `message/` (capture) | §6.1, §6.2 | Capture parsing, `ok`/`capture=` injection, `~handle:subject` self-describing format |
| `message/` (error) | §6.5, §6.6 | Standard/custom errors, error origin flags, SporeFailure blocked from nodes, all standard error codes |
| `message/` (cancelled) | §6.1 | Cancelled parsing, `capture=` injection, `NewCancelled` hub-generated cancelled |
| `message/` (spore) | §2.4, §10 | SPORE.* command parsing, args, flags, destination fixed to hub |
| `manifest/` | §2.4, §6.6, §8.1–8.4 | SPORE.* namespace rejection, reserved keyword rejection, metadata loading, API/error entries, `required` field parsing |
| `utilities/` | §8.2 | `app:` path resolution — relative, `~/`, absolute, empty, `n/a` pass-through |
| `router/` | §5.2, §5.3, §6.2 | Route registration/lookup, command listing by node, handle tracking, short-form routing |
| `hub/` | §4 | Handshake protocol (send ID → receive OK), whitespace trim, connection failure handling |
| `registry/` | §8.2 | `app:` resolved to absolute path in stored manifest copy; absolute and `n/a` preserved |
| `spore/` | §10 | Hub command execution: help, node.list/install/uninstall/help, command.list/help, error.list/help, response format validation |
