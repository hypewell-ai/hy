# hy - Hypewell Studio CLI

Command-line interface for Hypewell Studio.

## Installation

### Homebrew (macOS)

```bash
brew tap hypewell-ai/tap
brew install hy
```

### Binary Download

Download the latest release from [GitHub Releases](https://github.com/hypewell-ai/hy/releases).

## Quick Start

```bash
# Authenticate
hy auth login

# List your productions
hy productions list

# Create a new production
hy create --name "Product Launch" --topic "New features announcement"

# Trigger a build
hy build prod_xxx

# Chat with the AI assistant
hy thread chat "How should I structure my video?"
```

## Commands

### Authentication

```bash
hy auth login           # Log in and create API key
hy auth logout          # Log out and revoke key
hy auth status          # Show current auth status
```

### Productions

```bash
hy productions list                     # List all productions
hy productions list --status draft      # Filter by status
hy productions get prod_xxx             # Get production details
hy productions create --name "..." --topic "..."  # Create production
hy productions build prod_xxx           # Trigger build
hy productions status prod_xxx          # Check build status
hy productions delete prod_xxx          # Delete (soft delete)
```

Aliases: `prod`, `p`

### Assets

```bash
hy assets list                    # List all assets
hy assets list --type video       # Filter by type
hy assets upload ./intro.mp4      # Upload a file
hy assets get asset_xxx           # Get asset + download URL
hy assets delete asset_xxx        # Delete asset
```

Aliases: `asset`, `a`

### API Keys

```bash
hy keys list              # List API keys
hy keys create --name "CI"  # Create new key
hy keys revoke key_xxx    # Revoke key
```

### Thread (AI Chat)

```bash
hy thread chat "How should I structure my video?"
hy thread chat -p prod_xxx "Make the hook more engaging"
hy thread chat              # Interactive mode
hy thread history           # View chat history
```

## Configuration

Config file: `~/.config/hy/config.yaml`

```yaml
api_url: https://studio.hypewell.ai/api
workspace_id: ws_xxx
```

API key is stored securely in your system keychain.

## Environment Variables

- `HY_API_KEY` - Override API key
- `HY_API_URL` - Override API URL
- `HY_WORKSPACE_ID` - Override workspace ID

## Build from Source

```bash
git clone https://github.com/hypewell-ai/hy.git
cd hy
go build -o hy .
```

## License

Proprietary - Hypewell AI
