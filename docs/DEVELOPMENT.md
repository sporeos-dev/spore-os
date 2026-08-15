[← spore-os](../README.md)

# Development

## Dependencies

`spored` uses CGo to link against the shared parser library from
[spore-client-libs](https://github.com/sporeos-dev/spore-client-libs).
The pre-built static library must be present at `../spore-client-libs/dist/`
(i.e. the two repositories are expected to be siblings under the same parent
directory, e.g. `$DEV/spore-os` and `$DEV/spore-client-libs`).

On macOS the library ships as a universal fat binary covering both arm64 and
x86_64. A C compiler (Xcode Command Line Tools) is required to build.

## Building

```bash
cd spored
go build -o spored .
```

Run without touching system directories:

```bash
SPORE_DATA_DIR=/tmp/spore-dev go run .
```

## Testing

```bash
cd spored
go test ./... -count=1
```

## Deploying (macOS)

```bash
sudo launchctl bootout system /Library/LaunchDaemons/dev.sporeos.spored.plist
sudo cp spored /usr/local/bin/spored
sudo cp spored.manifest.spore.yaml "/Library/Application Support/spore-os/hub/"
sudo launchctl bootstrap system /Library/LaunchDaemons/dev.sporeos.spored.plist
```
