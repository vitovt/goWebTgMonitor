# goWebTgMonitor

## Overview
This is a Go Telegram bot designed to monitor the availability of an HTTPS service and send notifications when the service is down. The bot periodically checks the service, and if it fails two consecutive checks, it alerts a list of monitoring users on Telegram. Certain privileged users can also send a command to run a local script to “revive” the service, after which the bot checks again and notifies the group if the service is back online.

## Features
- Periodic HTTPS checks with configurable intervals.
- Ignores self-signed certificates (TLS verification disabled).
- Notifies a list of telegram users if the service is down twice in a row.
- Allows a subset of privileged users to execute a local script via the bot.
- Slash-command support (`/start`, `/help`, `/ozhyvyty`) and inline buttons for easy usage.
- All settings (URL, tokens, user IDs, intervals) are stored in a `config.json` file. Example is here [config.json.default](config.json.default)

## Requirements
- Go 1.18+ (or compatible version).
- A valid Telegram Bot Token (create a bot via [@BotFather](https://t.me/BotFather)).
- Access to the machine on which the script should be executed.
- (Optional) `sudo` privileges set up, if you need to run the script as root while the bot runs as a non-root user.

## Installation

1. **Compile the bot**
   ```bash
   git clone https://github.com/vitovt/goWebTgMonitor.git
   cd goWebTgMonitor
   make lin-dep-ubuntu #if needed
   make build
   ```
or `make build-docker-linux` if you want cpecific system GLIBC compatibility, for example debian-bullseye

or just download binary from release page if it is compatible with your GLIBC.

2. **Configure the Bot**  
   Create a `config.json` in the project directory with the required fields:
   ```sh
   cp config.json.default config.json
   ```
   - **telegramBotToken**: Bot token from BotFather.
   - **checkURL**: The HTTPS endpoint to check (IP/domain + port).
   - **monitorUsers**: Telegram user IDs to be notified when the service goes down.
   - **privilegedUsersSublist**: Subset of user IDs who can execute the “revive” command.
   - **checkIntervalSeconds**: How often (in seconds) to run the first service check.
   - **secondCheckDelaySeconds**: Delay (in seconds) before the second check if the first check fails.
   - **scriptWaitTimeSeconds**: How long to wait (in seconds) after running the script before re-checking the service.
   - **requestTimeoutSeconds**: HTTP request timeout (in seconds) for each service check.
   - **scriptPath**: Path to the local script that will be run when a privileged user sends the `/оживити` command.

3. **Run the Bot**  
   ```bash
   ./bin/goWebTgMonitor-1.0.1_linux_amd64
   ```
   The bot starts polling for updates, and you can control it via Telegram.

## Usage
1. **Start the Bot**  
   Send `/start` in a Telegram chat with the bot. You’ll receive a greeting, plus a help message, and the bot will display inline buttons.

2. **Help**  
   Send `/help` or click the “/help” button to see a brief list of commands:
   - `/start` – Greeting and help
   - `/help` – Show list of commands
   - `/оживити` – Execute the specified script (only for privileged users)

3. **Revive Command**  
   - Only users listed in `privilegedUsersSublist` can run `/ozhyvyty`.  
   - When `/ozhyvyty` is issued, the bot executes the script at `scriptPath`, waits, then re-checks the service. If it’s back up, a success message is sent to all `monitorUsers`.

4. **Notifications**  
   If the service fails two consecutive checks, the bot notifies all `monitorUsers`. It continues to check periodically until the service is restored.

## Security
- If you want the bot to run as a non-root user but still execute a script with `root` privileges, you can configure `sudo` rules (e.g., `visudo`) to allow password-less `sudo` for the script path.
- Keep your `telegramBotToken` private. Do not commit it to a public repository unless using environment variables or other secure methods.

## Contributing
1. Fork this repository.
2. Create a feature branch.
3. Commit your changes.
4. Push to your fork and open a pull request.

## License
This repository is licensed under the MIT License. See the [LICENSE](LICENSE) file for more information.
Feel free to modify and distribute them.
**© 2024 Vitovt ©**
