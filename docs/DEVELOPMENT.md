[← spore-os](../README.md)

# Development

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
