package main

import (
	"fmt"
	"log"
	"strings"
)

// newLockCommand builds !lock (lock=true) or !unlock (lock=false). Both share
// the same validation: the target must be a registered, lockable command.
func newLockCommand(lock bool) *Command {
	name := "lock"
	status := "locked"
	if !lock {
		name = "unlock"
		status = "unlocked"
	}

	return &Command{
		Name:      name,
		Usage:     "`!" + name + " <command>`",
		Summary:   "Temporarily turn a service off/on in this channel (admin only).",
		AdminOnly: true,
		Run: func(c *Context) error {
			if len(c.Args) < 1 {
				c.Reply(fmt.Sprintf("%sPlease tell Ate Luningning which command to %s, like `!%s %s`.", AteLuningningGreeting, name, name, AteLuningningAICommand))
				return nil
			}

			target, ok := c.Bot.registry.Lookup(strings.ToLower(c.Args[0]))
			if !ok || !target.Lockable {
				c.Reply(fmt.Sprintf("%sI can only %s the %s commands for now, dear.", AteLuningningErrorPrefix, name, c.Bot.registry.LockableList()))
				return nil
			}

			if err := c.Bot.store.SetLocked(target.Name, c.ChannelID(), lock); err != nil {
				log.Printf("Error setting lock status for %s in channel %s: %v", target.Name, c.ChannelID(), err)
				c.Reply(AteLuningningErrorPrefix + fmt.Sprintf("I couldn't change the status of `!%s` in this channel right now, my dear. Please check my recipe book (logs)!", target.Name))
				return nil
			}

			c.Reply(fmt.Sprintf("%sOkay, my dear, the `!%s` command is now **%s** in this channel!", AteLuningningGreeting, target.Name, status))
			return nil
		},
	}
}
