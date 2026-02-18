# make-it-public-tgbot

Telegram Bot for Make It Public Service - provides a convenient Telegram interface for managing Make It Public API tokens.

## Architecture

The telegram bot service consists of two main components deployed as a Docker Swarm stack:

- **mitbot**: The Telegram bot application that handles user interactions
- **redis**: Dedicated Redis instance for storing user data and conversations

### Service Communication

```
┌─────────────────────┐
│  Telegram Users     │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐      ┌──────────────────────┐
│      mitbot         │─────▶│  make-it-public API  │
│  (Telegram Bot)     │      │  (port 8082 on host) │
└──────────┬──────────┘      └──────────────────────┘
           │
           ▼
┌─────────────────────┐
│   redis (internal)  │
│  User data storage  │
└─────────────────────┘
```

**Network Configuration**:
- The bot runs in its own isolated overlay network (`mitbot-network`)
- Connects to the Make It Public API via exposed host port 8082
- Uses dedicated Redis instance (not shared with make-it-public service)

## Deployment

### Prerequisites

- Docker Swarm initialized on the target host
- GitHub Actions secrets configured in the `production` environment
- Make It Public service already deployed and accessible on port 8082

### Environment Variables

#### Non-Secret Variables
- `VERSION` - Docker image tag (automatically set from git tag)
- `NETWORK_NAME` - Overlay network name (default: `mitbot-network`)
- `MIT_DEFAULT_TTL` - Default token TTL in seconds (default: 604800 = 7 days)
- `LOG_LEVEL` - Logging level (default: `info`)

#### Secret Variables (GitHub Secrets)
- `BOT_TOKEN` - Telegram bot token from [@BotFather](https://t.me/botfather)
- `MIT_URL` - Make It Public API URL (e.g., `http://167.172.190.133:8082`)
- `HOST` - Deployment server hostname/IP
- `USERNAME` - SSH username for deployment
- `PORT` - SSH port for deployment
- `SSH_KEY` - SSH private key for deployment

### Deployment Process

The deployment is fully automated via GitHub Actions:

1. **On Pull Request**: Build and test the Docker image (amd64 only)
2. **On Tag Push** (e.g., `v1.0.0`):
   - Build multi-arch Docker images (amd64, arm64)
   - Push images to GitHub Container Registry
   - Deploy to production using the cloudlab workflow
   - Automatic rollback on failure

**To deploy a new version**:
```bash
git tag v1.0.0
git push origin v1.0.0
```

**Manual deployment** (if needed):
```bash
# SSH to the server
ssh -p 1923 deployer@167.172.190.133

# Navigate to deployment directory
cd ~/cloudlab/stacks/mitbot

# Pull latest images
docker stack deploy -c docker-compose.yml mitbot
```

### Monitoring

**Check service status**:
```bash
docker stack ps mitbot
```

**View service logs**:
```bash
# Bot logs
docker service logs -f mitbot_mitbot

# Redis logs
docker service logs -f mitbot_redis
```

**Health check**:
```bash
# Check if services are running
docker service ls | grep mitbot
```

### Rollback

If deployment fails, the system automatically rolls back to the previous version. For manual rollback:

```bash
# Find previous version
docker service inspect mitbot_mitbot --format '{{.PreviousSpec.TaskTemplate.ContainerSpec.Image}}'

# Manually rollback
docker service update --rollback mitbot_mitbot
```

## Configuration

The bot uses viper for configuration management, supporting both environment variables and config files.

**Environment variable mapping**:
- `BOT_TOKEN` → `bot.token`
- `MIT_URL` → `mit.url`
- `MIT_DEFAULT_TTL` → `mit.default_ttl`
- `REPO_REDIS_ADDR` → `repo.redis_addr`
- `REPO_KEY_PREFIX` → `repo.key_prefix`
- `LOG_LEVEL` → logging level

## Development

### Local Development

1. **Start dependencies** (Redis):
   ```bash
   docker run -d -p 6379:6379 redis:7.4-alpine
   ```

2. **Set environment variables**:
   ```bash
   export BOT_TOKEN="your-telegram-bot-token"
   export MIT_URL="http://localhost:8082"
   export REPO_REDIS_ADDR="localhost:6379"
   export REPO_KEY_PREFIX="MITTGBOT::"
   ```

3. **Run the bot**:
   ```bash
   go run cmd/mitbot/main.go run
   ```

### Building Docker Image

```bash
docker build -t ghcr.io/ksysoev/make-it-public-tgbot:dev .
```

### Testing

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...
```

## Bot Commands

- `/start` - Start interaction with the bot
- `/token` - Generate a new API token
- `/revoke` - Revoke an existing token
- `/list` - List all your active tokens
- `/help` - Show help message

## Project Structure

```
.
├── cmd/
│   └── mitbot/          # Main application entry point
├── pkg/
│   ├── bot/             # Telegram bot logic and handlers
│   ├── cmd/             # CLI commands and configuration
│   ├── core/            # Core business logic
│   ├── prov/            # External providers (MIT API client)
│   └── repo/            # Data repositories (Redis)
├── deploy/
│   └── docker-compose.yml  # Production deployment configuration
├── .github/
│   └── workflows/       # CI/CD workflows
├── Dockerfile           # Multi-stage Docker build
└── docker-compose.yml   # Development docker-compose (dev only)
```

## Troubleshooting

### Bot not responding

1. Check if services are running:
   ```bash
   docker service ls | grep mitbot
   ```

2. Check logs for errors:
   ```bash
   docker service logs mitbot_mitbot
   ```

3. Verify bot token is correct:
   ```bash
   docker service inspect mitbot_mitbot --format '{{range .Spec.TaskTemplate.ContainerSpec.Env}}{{println .}}{{end}}' | grep BOT_TOKEN
   ```

### Cannot connect to MIT API

1. Verify make-it-public service is running and port 8082 is accessible:
   ```bash
   curl http://localhost:8082/health
   ```

2. Check MIT_URL environment variable:
   ```bash
   docker service inspect mitbot_mitbot --format '{{range .Spec.TaskTemplate.ContainerSpec.Env}}{{println .}}{{end}}' | grep MIT_URL
   ```

### Redis connection issues

1. Check if Redis is running:
   ```bash
   docker service ps mitbot_redis
   ```

2. Test Redis connectivity:
   ```bash
   docker exec $(docker ps -q -f name=mitbot_redis) redis-cli ping
   ```

## License

See LICENSE file for details.
