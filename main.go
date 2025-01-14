package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Config struct to load settings
type Config struct {
	TelegramBotToken        string  `json:"telegramBotToken"`
	CheckURL                string  `json:"checkURL"`
	MonitorUsers            []int64 `json:"monitorUsers"`
	PrivilegedUsersSublist         []int64 `json:"privilegedUsersSublist"`
	CheckIntervalSeconds    int     `json:"checkIntervalSeconds"`
	SecondCheckDelaySeconds int     `json:"secondCheckDelaySeconds"`
	ScriptWaitTimeSeconds   int     `json:"scriptWaitTimeSeconds"`
	RequestTimeoutSeconds   int     `json:"requestTimeoutSeconds"`
	ScriptPath              string  `json:"scriptPath"`
}

var (
	config            Config
	bot               *tgbotapi.BotAPI
	lastCheckWasError bool // track if the last check indicated an error
	privilegedUsersSublist   map[int64]bool
)

// ======= MAIN =======
func main() {
	// Load configuration
	err := loadConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Convert PrivilegedUsersSublist to a map for efficient lookup
	privilegedUsersSublist = make(map[int64]bool)
	for _, id := range config.PrivilegedUsersSublist {
		privilegedUsersSublist[id] = true
	}

	// Create a new Telegram Bot instance
	bot, err = tgbotapi.NewBotAPI(config.TelegramBotToken)
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
	ticker := time.NewTicker(time.Duration(config.CheckIntervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C
		checkAndNotify()
	}
}

// loadConfig loads settings from a JSON file
func loadConfig(filePath string) error {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("could not read config file: %w", err)
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return fmt.Errorf("could not parse config file: %w", err)
	}

	return nil
}

// handleMessage processes incoming Telegram messages/commands
func handleMessage(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	userID := msg.From.ID

	// We only care about messages from the subset that can send "оживити"
	if msg.Text == "оживити" && privilegedUsersSublist[userID] {
		log.Printf("Received 'оживити' command from user %d", userID)

		// Execute the script
		err := runScript(config.ScriptPath)
		if err != nil {
			sendMessage(chatID, fmt.Sprintf("Помилка запуску скрипту: %v", err))
			return
		}

		// Wait the specified time
		time.Sleep(time.Duration(config.ScriptWaitTimeSeconds) * time.Second)

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
		time.Sleep(time.Duration(config.SecondCheckDelaySeconds) * time.Second)

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

// checkService tries to connect to the specified URL with a timeout, ignoring TLS errors.
func checkService() bool {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.RequestTimeoutSeconds)*time.Second)
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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, config.CheckURL, nil)
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
	for _, userID := range config.MonitorUsers {
		sendMessage(userID, text)
	}
}
