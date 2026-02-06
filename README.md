# hy

Command-line interface for Hypewell Studio.

## Installation

### Homebrew (macOS)

```bash
brew tap hypewell-ai/tap
brew install hy
```

### Binary Download

Download from [Releases](https://github.com/hypewell-ai/hy/releases).

### From Source

```bash
go install github.com/hypewell-ai/hy@latest
```

## Quick Start

```bash
# Authenticate
hy auth login

# List productions
hy productions list

# Create a production
hy productions create --name "My Video" --topic "Product Launch"

# Trigger a build
hy productions build prod_abc123

# Upload assets
hy assets upload video.mp4

# Manage API keys
hy keys create --name "CI Integration"
hy keys list
```

## Commands

```
hy auth login       Authenticate with Hypewell Studio
hy auth logout      Clear stored credentials
hy auth status      Show current auth status

hy productions list     List all productions
hy productions create   Create a new production
hy productions get      Get production details
hy productions build    Trigger a build
hy productions delete   Delete a production

hy assets list      List assets
hy assets upload    Upload an asset
hy assets delete    Delete an asset

hy keys create      Create an API key
hy keys list        List API keys
hy keys revoke      Revoke an API key

hy thread           Start interactive chat
hy thread send      Send a single message

hy config set       Set configuration
hy config get       Get configuration
hy version          Show version
```

## Configuration

Config is stored at `~/.config/hy/config.yaml`:

```yaml
api_url: https://studio.hypewell.ai/api
workspace_id: ws_abc123
```

Credentials are stored in the system keychain.

## Environment Variables

| Variable | Description |
|----------|-------------|
| `HY_API_KEY` | API key (overrides stored auth) |
| `HY_API_URL` | API base URL |
| `HY_WORKSPACE` | Workspace ID |

## Development

```bash
# Build
go build -o hy .

# Run tests
go test ./...

# Install locally
go install .
```

## License

Proprietary. © 2026 Hypewell AI.
