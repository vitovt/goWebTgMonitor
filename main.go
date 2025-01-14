package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ======= CONFIGURATION =======

// Replace with your actual Bot Token from @BotFather
const telegramBotToken = "YOUR_TELEGRAM_BOT_TOKEN"

// The HTTPS URL to check (domain/ip + port).
// Example: "https://123.45.67.89:443" or "https://mydomain:8443"
const checkURL = "https://YOUR_DOMAIN_OR_IP:YOUR_PORT/"

// Allowed user IDs for receiving downtime notifications.
var allowedUsers = []int64{
	12345678, // replace with real Telegram user IDs
	87654321,
}

// A subset of allowedUsers that can send "оживити" command.
var approvedSublist = map[int64]bool{
	12345678: true, // only some from allowedUsers can actually send "оживити"
	// add more if needed
}

// Service check intervals
const checkInterval = 60 * time.Second // how often to check service (e.g., every 60s)
const secondCheckDelay = 30 * time.Second
const scriptWaitTime = 40 * time.Second
const requestTimeout = 30 * time.Second

// ======= GLOBAL FLAGS =======
var (
	bot               *tgbotapi.BotAPI
	lastCheckWasError bool // track if the last check indicated an error
)

// ======= MAIN =======
func main() {
	var err error

	// Create a new Telegram Bot instance
	bot, err = tgbotapi.NewBotAPI(telegramBotToken)
	if err != nil {
		log.Fatalf("Error creating bot: %v", err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Start receiving updates
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	// Goroutine to handle Telegram updates (commands, messages, etc.)
	go func() {
		for update := range updates {
			// If it has a message
			if update.Message != nil {
				handleMessage(update.Message)
			}
		}
	}()

	// Periodically check service availability
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		<-ticker.C
		checkAndNotify()
	}
}

// handleMessage processes incoming Telegram messages/commands
func handleMessage(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	userID := msg.From.ID

	// We only care about messages from the subset that can send "оживити"
	if msg.Text == "оживити" && approvedSublist[userID] {
		log.Printf("Received 'оживити' command from user %d", userID)

		// Execute the script
		err := runScript("/root/main_script.sh")
		if err != nil {
			sendMessage(chatID, fmt.Sprintf("Помилка запуску скрипту: %v", err))
			return
		}

		// Wait the specified time
		time.Sleep(scriptWaitTime)

		// Check service again
		ok := checkService()
		if ok {
			// If OK, send success message to entire list1
			broadcastMessage("Сервер було оживлено. Все в порядку!")
			// Reset the error flag
			lastCheckWasError = false
		} else {
			// If not OK, continue normal cycle (do nothing special here)
			sendMessage(chatID, "Сервіс все ще недоступний, продовжую перевірку.")
		}
	}
}

// checkAndNotify checks the service and sends notifications if it is down
func checkAndNotify() {
	// First check
	ok := checkService()
	if !ok {
		log.Println("Service check failed. Retrying in 30 seconds...")
		time.Sleep(secondCheckDelay)

		// Second check
		ok2 := checkService()
		if !ok2 {
			log.Println("Service is confirmed down on second check.")
			// If it wasn't already in error state, notify
			if !lastCheckWasError {
				broadcastMessage("Увага! Сервіс недоступний на двох поспіль перевірках.")
			}
			lastCheckWasError = true
			return
		}
	}

	// If service is up, reset the error flag
	lastCheckWasError = false
}

// checkService tries to connect to the specified URL with a 30-second timeout,
// ignoring TLS certificate errors (e.g. self-signed).
func checkService() bool {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	// Custom HTTP client that skips TLS verification
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // ignore self-signed certificate
		},
	}
	client := &http.Client{
		Transport: tr,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checkURL, nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error checking service: %v", err)
		return false
	}
	defer resp.Body.Close()

	// Consider the service 'up' if we get a 2xx or 3xx status code
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return true
	}

	log.Printf("Service responded with status code %d", resp.StatusCode)
	return false
}

// runScript executes an external script on the local machine
func runScript(scriptPath string) error {
	cmd := exec.Command("/bin/bash", scriptPath)
	return cmd.Run()
}

// sendMessage sends a message to a single chat
func sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("Error sending message to chat %d: %v", chatID, err)
	}
}

// broadcastMessage sends a message to all allowed users
func broadcastMessage(text string) {
	for _, userID := range allowedUsers {
		sendMessage(userID, text)
	}
}
