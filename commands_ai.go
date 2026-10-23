package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func newVerseCommand() *Command {
	return &Command{
		Name:    "verse",
		Usage:   "`!verse`",
		Summary: "Some spiritual nourishment from the Good Book! 📖",
		Inline:  true,
		Run: func(c *Context) error {
			verse, err := fetchBibleVerse(c.Bot.cfg.Debug)
			if err != nil {
				log.Printf("Verse retrieval error: %v", err)
				c.Reply(AteLuningningErrorPrefix + "My apologies, dear, I couldn't quite whip that up right now. Try again later, okay?")
				return nil
			}
			c.Reply(fmt.Sprintf("%sHere's some spiritual nourishment for your soul, straight from Ate Luningning's pantry of wisdom! 📖", AteLuningningGreeting))
			c.ReplyEmbed(createVerseEmbed(verse))
			return nil
		},
	}
}

func newHoroscopeCommand() *Command {
	return &Command{
		Name:     "horoscope",
		Usage:    "`!horoscope <zodiacSign>`",
		Summary:  "What the stars are cooking for your daily forecast, powered by my smart brain! (e.g., `!horoscope leo`).",
		Inline:   true,
		Lockable: true,
		Run: func(c *Context) error {
			if c.Bot.cfg.DeepSeekAPIKey == "" {
				c.Reply(AteLuningningErrorPrefix + "It looks like my horoscope crystal ball isn't connected right now. Please tell my programmer to check the `DEEPSEEK_API_KEY`!")
				return nil
			}
			if len(c.Args) < 1 {
				c.Reply(fmt.Sprintf("%sPlease tell Ate Luningning your zodiac sign, my dear, like `!horoscope leo`.", AteLuningningGreeting))
				return nil
			}

			sign := c.Args[0]
			c.Reply(fmt.Sprintf("%sLet me consult the stars for you, my dear %s...", AteLuningningGreeting, CapitalizeFirstLetter(strings.ToLower(sign))))

			stopTyping := startTyping(c.Discord, c.ChannelID())
			data, err := c.Bot.ai.Horoscope(sign)
			stopTyping()

			if err != nil {
				log.Printf("Horoscope retrieval error for %s from DeepSeek: %v", sign, err)
				c.Reply(aiErrorText("I couldn't quite read the stars for you right now. ", err, ". Please try again or provide a valid sign."))
				return nil
			}

			c.Reply(fmt.Sprintf("%sLet's see what the stars are cooking for you today, my dear!", AteLuningningGreeting))
			c.ReplyEmbed(createHoroscopeEmbed(data))
			c.Reply("*(Just a little heads-up from Ate Luningning: This horoscope is your daily forecast, cooked by my smart brain!)*")
			return nil
		},
	}
}

func newAskCommand() *Command {
	return &Command{
		Name:     AteLuningningAICommand,
		Usage:    "`!askluningning <prompt>`",
		Summary:  "Ask Ate Luningning's smart brain anything! (e.g., `!askluningning What is the capital of France?`).",
		Lockable: true,
		Run: func(c *Context) error {
			if c.Bot.cfg.DeepSeekAPIKey == "" {
				c.Reply(AteLuningningErrorPrefix + "My smart brain isn't fully connected right now. Please tell my programmer to set the `DEEPSEEK_API_KEY`!")
				return nil
			}

			prompt := c.Prompt()
			hasAttachments := len(c.Msg.Attachments) > 0
			if hasAttachments {
				for _, attachment := range c.Msg.Attachments {
					textContent, err := processAttachment(attachment)
					if err != nil {
						log.Printf("Error processing attachment %s: %v", attachment.Filename, err)
						c.Reply(fmt.Sprintf("%sI couldn't read the file '%s'. Please make sure it's a .txt file!", AteLuningningErrorPrefix, attachment.Filename))
						continue
					}
					if prompt != "" {
						prompt += "\n\n[File: " + attachment.Filename + "]\n" + textContent
					} else {
						prompt = "[File: " + attachment.Filename + "]\n" + textContent
					}
				}
			}

			if prompt == "" && !hasAttachments {
				c.Reply(fmt.Sprintf("%sWhat do you want to ask my smart brain, dear? Just type `!%s <your question here>` or attach a .txt file!", AteLuningningGreeting, AteLuningningAICommand))
				return nil
			}
			if prompt == "" && hasAttachments {
				prompt = "Please analyze the content of the attached file(s)."
			}

			c.Reply(fmt.Sprintf("%sAte Luningning is consulting the magic crystal ball for you! Give me a moment while I prepare your answer...", AteLuningningGreeting))

			stopTyping := startTyping(c.Discord, c.ChannelID())
			response, err := c.Bot.ai.Chat(prompt)
			stopTyping()

			if err != nil {
				log.Printf("DeepSeek API error: %v", err)
				c.Reply(aiErrorText("My smart brain is taking a bit too long to respond. ", err, ""))
				return nil
			}

			sendLongMessage(c.Discord, c.ChannelID(), response, "Here's what the cosmos revealed for you, my dear:")
			return nil
		},
	}
}

