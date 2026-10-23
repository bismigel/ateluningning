package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// BibleVerse is the structured response from the Bible API.
type BibleVerse struct {
	Translation map[string]interface{} `json:"translation"`
	RandomVerse map[string]interface{} `json:"random_verse"`
}

// fetchBibleVerse retrieves a random verse.
func fetchBibleVerse(debug bool) (*BibleVerse, error) {
	client := &http.Client{Timeout: RequestTimeout}

	if debug {
		log.Printf("DEBUG: Fetching Bible verse from: %s", BibleAPIURL)
	}

	resp, err := client.Get(BibleAPIURL)
	if err != nil {
		return nil, fmt.Errorf("bible verse API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("bible verse API returned status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024))
	if err != nil {
		return nil, fmt.Errorf("error reading API response: %w", err)
	}

	var verse BibleVerse
	if err := json.Unmarshal(body, &verse); err != nil {
		return nil, fmt.Errorf("failed to parse verse data: %w", err)
	}
	return &verse, nil
}

// createVerseEmbed renders a verse as a Discord embed.
func createVerseEmbed(verse *BibleVerse) *discordgo.MessageEmbed {
	var builder strings.Builder

	builder.WriteString("**Translation Details:**\n")
	for key, value := range verse.Translation {
		builder.WriteString(fmt.Sprintf("- %s: %v\n", key, value))
	}

	builder.WriteString("\n**Random Verse:**\n")
	for key, value := range verse.RandomVerse {
		builder.WriteString(fmt.Sprintf("- %s: %v\n", key, value))
	}

	return &discordgo.MessageEmbed{
		Title:       "Daily Bible Verse 📖",
		Description: builder.String(),
		Color:       0x3498db,
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}
