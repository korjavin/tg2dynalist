package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

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

	// Initialize Cloudflare R2 client if environment variables are set
	var r2Client *CloudflareR2Client
	if os.Getenv("CF_ACCOUNT_ID") != "" {
		r2Client, err = NewCloudflareR2Client()
		if err != nil {
			log.Printf("Warning: Failed to initialize Cloudflare R2 client: %v", err)
			log.Printf("Media uploads will be disabled")
		} else {
			log.Printf("Cloudflare R2 client initialized successfully")
		}
	} else {
		log.Printf("Cloudflare R2 environment variables not set, media uploads will be disabled")
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

		// Process the message
		processMessage(bot, update.Message, dynalistToken, r2Client)
	}
}

// processMessage handles a Telegram message and adds it to Dynalist
func processMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message, dynalistToken string, r2Client *CloudflareR2Client) {
	// Get the message text
	messageText := message.Text
	var note string
	var mediaURL string
	var err error

	// Check if the message is forwarded
	if message.ForwardFrom != nil || message.ForwardFromChat != nil {
		var forwardInfo string
		if message.ForwardFrom != nil {
			// Forwarded from user
			forwardInfo = fmt.Sprintf("Forwarded from: %s %s (@%s)",
				message.ForwardFrom.FirstName,
				message.ForwardFrom.LastName,
				message.ForwardFrom.UserName)
		} else if message.ForwardFromChat != nil {
			// Forwarded from channel or group
			forwardInfo = fmt.Sprintf("Forwarded from: %s", message.ForwardFromChat.Title)
			if message.ForwardFromChat.UserName != "" {
				forwardInfo += fmt.Sprintf(" (@%s)", message.ForwardFromChat.UserName)
			}
		}

		if messageText == "" {
			messageText = forwardInfo
		} else {
			note = forwardInfo
		}
	}

	// Check if message contains media we want to process (photos)
	if message.Photo != nil && r2Client != nil {
		// Get the largest photo
		photoSize := message.Photo[len(message.Photo)-1]

		// Get file URL
		fileURL, err := bot.GetFileDirectURL(photoSize.FileID)
		if err != nil {
			log.Printf("Failed to get photo URL: %v", err)
		} else {
			// Download the file
			fileData, err := DownloadFileFromTelegram(fileURL)
			if err != nil {
				log.Printf("Failed to download photo: %v", err)
			} else {
				// Upload to Cloudflare R2
				mediaURL, err = r2Client.UploadFile(fileData, ".jpg")
				if err != nil {
					log.Printf("Failed to upload photo to R2: %v", err)
				} else {
					log.Printf("Photo uploaded to R2: %s", mediaURL)

					// Add the media URL to the note
					if note != "" {
						note += "\n\n"
					}
					pathURL := mediaURL[strings.LastIndex(strings.TrimSuffix(mediaURL, "/details"), "/")+1:]
					note += fmt.Sprintf("Image: %s", r2Client.GetDashboardURL(pathURL))
				}
			}
		}
	}

	// Skip if no text and no media URL
	if messageText == "" && mediaURL == "" {
		// Send message that we need at least text
		msg := tgbotapi.NewMessage(message.Chat.ID, "Cannot add empty message to Dynalist.")
		msg.ReplyToMessageID = message.MessageID
		bot.Send(msg)
		return
	}

	// If we have a media URL but no text, use a placeholder
	if messageText == "" && mediaURL != "" {
		messageText = "Image from Telegram"
	}

	// Add caption to message text if available
	if message.Caption != "" {
		if messageText != "" {
			messageText += "\n\n"
		}
		messageText += message.Caption
	}

	// Forward the message to Dynalist
	err = AddToDynalist(dynalistToken, messageText, note)
	if err != nil {
		log.Printf("Failed to add message to Dynalist: %v", err)

		// Send error message back to the user
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Error adding to Dynalist: %v", err))
		msg.ReplyToMessageID = message.MessageID
		bot.Send(msg)
		return
	}

	// Prepare confirmation message
	confirmationText := "Added to Dynalist inbox"
	if mediaURL != "" {
		confirmationText = "Added to Dynalist inbox with image"
	}

	// Send confirmation message
	msg := tgbotapi.NewMessage(message.Chat.ID, confirmationText)
	msg.ReplyToMessageID = message.MessageID
	bot.Send(msg)
}
