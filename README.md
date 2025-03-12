# tg2dynalist

A Telegram bot that forwards messages to Dynalist inbox.

## Features

- Forwards text messages from Telegram to Dynalist inbox
- Authenticates users based on Telegram user ID
- Simple deployment with Docker

## Prerequisites

- Telegram Bot Token (from [@BotFather](https://t.me/BotFather))
- Dynalist API Token (from [Dynalist Settings](https://dynalist.io/developer))
- Your Telegram User ID (you can get it from [@userinfobot](https://t.me/userinfobot))

## Environment Variables

The bot requires the following environment variables:

- `BOT_TOKEN`: Your Telegram bot token
- `DYNALIST_TOKEN`: Your Dynalist API token
- `TG_USER_ID`: Your Telegram user ID (as a number)

## Running Locally

### Using Go

```bash
# Set environment variables
export BOT_TOKEN=your_telegram_bot_token
export DYNALIST_TOKEN=your_dynalist_token
export TG_USER_ID=your_telegram_user_id

# Run the bot
go run main.go
```

### Using Docker

```bash
docker run -e BOT_TOKEN=your_telegram_bot_token \
           -e DYNALIST_TOKEN=your_dynalist_token \
           -e TG_USER_ID=your_telegram_user_id \
           ghcr.io/korjavin/tg2dynalist:latest
```

## Building from Source

```bash
# Clone the repository
git clone https://github.com/korjavin/tg2dynalist.git
cd tg2dynalist

# Build the binary
go build -o tg2dynalist

# Run the binary
./tg2dynalist
```

## Docker Build

```bash
docker build -t tg2dynalist .
```

## Usage

1. Start a chat with your bot on Telegram
2. Send any text message
3. The bot will forward the message to your Dynalist inbox
4. The bot will reply with a confirmation message

## License

MIT