# Claude Usage Limit Checker

A Go application that monitors your Claude usage via direct API calls and sends Discord notifications when usage exceeds configured thresholds.

## Features

- 🚀 **Direct API Access**: Uses Claude's internal API endpoint for fast, reliable usage checks
- 📊 **Real-time Monitoring**: Periodically checks your Claude usage (5-hour and 7-day periods)
- 🔔 **Discord Notifications**: Sends alerts via Discord webhook when thresholds are exceeded
- ⚙️ **Configurable Thresholds**: Set multiple warning levels (default: 80%, 90%, 95%)
- 🎨 **Color-coded Alerts**: Different colors for different warning levels
- ⏰ **Smart Cooldown**: Prevents notification spam with built-in cooldown periods
- 🪶 **Lightweight**: No browser automation - just simple HTTP requests

## Prerequisites

- Go 1.21 or higher
- Discord webhook URL
- Claude account with active session

## Installation

1. Clone the repository:
```bash
git clone https://github.com/CY-Chang-tw/ClaudeUsageLimitChecker.git
cd ClaudeUsageLimitChecker
```

2. Install dependencies:
```bash
go mod download
```

3. Get your Claude credentials:

   **a. Get your Session Key:**
   - Open https://claude.ai/settings/usage in your browser
   - Open DevTools (F12) → Network tab
   - Refresh the page
   - Click on the `usage` request
   - Go to Headers → Request Headers → Cookie
   - Find and copy the value after `sessionKey=` (starts with `sk-ant-sid01-`)

   **b. Get your Organization ID:**
   - From the same Network request, look at the URL
   - Copy the UUID from `/api/organizations/{your-org-id}/usage`

4. Copy `.env.example` to `.env`:
```bash
cp .env.example .env
```

5. Edit `.env` with your credentials:
```env
CLAUDE_SESSION_KEY=sk-ant-sid01-YOUR_SESSION_KEY_HERE
CLAUDE_ORG_ID=your-organization-id-here
DISCORD_WEBHOOK_URL=your_discord_webhook_url_here
USAGE_THRESHOLD=80
CHECK_INTERVAL=60
WARNING_LEVELS=80,90,95
```

## Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `CLAUDE_SESSION_KEY` | Your Claude session key from cookies | - | ✅ Yes |
| `CLAUDE_ORG_ID` | Your Claude organization ID | - | ✅ Yes |
| `DISCORD_WEBHOOK_URL` | Your Discord webhook URL | - | ✅ Yes |
| `USAGE_THRESHOLD` | Primary usage threshold percentage | 80 | No |
| `CHECK_INTERVAL` | Check interval in minutes | 60 | No |
| `WARNING_LEVELS` | Comma-separated warning percentages (fallback) | 80,90,95 | No |
| `FIVE_HOUR_WARNING_LEVELS` | Thresholds for 5-hour period (optional) | WARNING_LEVELS | No |
| `SEVEN_DAY_WARNING_LEVELS` | Thresholds for 7-day period (optional) | WARNING_LEVELS | No |

### Setting up Discord Webhook

1. Go to your Discord server settings
2. Navigate to Integrations → Webhooks
3. Click "New Webhook"
4. Configure the webhook (name, channel, avatar)
5. Copy the webhook URL
6. Paste it in your `.env` file

## Usage

### Run directly:

```bash
go run main.go
```

### Build and run:

```bash
go build -o ClaudeUsageLimitChecker.exe
./ClaudeUsageLimitChecker.exe
```

### Run as background service (Windows):

#### Option 1: Run now
```bash
./ClaudeUsageLimitChecker.exe
```

#### Option 2: Silent background mode
Double-click `start-silent.vbs` in Windows Explorer

#### Option 3: Auto-start on Windows login
1. Press `Win + R`
2. Type: `shell:startup` and press Enter
3. Copy `start-silent.vbs` to that folder
4. It will start automatically on every login!

### Deploy on Linux Server/VPS (Recommended for 24/7 monitoring):

#### Automated Installation (Easiest)

```bash
# Clone the repository
git clone https://github.com/CY-Chang-tw/ClaudeUsageLimitChecker.git
cd ClaudeUsageLimitChecker

# Run the installation script
chmod +x install.sh
./install.sh
```

The script will:
- ✅ Check and install Go if needed
- ✅ Build the application
- ✅ Create `.env` file and prompt for credentials
- ✅ Create systemd service for auto-start
- ✅ Start the monitor immediately

#### Manual Installation

```bash
# 1. Clone and build
git clone https://github.com/CY-Chang-tw/ClaudeUsageLimitChecker.git
cd ClaudeUsageLimitChecker
go mod download
go build -o ClaudeUsageLimitChecker

# 2. Configure
cp .env.example .env
nano .env  # Add your credentials

# 3. Create systemd service
sudo nano /etc/systemd/system/claude-monitor.service
```

