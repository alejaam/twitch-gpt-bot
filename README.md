# twitch-gpt-bot

A Go service that connects directly to Twitch Chat (via TLS IRC) to answer viewer commands (e.g. `!gpt <prompt>`) using **Google Gemini** or **NVIDIA NIM**.

Designed for single-container deployment (Dokploy, Docker, or standalone binary) behind NAT without requiring a public IP, domain, or port-forwarding.

---

## Highlights

- **Outbound TLS connection:** Connects directly to `irc.chat.twitch.tv:6697`. No incoming ports or webhooks needed.
- **Provider Choice:** Supports **Google Gemini** (`gemini-2.5-flash`, `gemini-1.5-flash`, etc.) and **NVIDIA NIM** (`LLM_PROVIDER=gemini` or `LLM_PROVIDER=nvidia`).
- **Twitch-optimized:** Automatically mentions the requesting viewer (`@user`), strips extra whitespace into a single line, and respects Twitch's 450 character limit.
- **Abuse Controls:** In-memory per-user cooldown (`RATE_LIMIT_SECONDS`) to prevent chat spam.
- **Dokploy friendly:** Includes an optional `/health` HTTP endpoint on `PORT` for container health monitoring.

---

## Quick Start

1. Copy `.env.example` to `.env`:
   ```bash
   cp .env.example .env
   ```
2. Generate a Twitch OAuth token for your bot account:
   - Use a tool like [twitchtokengenerator.com](https://twitchtokengenerator.com/) or Twitch CLI.
   - Required scopes: `chat:read` and `chat:edit`.
3. Configure `.env`:
   ```env
   TWITCH_BOT_USERNAME=tu_bot_o_usuario
   TWITCH_OAUTH_TOKEN=oauth:tu_token_aqui
   TWITCH_CHANNEL=tu_canal
   COMMAND_PREFIX=!gpt

   LLM_PROVIDER=gemini
   GEMINI_API_KEY=tu_google_api_key
   GEMINI_MODEL=gemini-2.5-flash
   ```
4. Run tests:
   ```bash
   go test ./...
   ```
5. Run the bot:
   ```bash
   go run ./cmd
   ```

---

## Configuration Reference

| Variable | Default | Purpose |
| --- | --- | --- |
| `TWITCH_BOT_USERNAME` | *required* | Twitch username of the bot account. |
| `TWITCH_OAUTH_TOKEN` | *required* | OAuth token (`oauth:xxxx`) with `chat:read` + `chat:edit`. |
| `TWITCH_CHANNEL` | *required* | Target Twitch channel name (with or without `#`). |
| `COMMAND_PREFIX` | `!gpt` | Chat trigger prefix. |
| `RATE_LIMIT_SECONDS` | `8` | Cooldown period per viewer in seconds. |
| `LLM_PROVIDER` | `gemini` | LLM engine: `gemini` or `nvidia`. |
| `GEMINI_API_KEY` | required if gemini | Google AI Studio / Gemini API key. |
| `GEMINI_MODEL` | `gemini-2.5-flash` | Gemini model name. |
| `GEMINI_MAX_TOKENS` | `150` | Maximum tokens generated. |
| `GEMINI_TEMPERATURE` | `0.7` | Temperature (0.0 - 2.0). |
| `NVIDIA_API_KEY` | required if nvidia | NVIDIA Build NIM API key. |
| `NVIDIA_MODEL` | required if nvidia | Exact NVIDIA model ID. |
| `NVIDIA_MAX_TOKENS` | `120` | Maximum tokens generated. |
| `NVIDIA_TEMPERATURE` | `0.7` | Temperature (0.0 - 2.0). |
| `SYSTEM_PROMPT` | concise assistant | System instruction sent with prompts. |
| `PORT` | `8080` | Internal HTTP port for Dokploy health checks. |

---

## Deployment with Dokploy

1. Create a new Application in Dokploy pointed to your repository.
2. In the **Environment** tab, add your production environment variables (from `.env.example`).
3. Set the Health Check path to `/health` on port `8080`.
4. Deploy. The container connects outbound to Twitch chat and listens for `!gpt`.

*Tip:* In your Twitch channel, type `/mod <TWITCH_BOT_USERNAME>` so Twitch does not apply message rate limits to the bot.
