package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Context is what a command handler receives: the bot, the Discord session, the
// originating message, and the parsed arguments (everything after the command
// name, already split on whitespace).
type Context struct {
	Bot     *Bot
	Discord *discordgo.Session
	Msg     *discordgo.MessageCreate
	Args    []string
}

func (c *Context) ChannelID() string { return c.Msg.ChannelID }
func (c *Context) UserID() string    { return c.Msg.Author.ID }

// Prompt returns the arguments joined by spaces.
func (c *Context) Prompt() string { return strings.Join(c.Args, " ") }

// Reply sends a plain message to the originating channel.
func (c *Context) Reply(content string) { SafeSend(c.Discord, c.ChannelID(), content) }

// ReplyEmbed sends an embed to the originating channel.
func (c *Context) ReplyEmbed(embed *discordgo.MessageEmbed) {
	SafeSendEmbed(c.Discord, c.ChannelID(), embed)
}

// Command describes a single bot command. Adding a command means implementing
// Run and registering it in registerCommands; locking, admin gating and help
// listing are all derived from these fields.
type Command struct {
	Name      string   // the word after the prefix, e.g. "horoscope"
	Aliases   []string // alternative names
	Usage     string   // shown in !help, e.g. "`!horoscope <zodiacSign>`"
	Summary   string   // shown in !help
	Inline    bool     // !help layout hint
	AdminOnly bool     // only AdminUserIDs may run it
	Lockable  bool     // can be toggled with !lock / !unlock
	Hidden    bool     // not listed in !help
	Run       func(*Context) error
}

// Registry maps names/aliases to commands and preserves registration order for
// help output.
type Registry struct {
	byName  map[string]*Command
	ordered []*Command
}

// NewRegistry creates an empty command registry.
func NewRegistry() *Registry {
	return &Registry{byName: make(map[string]*Command)}
}

// Register adds commands. The first command to claim a name wins.
func (r *Registry) Register(cmds ...*Command) {
	for _, c := range cmds {
		r.ordered = append(r.ordered, c)
		if _, exists := r.byName[c.Name]; !exists {
			r.byName[c.Name] = c
		}
		for _, alias := range c.Aliases {
			if _, exists := r.byName[alias]; !exists {
				r.byName[alias] = c
			}
		}
	}
}

// Lookup resolves a command name or alias.
func (r *Registry) Lookup(name string) (*Command, bool) {
	c, ok := r.byName[name]
	return c, ok
}

// All returns commands in registration order.
func (r *Registry) All() []*Command { return r.ordered }

// LockableList renders the lockable commands for the !lock error message.
func (r *Registry) LockableList() string {
	var names []string
	for _, c := range r.ordered {
		if c.Lockable {
			names = append(names, "`!"+c.Name+"`")
		}
	}
	return humanJoin(names)
}

// registerCommands wires every command into the registry. This is the single
// place to add a new command; everything else (dispatch, !help, lock support)
// is derived from it.
func (b *Bot) registerCommands() {
	b.registry.Register(
		newHelloCommand(),
		newPingCommand(),
		newUptimeCommand(),
		newVerseCommand(),
		newHoroscopeCommand(),
		newAskCommand(),
		newChismisCommand(),
		newYokonaCommand(),
		newCalcCommand(),
		newChooseCommand(),
		newRollCommand(),
		newHelpCommand(),
		newLockCommand(true),
		newLockCommand(false),
	)
}

// newHelpCommand builds the !help menu from the registry, so new commands show
// up automatically.
func newHelpCommand() *Command {
	return &Command{
		Name:   "help",
		Usage:  "`!help`",
		Hidden: true,
		Run: func(c *Context) error {
			embed := &discordgo.MessageEmbed{
				Title:       "🤖 Ate Luningning's Menu of Services",
				Description: "Here are the delicious commands I can serve you today:",
				Color:       0x7289DA,
				Timestamp:   time.Now().Format(time.RFC3339),
				Footer: &discordgo.MessageEmbedFooter{
					Text: "Ate Luningning is always here to help you get your daily dose of wisdom and fun!",
				},
			}

			for _, cmd := range c.Bot.registry.All() {
				if cmd.Hidden || cmd.AdminOnly {
					continue
				}
				name := cmd.Usage
				if name == "" {
					name = "`!" + cmd.Name + "`"
				}
				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
					Name:   name,
					Value:  cmd.Summary,
					Inline: cmd.Inline,
				})
			}

			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "⚙️ Admin Commands (For Ate Luningning's helpers)",
				Value:  fmt.Sprintf("`!lock <command>` / `!unlock <command>` : To temporarily turn off/on services like `!%s` or `!horoscope` **in this channel only**. (Only for special helpers!)", AteLuningningAICommand),
				Inline: false,
			})

			c.ReplyEmbed(embed)
			return nil
		},
	}
}
