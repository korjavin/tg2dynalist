package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	dynalistAPIURL = "https://dynalist.io/api/v1/inbox/add"
)

// DynalistRequest represents the request body for the Dynalist API
type DynalistRequest struct {
	Token    string `json:"token"`
	Index    int    `json:"index,omitempty"`
	Content  string `json:"content"`
	Note     string `json:"note,omitempty"`
	Checked  bool   `json:"checked,omitempty"`
	Checkbox bool   `json:"checkbox,omitempty"`
}

// DynalistResponse represents the response from the Dynalist API
type DynalistResponse struct {
	Code    string `json:"_code"`
	Message string `json:"_msg,omitempty"`
	FileID  string `json:"file_id,omitempty"`
	NodeID  string `json:"node_id,omitempty"`
	Index   int    `json:"index,omitempty"`
}

func main() {
	// Get environment variables
	botToken := os.Getenv("BOT_TOKEN")
	dynalistToken := os.Getenv("DYNALIST_TOKEN")
	tgUserIDStr := os.Getenv("TG_USER_ID")

	// Validate environment variables
	if botToken == "" || dynalistToken == "" || tgUserIDStr == "" {
		log.Fatal("BOT_TOKEN, DYNALIST_TOKEN, and TG_USER_ID environment variables must be set")
	}

	// Convert TG_USER_ID to int64
	tgUserID, err := strconv.ParseInt(tgUserIDStr, 10, 64)
	if err != nil {
		log.Fatalf("Invalid TG_USER_ID: %v", err)
	}

	// Initialize Telegram bot
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatalf("Failed to initialize Telegram bot: %v", err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Set up update configuration
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	// Start receiving updates
	updates := bot.GetUpdatesChan(updateConfig)

	// Process updates
	for update := range updates {
		// Check if we have a message
		if update.Message == nil {
			continue
		}

		// Check if the message is from the authorized user
		if update.Message.From.ID != tgUserID {
			log.Printf("Unauthorized message from user ID: %d", update.Message.From.ID)
			continue
		}

		// Check if message contains media
		if update.Message.Photo != nil || update.Message.Video != nil ||
			update.Message.Audio != nil || update.Message.Document != nil ||
			update.Message.Sticker != nil || update.Message.Animation != nil {
			// Send message that only text is supported
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Only text messages are supported. Media cannot be uploaded to Dynalist.")
			msg.ReplyToMessageID = update.Message.MessageID
			bot.Send(msg)
			continue
		}

		// Get the message text
		messageText := update.Message.Text
		if messageText == "" {
			// Skip empty messages
			continue
		}

		// Forward the message to Dynalist
		err := addToDynalist(dynalistToken, messageText)
		if err != nil {
			log.Printf("Failed to add message to Dynalist: %v", err)

			// Send error message back to the user
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Error adding to Dynalist: %v", err))
			msg.ReplyToMessageID = update.Message.MessageID
			bot.Send(msg)
			continue
		}

		// Send confirmation message
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Added to Dynalist inbox")
		msg.ReplyToMessageID = update.Message.MessageID
		bot.Send(msg)
	}
}

// addToDynalist sends a message to the Dynalist inbox
func addToDynalist(token, content string) error {
	// Create request body
	reqBody := DynalistRequest{
		Token:   token,
		Content: content,
	}

	// Marshal request body to JSON
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", dynalistAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var dynalistResp DynalistResponse
	if err := json.NewDecoder(resp.Body).Decode(&dynalistResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Check response code
	if dynalistResp.Code != "Ok" {
		if dynalistResp.Message != "" {
			return fmt.Errorf("dynalist API error: %s", dynalistResp.Message)
		}
		return fmt.Errorf("dynalist API error: code %s", dynalistResp.Code)
	}

	return nil
}
