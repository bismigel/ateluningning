package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Bot wires together configuration, the AI client, the lock store, sessions and
// the command registry, and owns the Discord connection lifecycle.
type Bot struct {
	cfg      *AppConfig
	ai       *AIClient
	store    *LockStore
	sessions *SessionManager
	registry *Registry

	startMu   sync.RWMutex
	startTime time.Time
}

// NewBot constructs a bot and registers its commands.
func NewBot(cfg *AppConfig, store *LockStore) *Bot {
	b := &Bot{
		cfg:      cfg,
		ai:       NewAIClient(cfg.DeepSeekAPIKey, DefaultDeepSeekModel, cfg.Debug),
		store:    store,
		sessions: NewSessionManager(),
		registry: NewRegistry(),
	}
	b.registerCommands()
	return b
}

// Run connects to Discord and blocks until a termination signal is received.
func (b *Bot) Run() error {
	go b.cleanupLoop()

	dg, err := discordgo.New("Bot " + b.cfg.DiscordToken)
	if err != nil {
		return fmt.Errorf("failed to create Discord session: %w", err)
	}

	dg.AddHandler(b.onReady)
	dg.AddHandler(b.onMessage)
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent

	if err := dg.Open(); err != nil {
		return fmt.Errorf("cannot open Discord connection: %w", err)
	}
	defer func() {
		if closeErr := dg.Close(); closeErr != nil {
			log.Printf("Error closing Discord connection: %v", closeErr)
		}
	}()

	log.Println("Bot is now running. Press CTRL-C to exit.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	log.Println("Received termination signal. Shutting down...")
	return nil
}

// cleanupLoop periodically expires idle chat sessions.
func (b *Bot) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		b.sessions.CleanupInactiveSessions()
	}
}

// onReady records startup time and logs the connected guilds.
func (b *Bot) onReady(s *discordgo.Session, _ *discordgo.Ready) {
	log.Printf("Bot connected as %s#%s (ID: %s)", s.State.User.Username, s.State.User.Discriminator, s.State.User.ID)
	for _, guild := range s.State.Guilds {
		log.Printf("Connected to guild: %s (ID: %s)", guild.Name, guild.ID)
	}
	b.setStart(time.Now())
	rand.Seed(time.Now().UnixNano())
}

func (b *Bot) setStart(t time.Time) {
	b.startMu.Lock()
	b.startTime = t
	b.startMu.Unlock()
}

// Uptime returns how long the bot has been serving (zero until ready).
func (b *Bot) Uptime() time.Duration {
	b.startMu.RLock()
	defer b.startMu.RUnlock()
	if b.startTime.IsZero() {
		return 0
	}
	return time.Since(b.startTime)
}

// isAdmin reports whether userID is a configured administrator.
func (b *Bot) isAdmin(userID string) bool {
	for _, adminID := range b.cfg.AdminUserIDs {
		if userID == adminID {
			return true
		}
	}
	return false
}

// onMessage is the single entry point for incoming messages: it routes plain
// messages into an active chat session, and prefixed messages into the command
// registry.
func (b *Bot) onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	logIncoming(m)

	// A message with attachments (or without the prefix) continues an active
	// session instead of being parsed as a command.
	if session, exists := b.sessions.GetSession(m.Author.ID, m.ChannelID); exists {
		if !strings.HasPrefix(m.Content, Prefix) || len(m.Attachments) > 0 {
			b.handleSessionChat(s, m, session)
			return
		}
	}

	if !strings.HasPrefix(m.Content, Prefix) {
		return
	}

	fields := strings.Fields(strings.TrimPrefix(m.Content, Prefix))
	if len(fields) == 0 {
		return
	}
	b.dispatch(s, m, fields[0], fields[1:])
}

