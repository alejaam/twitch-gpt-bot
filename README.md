# twitch-gpt-bot

A small Go service that lets a Twitch bot fetch a concise NVIDIA NIM response through `GET /gpt`. It uses only the Go standard HTTP library plus `godotenv` for local configuration.

## Quick path

1. Copy `.env.example` to `.env`.
2. In [NVIDIA Build](https://build.nvidia.com/), choose a model available to your API key and copy its model ID into `NVIDIA_MODEL`.
3. Fill in `NVIDIA_API_KEY` and `API_SHARED_SECRET` locally or in Dokploy; never commit either value.
4. Start locally with `docker compose up --build`.
5. Confirm `curl http://localhost:8080/health` returns `ok`.

## Endpoints

### `GET /gpt`

Required query parameters:

- `query`: the viewer's prompt.
- `user`: the Twitch username used for the in-memory cooldown.
- `secret`: must match `API_SHARED_SECRET`.

The endpoint always returns `text/plain`:

- `200`: generated response, normalized to one line and capped at 450 characters.
- `400`: malformed request.
- `401`: invalid shared secret.
- `429`: that user is still in the cooldown window.
- `502`: NVIDIA could not provide a response; a safe fallback is returned.

Example:

```bash
curl --get 'http://localhost:8080/gpt' \
  --data-urlencode 'query=Explain dependency injection in one sentence' \
  --data-urlencode 'user=viewer_name' \
  --data-urlencode 'secret=replace-with-your-secret'
```

### `GET /health`

Returns `200 ok` for Dokploy health checks. It does not call NVIDIA.

## Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `PORT` | `8080` | HTTP listen port. |
| `NVIDIA_API_KEY` | required | NVIDIA Build/NIM credential. Never commit it. |
| `NVIDIA_MODEL` | required | Exact model ID enabled for the configured NVIDIA key. |
| `NVIDIA_MAX_TOKENS` | `120` | Upper bound for generated tokens per call. |
| `NVIDIA_TEMPERATURE` | `0.7` | Generation randomness, from `0` to `2`. |
| `SYSTEM_PROMPT` | concise chat assistant | System instruction sent with every request. |
| `RATE_LIMIT_SECONDS` | `8` | Per-user in-memory cooldown. |
| `API_SHARED_SECRET` | required | Shared secret accepted by `/gpt`. |

Generate the shared secret outside source control:

```bash
openssl rand -hex 32
```

## Cost and abuse controls

The initial implementation deliberately limits scope to a per-user cooldown and a low NVIDIA output-token cap. There is no global daily quota, persistent storage, moderator bypass, or external dependency.

This is suitable for one container. A restart resets cooldown state, and multiple replicas would not share limits. Keep the service at one replica until an explicit shared-limiter design is introduced.

Use HTTPS in front of the service. Since the first bot integration sends the shared secret in a URL, also configure any reverse proxy or access logs to redact query strings where possible.

## Twitch bot example

For a Nightbot-style command, configure the external URL in the platform UI using its URL encoding helpers. The concrete syntax differs by provider; the request must URL-encode the viewer question and include the viewer username plus the shared secret. Do not expose the raw secret in public chat messages.

## Local development

```bash
cp .env.example .env
# edit .env
go test ./...
go run ./cmd
```

Docker is the supported local path when Go is not installed on the host:

```bash
docker build -t twitch-gpt-bot .
docker run --rm --env-file .env -p 8080:8080 twitch-gpt-bot
```

## Dokploy deployment

Build from this repository with the `Dockerfile`, add the variables from `.env.example` in Dokploy's secret/environment UI, and configure the health check path as `/health`. Do not upload a `.env` file containing production credentials.

Use `NVIDIA_MODEL` exactly as NVIDIA Build presents it for your key. Available models and free-tier limits are account-dependent, so the application intentionally does not guess a default model.

## Future: Twitch EventSub WebSocket

A direct EventSub integration would replace third-party `urlfetch` bots with a WebSocket consumer that verifies Twitch events, owns command parsing, and can obtain authoritative broadcaster/moderator roles. It is intentionally not implemented here because it introduces Twitch OAuth, subscription lifecycle management, reconnect handling, and a different operational model.
