package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Configuration constants.
const (
	Prefix               = "!"
	BibleAPIURL          = "https://bible-api.com/data/web/random"
	DeepSeekAPIHost      = "api.deepseek.com"
	DeepSeekAPIPath      = "/chat/completions"
	DefaultDeepSeekModel = "deepseek-chat"
	RequestTimeout       = 60 * time.Second
	EnvFileName          = ".env"
	DatabaseFile         = "./bot.db"

	AteLuningningAICommand = "askluningning"
	WorkStartCommand       = "chismis"
	SessionTimeout         = 30 * time.Minute
)

// AppConfig holds application-wide configuration.
type AppConfig struct {
	DiscordToken   string
	DeepSeekAPIKey string
	AdminUserIDs   []string
	Debug          bool
}

// loadConfiguration loads and validates application configuration from the
// environment (falling back to .env).
func loadConfiguration() (*AppConfig, error) {
	if err := godotenv.Load(EnvFileName); err != nil {
		return nil, fmt.Errorf("error loading %s file: %w", EnvFileName, err)
	}

	config := &AppConfig{
		DiscordToken:   os.Getenv("DISCORD_BOT_TOKEN"),
		DeepSeekAPIKey: os.Getenv("DEEPSEEK_API_KEY"),
		Debug:          os.Getenv("DEBUG") == "true",
	}

	adminIDsStr := os.Getenv("ADMIN_USER_IDS")
	if adminIDsStr != "" {
		config.AdminUserIDs = strings.Split(adminIDsStr, ",")
		for i, id := range config.AdminUserIDs {
			config.AdminUserIDs[i] = strings.TrimSpace(id)
		}
	} else {
		log.Printf("WARNING: ADMIN_USER_IDS is not set in %s. Lock/unlock commands will not be available.", EnvFileName)
	}

	if config.DiscordToken == "" {
		return nil, fmt.Errorf("DISCORD_BOT_TOKEN is required in %s", EnvFileName)
	}
	if config.DeepSeekAPIKey == "" {
		log.Printf("WARNING: DEEPSEEK_API_KEY is not set in %s. `!%s` and `!chismis` commands will not work.", EnvFileName, AteLuningningAICommand)
	}

	return config, nil
}

// configureLogging sets up logging based on configuration.
func configureLogging(debug bool) {
	if debug {
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile | log.Lmicroseconds)
	} else {
		log.SetFlags(log.Ldate | log.Ltime)
	}
}