// dispatch applies admin gating and command locks, then runs the handler.
func (b *Bot) dispatch(s *discordgo.Session, m *discordgo.MessageCreate, name string, args []string) {
	cmd, ok := b.registry.Lookup(name)
	if !ok {
		SafeSend(s, m.ChannelID, "Oh, dear, I didn't quite catch that. Ate Luningning doesn't know that command. Try `!help` to see all my services!")
		return
	}

	if cmd.AdminOnly && !b.isAdmin(m.Author.ID) {
		SafeSend(s, m.ChannelID, AteLuningningErrorPrefix+"Oh, I'm sorry, my dear, only Ate Luningning's special helpers can use that command!")
		return
	}

	if cmd.Lockable {
		locked, err := b.store.IsLocked(cmd.Name, m.ChannelID)
		if err != nil {
			log.Printf("Database error checking %s lock status: %v", cmd.Name, err)
			SafeSend(s, m.ChannelID, AteLuningningErrorPrefix+"My little database helper is having a problem. Please try again later!")
			return
		}
		if locked {
			SafeSend(s, m.ChannelID, fmt.Sprintf("%sI'm sorry, my dear, the `!%s` command is currently **locked** in this channel. Ate Luningning needs a break for that one right now!", AteLuningningErrorPrefix, cmd.Name))
			return
		}
	}

	ctx := &Context{Bot: b, Discord: s, Msg: m, Args: args}
	if err := cmd.Run(ctx); err != nil {
		log.Printf("command %s failed: %v", cmd.Name, err)
	}
}

// handleSessionChat processes a free-form message inside an active session.
func (b *Bot) handleSessionChat(s *discordgo.Session, m *discordgo.MessageCreate, session *Session) {
	messageContent := m.Content

	if len(m.Attachments) > 0 {
		for _, attachment := range m.Attachments {
			textContent, err := processAttachment(attachment)
			if err != nil {
				log.Printf("Error processing attachment %s: %v", attachment.Filename, err)
				SafeSend(s, m.ChannelID, fmt.Sprintf("%sI couldn't read the file '%s'. Please make sure it's a .txt file!", AteLuningningErrorPrefix, attachment.Filename))
				continue
			}
			if messageContent != "" {
				messageContent += "\n\n[File: " + attachment.Filename + "]\n" + textContent
			} else {
				messageContent = "[File: " + attachment.Filename + "]\n" + textContent
			}
		}
	}

	if messageContent == "" {
		SafeSend(s, m.ChannelID, fmt.Sprintf("%sI see you sent something, but I couldn't find any text to process!", AteLuningningErrorPrefix))
		return
	}

	// Snapshot the history BEFORE recording the new turn: ChatWithSession
	// appends the prompt itself, so passing the post-AddMessage history would
	// send the latest user message twice.
	history := make([]DeepSeekMessage, len(session.Messages))
	copy(history, session.Messages)
	b.sessions.AddMessage(m.Author.ID, m.ChannelID, "user", messageContent)

	stopTyping := startTyping(s, m.ChannelID)
	response, err := b.ai.ChatWithSession(messageContent, history)
	stopTyping()

	if err != nil {
		log.Printf("DeepSeek API error in session chat: %v", err)
		SafeSend(s, m.ChannelID, aiErrorText("My smart brain is taking a bit too long to respond. ", err, ""))
		return
	}

	b.sessions.AddMessage(m.Author.ID, m.ChannelID, "assistant", response)
	sendLongMessage(s, m.ChannelID, response, AteLuningningGreeting)
}

// logIncoming logs a one-line summary of an incoming message.
func logIncoming(m *discordgo.MessageCreate) {
	logMsg := fmt.Sprintf("Message received in channel %s from %s (ID: %s)", m.ChannelID, m.Author.Username, m.Author.ID)

	if m.Content != "" {
		content := m.Content
		if len(content) > 100 {
			content = content[:100] + "..."
		}
		logMsg += fmt.Sprintf(": %s", content)
	}

	if len(m.Attachments) > 0 {
		names := make([]string, 0, len(m.Attachments))
		for _, att := range m.Attachments {
			names = append(names, att.Filename)
		}
		logMsg += fmt.Sprintf(" [Attachments: %s]", strings.Join(names, ", "))
	}

	log.Printf(logMsg)
}