Add this content (replace `your-username` and paths):
```ini
[Unit]
Description=Claude Usage Limit Checker
After=network.target

[Service]
Type=simple
User=your-username
WorkingDirectory=/home/your-username/ClaudeUsageLimitChecker
ExecStart=/home/your-username/ClaudeUsageLimitChecker/ClaudeUsageLimitChecker
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl daemon-reload
sudo systemctl enable claude-monitor
sudo systemctl start claude-monitor
sudo systemctl status claude-monitor
```

#### Linux Service Management

```bash
# View logs
sudo journalctl -u claude-monitor -f

# Restart service
sudo systemctl restart claude-monitor

# Stop service
sudo systemctl stop claude-monitor

# Check status
sudo systemctl status claude-monitor
```

#### Alternative: Run with screen (Simpler, no systemd)

```bash
# Install screen
sudo apt install screen -y

# Start screen session
screen -S claude-monitor
./ClaudeUsageLimitChecker

# Detach: Press Ctrl+A, then D
# Reattach: screen -r claude-monitor
```

## How It Works

1. **API Request**: Makes HTTP GET request to `https://claude.ai/api/organizations/{org-id}/usage`
2. **Authentication**: Uses your session key from cookies for authentication
3. **Data Parsing**: Parses JSON response containing usage data for different time periods
4. **Threshold Checking**: Compares current usage percentage against configured thresholds for each period
5. **Discord Alerts**: Sends color-coded notifications when thresholds are exceeded
6. **Cooldown Period**: Prevents spam with period-specific cooldowns (5-hour: 30min, 7-day: 1hr)

## API Response Format

The Claude API returns usage data in this format:

```json
{
    "five_hour": {
        "utilization": 13.0,
        "resets_at": "2025-11-30T13:59:59.718700+00:00"
    },
    "seven_day": {
        "utilization": 50.0,
        "resets_at": "2025-12-02T06:59:59.718721+00:00"
    }
}
```

The app prioritizes monitoring the 7-day limit over the 5-hour limit.

## Notification Format

Notifications include both usage periods with detailed information:

**📊 Current Session (5-hour)**
- Usage percentage for the current 5-hour window
- Reset time (when the 5-hour period renews)

**📈 Weekly Limits (7-day)**
- Usage percentage for the 7-day period
- Reset time (when the weekly limit renews)

**Color Coding** (based on primary monitored period):
- 🟢 Green: < 80%
- 🟡 Yellow: 80-89%
- 🟠 Orange: 90-94%
- 🔴 Red: ≥ 95%

The notification footer indicates which period is being used as the primary monitor for threshold alerts (prioritizes 7-day over 5-hour).

## Important Notes

⚠️ **Session Key Expiration**:
- Your session key may expire after some time
- If notifications stop working, get a new session key from your browser
- The app will log an error if authentication fails

⚠️ **Security**:
- **NEVER** commit your `.env` file to version control
- Keep your session key and Discord webhook URL private
- Your `.env` file contains sensitive credentials
- The `.gitignore` is already configured to exclude `.env`

⚠️ **Rate Limiting**:
- Default check interval is 60 minutes to avoid rate limiting
- Don't set CHECK_INTERVAL too low (minimum recommended: 30 minutes)

## Troubleshooting

### Authentication fails
- Get a fresh session key from your browser
- Make sure you copied the entire key (starts with `sk-ant-sid01-`)
- Check that you're using the correct organization ID

### Discord notifications not working
- Verify your webhook URL is correct
- Check Discord server permissions
- Test the webhook with: `curl -X POST -H "Content-Type: application/json" -d '{"content":"test"}' YOUR_WEBHOOK_URL`

### No usage data returned
- Verify your organization ID is correct
- Check if you're logged in to Claude in your browser
- Your session might have expired - get a new session key

## Development

### Project Structure
```
.
├── api/
│   └── anthropic.go    # Claude API client (HTTP requests)
├── config/
│   └── config.go       # Configuration management
├── notifier/
│   └── discord.go      # Discord notification service
├── main.go             # Main application entry point
├── start.bat           # Windows startup script
├── start-silent.vbs    # Silent background launcher
├── .env.example        # Environment variables template
└── README.md           # This file
```

### Adding New Features

The codebase is modular and easy to extend:
- `api/anthropic.go`: Modify API requests or add new endpoints
- `notifier/discord.go`: Customize notification format or add new channels
- `main.go`: Adjust monitoring logic or add new threshold types

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - feel free to use this project for personal or commercial purposes.

## Disclaimer

This tool uses Claude's internal API endpoints for monitoring usage. It is intended for personal use only to help you track your own usage. Please respect Anthropic's Terms of Service when using this tool.

## Acknowledgments

Built with ❤️ using:
- [Go](https://golang.org/) - The Go programming language
- [godotenv](https://github.com/joho/godotenv) - Environment variable management
