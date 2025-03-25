package main

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file found")
	}

	// Get the bot token from the environment
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is not set in the environment")
	}

	// Initialize Telegram bot
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = true
	log.Printf("Authorized as %s", bot.Self.UserName)

	// Configure updates
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	// Create a temporary directory for downloads
	tmpDir := "downloads"
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		log.Panicf("Failed to create downloads directory: %v", err)
	}

	for update := range updates {
		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID
		messageText := update.Message.Text

		if messageText != "" && isYouTubeLink(messageText) {
			// Acknowledge the request
			sendMessage(bot, chatID, "⏳ Downloading your video and converting to MP3. Please wait...")

			// Process the YouTube URL
			videoURL := extractYouTubeURL(messageText)
			if videoURL == "" {
				sendMessage(bot, chatID, "❌ Invalid YouTube URL")
				continue
			}

			// Download and process the video
			mp3Path, err := downloadAndConvert(videoURL, tmpDir)
			if err != nil {
				sendMessage(bot, chatID, fmt.Sprintf("❌ Error: %v", err))
				continue
			}

			// Check file size before sending (Telegram has 50MB limit for bots)
			fileInfo, err := os.Stat(mp3Path)
			if err != nil {
				sendMessage(bot, chatID, "❌ Error checking file size")
				cleanupFiles(mp3Path)
				continue
			}

			if fileInfo.Size() > 50*1024*1024 { // 50MB
				sendMessage(bot, chatID, "❌ File is too large (max 50MB)")
				cleanupFiles(mp3Path)
				continue
			}

			// Send the MP3 file
			if err := sendAudioFile(bot, chatID, mp3Path); err != nil {
				sendMessage(bot, chatID, fmt.Sprintf("❌ Error sending file: %v", err))
			}

			// Clean up
			cleanupFiles(mp3Path)
		} else {
			sendMessage(bot, chatID, "📝 Please send me a valid YouTube link and I'll convert it to MP3 for you!")
		}
	}
}

// Helper function to send messages
func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

// Helper function to send audio files
func sendAudioFile(bot *tgbotapi.BotAPI, chatID int64, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("opening audio file: %w", err)
	}
	defer file.Close()

	fileReader := tgbotapi.FileReader{
		Name:   filepath.Base(filePath),
		Reader: file,
	}

	audio := tgbotapi.NewAudio(chatID, fileReader)
	audio.Title = strings.TrimSuffix(filepath.Base(filePath), ".mp3")
	audio.Performer = "YouTube Downloader"

	_, err = bot.Send(audio)
	return err
}

// Clean up temporary files
func cleanupFiles(paths ...string) {
	for _, path := range paths {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			log.Printf("Error removing file %s: %v", path, err)
		}
	}
}

// Extract the clean YouTube URL
func extractYouTubeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	// Handle youtu.be links
	if strings.Contains(u.Host, "youtu.be") {
		return "https://www.youtube.com/watch?v=" + strings.TrimPrefix(u.Path, "/")
	}

	// Standard YouTube URL
	if strings.Contains(u.Host, "youtube.com") {
		return "https://www.youtube.com/watch?v=" + u.Query().Get("v")
	}

	return ""
}

// Check if a string contains a YouTube link
func isYouTubeLink(url string) bool {
	return strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be")
}

// Download and convert YouTube video to MP3
func downloadAndConvert(videoURL, outputDir string) (string, error) {
    // Generate safe filename
    videoID := ""
    if strings.Contains(videoURL, "v=") {
        videoID = strings.Split(strings.Split(videoURL, "v=")[1], "&")[0]
    } else if strings.Contains(videoURL, "youtu.be/") {
        videoID = strings.Split(strings.Split(videoURL, "youtu.be/")[1], "?")[0]
    }
    
    if videoID == "" {
        return "", fmt.Errorf("could not extract video ID from URL")
    }

    mp3Path := filepath.Join(outputDir, fmt.Sprintf("%s.mp3", videoID))

    // yt-dlp command with better error handling
	cmd := exec.Command("/usr/local/bin/yt-dlp",  // Use absolute path
        "-x",
        "--audio-format", "mp3",
        "--audio-quality", "0",
        "-o", mp3Path,
        "--force-ipv4",
        "--geo-bypass",
        "--retries", "10",
        "--fragment-retries", "10",
        "--throttled-rate", "100K",
        "--no-warnings",
        videoURL,
    )

    // Capture output for debugging
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    // Run with timeout
    err := cmd.Run()
    if err != nil {
        return "", fmt.Errorf("yt-dlp error: %v\nStdout: %s\nStderr: %s", 
            err, stdout.String(), stderr.String())
    }

    // Verify output
    if _, err := os.Stat(mp3Path); os.IsNotExist(err) {
        return "", fmt.Errorf("output file not created")
    }

    return mp3Path, nil
}