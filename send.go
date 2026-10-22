package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// SafeSend sends a plain message, logging instead of panicking on error.
func SafeSend(s *discordgo.Session, channelID, content string) {
	if content == "" {
		log.Printf("WARNING: Attempted to send empty message to channel %s", channelID)
		return
	}

	logContent := content
	if len(logContent) > 100 {
		logContent = logContent[:100] + "..."
	}
	log.Printf("Sending message to channel %s: %s", channelID, logContent)

	if _, err := s.ChannelMessageSend(channelID, content); err != nil {
		log.Printf("Error sending message to channel %s: %v", channelID, err)
	}
}

// SafeSendEmbed sends an embed, logging instead of panicking on error.
func SafeSendEmbed(s *discordgo.Session, channelID string, embed *discordgo.MessageEmbed) {
	if embed == nil {
		log.Printf("WARNING: Attempted to send nil embed to channel %s", channelID)
		return
	}

	embedTitle := "No title"
	if embed.Title != "" {
		embedTitle = embed.Title
	}
	log.Printf("Sending embed to channel %s: %s", channelID, embedTitle)

	if _, err := s.ChannelMessageSendEmbed(channelID, embed); err != nil {
		log.Printf("Embed send error in channel %s: %v", channelID, err)
	}
}

// startTyping runs a typing indicator until the returned stop function is
// called. The stop function is safe to call more than once.
func startTyping(s *discordgo.Session, channelID string) func() {
	done := make(chan struct{})
	var once sync.Once
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if err := s.ChannelTyping(channelID); err != nil {
					log.Printf("Error sending typing indicator: %v", err)
				}
			}
		}
	}()
	return func() { once.Do(func() { close(done) }) }
}

// sendLongMessage delivers content that may exceed Discord's 2000-char limit,
// splitting it (with rate-limit-aware delays) or attaching a .txt file.
func sendLongMessage(s *discordgo.Session, channelID, content, prefix string) {
	const discordLimit = 2000
	const txtFileThreshold = 3 * discordLimit

	if len(content) <= discordLimit {
		if prefix != "" {
			SafeSend(s, channelID, fmt.Sprintf("%s\n%s", prefix, content))
		} else {
			SafeSend(s, channelID, fmt.Sprintf("Ate Luningning says:\n%s", content))
		}
		return
	}

	if len(content) <= txtFileThreshold {
		time.Sleep(200 * time.Millisecond)
		if prefix != "" {
			SafeSend(s, channelID, fmt.Sprintf("%s (This is a long response, splitting it up!)", prefix))
		} else {
			SafeSend(s, channelID, "Ate Luningning's response (splitting into parts)")
		}

		const partPrefixOverhead = 15
		maxContentPerPart := discordLimit - partPrefixOverhead
		totalParts := (len(content) + maxContentPerPart - 1) / maxContentPerPart

		for partNum := 1; partNum <= totalParts; partNum++ {
			if partNum > 1 {
				time.Sleep(800 * time.Millisecond)
			}
			start := (partNum - 1) * maxContentPerPart
			end := start + maxContentPerPart
			if end > len(content) {
				end = len(content)
			}
			partText := fmt.Sprintf("[Part %d/%d]\n%s", partNum, totalParts, content[start:end])
			if len(partText) > discordLimit {
				partText = partText[:discordLimit]
			}
			SafeSend(s, channelID, partText)
		}
		return
	}

	// Very long: attach as a .txt file.
	tmpFile, err := os.CreateTemp("", "ateluningning_*.txt")
	if err != nil {
		log.Printf("Error creating temp file for long message: %v", err)
		sendLongMessage(s, channelID, content, prefix+" (Couldn't create file, splitting instead)")
		return
	}
	defer os.Remove(tmpFile.Name())

	if _, err = tmpFile.WriteString(content); err != nil {
		log.Printf("Error writing to temp file: %v", err)
		tmpFile.Close()
		sendLongMessage(s, channelID, content, prefix+" (Couldn't write to file, splitting instead)")
		return
	}
	tmpFile.Close()

	fileBytes, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		log.Printf("Error reading temp file: %v", err)
		sendLongMessage(s, channelID, content, prefix+" (Couldn't read file, splitting instead)")
		return
	}

	message := fmt.Sprintf("%s (This response is very long, so I've attached it as a file!)", prefix)
	if message == " (This response is very long, so I've attached it as a file!)" {
		message = "Ate Luningning's response is very long, so I've attached it as a file!"
	}
	SafeSend(s, channelID, message)

	log.Printf("Sending file attachment to channel %s: ateluningning_response.txt (%d bytes)", channelID, len(fileBytes))
	if _, err = s.ChannelFileSend(channelID, "ateluningning_response.txt", bytes.NewReader(fileBytes)); err != nil {
		log.Printf("Error sending file: %v", err)
		SafeSend(s, channelID, "Oh dear! I couldn't send the file. Let me try splitting the message instead...")
		sendLongMessage(s, channelID, content, "Here's the response (split into parts):")
		return
	}
	log.Printf("File sent successfully to channel %s", channelID)
}

// processAttachment downloads a .txt attachment and returns its text.
func processAttachment(attachment *discordgo.MessageAttachment) (string, error) {
	filename := strings.ToLower(attachment.Filename)
	if !strings.HasSuffix(filename, ".txt") {
		return "", fmt.Errorf("only .txt files are supported, got: %s", attachment.Filename)
	}

	log.Printf("Processing attachment: %s (URL: %s, Size: %d bytes)",
		attachment.Filename, attachment.URL, attachment.Size)

	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest("GET", attachment.URL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("User-Agent", "DiscordBot (https://github.com/bwmarrin/discordgo, v0.27.1)")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download attachment: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download attachment: HTTP %d", resp.StatusCode)
	}

	const maxSize = int64(10 * 1024 * 1024)
	content, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
	if err != nil {
		return "", fmt.Errorf("failed to read attachment content: %v", err)
	}

	textContent := strings.TrimSpace(string(content))
	log.Printf("Successfully processed attachment %s: %d characters", attachment.Filename, len(textContent))

	if len(textContent) > 10000 {
		textContent = textContent[:10000] + "\n\n[File truncated - too large for processing]"
	}
	return textContent, nil
}
