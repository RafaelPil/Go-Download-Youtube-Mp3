package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"github.com/kkdai/youtube/v2"
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
	client := youtube.Client{}

	// Get video info
	video, err := client.GetVideo(videoURL)
	if err != nil {
		return "", fmt.Errorf("getting video info: %w", err)
	}

	// Find the best audio format
	formats := video.Formats.WithAudioChannels()
	if len(formats) == 0 {
		return "", fmt.Errorf("no audio formats available")
	}

	// Select the best quality audio format
	bestFormat := formats[0]
	for _, format := range formats {
		if format.Bitrate > bestFormat.Bitrate {
			bestFormat = format
		}
	}

	// Create output file paths
	videoPath := filepath.Join(outputDir, fmt.Sprintf("%s.mp4", video.ID))
	mp3Path := filepath.Join(outputDir, fmt.Sprintf("%s.mp3", video.ID))

	// Download the video
	if err := downloadVideo(&client, video, &bestFormat, videoPath); err != nil {
		return "", fmt.Errorf("downloading video: %w", err)
	}

	// Convert to MP3
	if err := convertToMP3(videoPath, mp3Path); err != nil {
		cleanupFiles(videoPath)
		return "", fmt.Errorf("converting to MP3: %w", err)
	}

	// Remove the temporary video file
	cleanupFiles(videoPath)

	return mp3Path, nil
}

// Download the video stream
func downloadVideo(client *youtube.Client, video *youtube.Video, format *youtube.Format, outputPath string) error {
    maxRetries := 3
    var lastError error

    for i := 0; i < maxRetries; i++ {
        // Try both GetStream and GetStreamURL approaches
        var reader io.ReadCloser
        var err error

        // First try GetStream
        reader, _, err = client.GetStream(video, format)
        if err != nil {
            // If GetStream fails, try GetStreamURL
            var streamURL string
            streamURL, err = client.GetStreamURL(video, format)
            if err != nil {
                lastError = fmt.Errorf("both GetStream and GetStreamURL failed: %w", err)
                time.Sleep(time.Second * time.Duration(i+1))
                continue
            }

            // Download using HTTP client
            resp, err := http.Get(streamURL)
            if err != nil {
                lastError = fmt.Errorf("http.Get failed: %w", err)
                time.Sleep(time.Second * time.Duration(i+1))
                continue
            }
            reader = resp.Body
        }

        // Create output file
        file, err := os.Create(outputPath)
        if err != nil {
            reader.Close()
            return fmt.Errorf("creating file: %w", err)
        }

        // Copy data
        _, err = io.Copy(file, reader)
        reader.Close()
        file.Close()

        if err != nil {
            os.Remove(outputPath)
            lastError = fmt.Errorf("copying data: %w", err)
            time.Sleep(time.Second * time.Duration(i+1))
            continue
        }

        return nil
    }

    return fmt.Errorf("after %d attempts: %w", maxRetries, lastError)
}
// Convert video to MP3 using FFmpeg
func convertToMP3(inputPath, outputPath string) error {
	// Check if input file exists
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return fmt.Errorf("input file not found")
	}

	// Prepare FFmpeg command
	cmd := exec.Command(
		"ffmpeg",
		"-i", inputPath,       // Input file
		"-vn",                 // No video
		"-c:a", "libmp3lame", // MP3 codec
		"-q:a", "2",          // Quality (2 = ~190 kbps VBR)
		"-y",                 // Overwrite output
		outputPath,           // Output file
	)

	// Capture output for debugging
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	// Run the command with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("ffmpeg error: %w", err)
		}
	case <-time.After(5 * time.Minute):
		return fmt.Errorf("conversion timed out after 5 minutes")
	}

	// Verify output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return fmt.Errorf("output file not created")
	}

	return nil
}