func newChismisCommand() *Command {
	return &Command{
		Name:     WorkStartCommand,
		Usage:    "`!chismis`",
		Summary:  "Start a session-based chat with Ate Luningning! I'll remember our conversation for 30 minutes. Just talk to me normally after starting.",
		Lockable: true,
		Run: func(c *Context) error {
			if c.Bot.cfg.DeepSeekAPIKey == "" {
				c.Reply(AteLuningningErrorPrefix + "My smart brain isn't fully connected right now. Please tell my programmer to set the `DEEPSEEK_API_KEY`!")
				return nil
			}

			if session, exists := c.Bot.sessions.GetSession(c.UserID(), c.ChannelID()); exists {
				c.Reply(fmt.Sprintf("%sYou already have an active work session, my dear! (Last active: %s ago)", AteLuningningGreeting, formatDuration(time.Since(session.LastActive))))
				return nil
			}

			c.Bot.sessions.StartSession(c.UserID(), c.ChannelID())
			c.Reply(fmt.Sprintf("%sWork session started! I'll remember our conversation for the next 30 minutes. Just talk to me normally (no need for !commands). Type `!yokona` to end the session early.", AteLuningningGreeting))
			c.Bot.sessions.AddMessage(c.UserID(), c.ChannelID(), "system", "You are Ate Luningning, a friendly and helpful Filipino auntie who gives advice with warmth and humor. You're having a conversation with someone who started a work session with you.")
			return nil
		},
	}
}

func newYokonaCommand() *Command {
	return &Command{
		Name:     "yokona",
		Usage:    "`!yokona`",
		Summary:  "End your current work session and clear the conversation history.",
		Lockable: true,
		Run: func(c *Context) error {
			if c.Bot.sessions.EndSession(c.UserID(), c.ChannelID()) {
				c.Reply(fmt.Sprintf("%sWork session ended! Our conversation has been cleared. Start a new one with `!chismis` whenever you're ready, my dear!", AteLuningningGreeting))
			} else {
				c.Reply(fmt.Sprintf("%sYou don't have an active work session to end, dear. Start one with `!chismis`!", AteLuningningGreeting))
			}
			return nil
		},
	}
}

// createHoroscopeEmbed renders structured horoscope data as a Discord embed.
func createHoroscopeEmbed(data *HoroscopeData) *discordgo.MessageEmbed {
	displaySign := CapitalizeFirstLetter(data.ZodiacSign)

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s Daily Horoscope 🌟", displaySign),
		Description: data.Prediction,
		Color:       0xFEE75C,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Love Compatibility", Value: data.LoveCompatibility, Inline: false},
			{Name: "Strengths", Value: data.Strength, Inline: false},
			{Name: "Weaknesses", Value: data.Weakness, Inline: false},
			{Name: "Lucky Color", Value: data.Color, Inline: true},
			{Name: "Lucky Number", Value: data.Number, Inline: true},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Horoscope by Ate Luningning's Smart Brain (Daily Forecast)",
		},
	}
}
