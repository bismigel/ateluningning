package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

func newHelloCommand() *Command {
	return &Command{
		Name:    "hello",
		Usage:   "`!hello`",
		Summary: "A warm greeting from Ate Luningning!",
		Inline:  true,
		Run: func(c *Context) error {
			c.Reply(fmt.Sprintf("%sHello there, my dear!!!!! %s is here to serve you some wisdom and good vibes. What can I get for you today? Try `!help` for the menu!", AteLuningningGreeting, AteLuningningName))
			return nil
		},
	}
}

func newPingCommand() *Command {
	return &Command{
		Name:    "ping",
		Usage:   "`!ping`",
		Summary: "A quick tap to check if Ate Luningning's kitchen is open!",
		Inline:  true,
		Run: func(c *Context) error {
			latency := time.Since(c.Msg.Timestamp)
			c.Reply(fmt.Sprintf("Oh, a little tap on my lunch counter! Pong! 🏓 (My little helper says I'm super fast, only %s!)", latency.Round(time.Millisecond)))
			return nil
		},
	}
}

func newUptimeCommand() *Command {
	return &Command{
		Name:    "uptime",
		Usage:   "`!uptime`",
		Summary: "See how long Ate Luningning's been serving you up!",
		Inline:  true,
		Run: func(c *Context) error {
			c.Reply(fmt.Sprintf("%sMy kitchen's been open for business for: %s! Always ready to serve!", AteLuningningGreeting, formatDuration(c.Bot.Uptime())))
			return nil
		},
	}
}

func newCalcCommand() *Command {
	return &Command{
		Name:    "calc",
		Usage:   "`!calc <num1> <op> <num2>`",
		Summary: "Let Ate Luningning crunch those numbers for you! (e.g., `!calc 5 + 3`). Supports `+`, `-`, `*`, `/`.",
		Run: func(c *Context) error {
			if len(c.Args) != 3 {
				c.Reply(fmt.Sprintf("%sUsage: `!calc <number1> <operator> <number2>` (e.g., `!calc 5 + 3`)", AteLuningningGreeting))
				return nil
			}

			num1, num1Err := strconv.ParseFloat(c.Args[0], 64)
			num2, num2Err := strconv.ParseFloat(c.Args[2], 64)
			if num1Err != nil || num2Err != nil {
				c.Reply(AteLuningningErrorPrefix + "Hmm, those don't look like proper numbers, my dear.")
				return nil
			}

			var result float64
			switch c.Args[1] {
			case "+":
				result = num1 + num2
			case "-":
				result = num1 - num2
			case "*":
				result = num1 * num2
			case "/":
				if num2 == 0 {
					c.Reply(AteLuningningErrorPrefix + "Oh, we can't divide by zero! That's a big no-no in my kitchen!")
					return nil
				}
				result = num1 / num2
			default:
				c.Reply(AteLuningningErrorPrefix + "That's not an operator I know, dear. Please use `+`, `-`, `*`, or `/`.")
				return nil
			}

			c.Reply(fmt.Sprintf("Ate Luningning's calculation says: %.2f", result))
			return nil
		},
	}
}

func newChooseCommand() *Command {
	return &Command{
		Name:    "choose",
		Usage:   "`!choose <option1>, <option2>, ...`",
		Summary: "Can't decide? Let Ate Luningning pick from the menu for you! (e.g., `!choose sinigang, adobo, pancit`).",
		Run: func(c *Context) error {
			if len(c.Args) == 0 {
				c.Reply(fmt.Sprintf("%sUsage: `!choose <option1>, <option2>, ...`", AteLuningningGreeting))
				return nil
			}

			options := strings.Split(strings.Join(c.Args, " "), ",")
			cleanOptions := make([]string, 0, len(options))
			for _, opt := range options {
				if trimmed := strings.TrimSpace(opt); trimmed != "" {
					cleanOptions = append(cleanOptions, trimmed)
				}
			}
			if len(cleanOptions) == 0 {
				c.Reply(AteLuningningErrorPrefix + "Please give me at least one valid option to choose from, dear.")
				return nil
			}

			chosen := cleanOptions[rand.Intn(len(cleanOptions))]
			c.Reply(fmt.Sprintf("%sHmm, tough choice, huh? Let Ate Luningning help you pick from the menu! I say: **%s**", AteLuningningGreeting, chosen))
			return nil
		},
	}
}

func newRollCommand() *Command {
	return &Command{
		Name:    "roll",
		Usage:   "`!roll <NdX>`",
		Summary: "Time to shake things up! Ate Luningning will roll some dice for you (e.g., `!roll 1d6`, `!roll 2d10`).",
		Run: func(c *Context) error {
			if len(c.Args) != 1 {
				c.Reply(fmt.Sprintf("%sUsage: `!roll <NdX>` (e.g., `!roll 1d6`, `!roll 2d10`)", AteLuningningGreeting))
				return nil
			}

			rollStr := strings.ToLower(c.Args[0])
			parts := strings.Split(rollStr, "d")
			if len(parts) != 2 {
				c.Reply(AteLuningningErrorPrefix + "That's not how we roll dice, dear. Please use `NdX` (e.g., `1d6`).")
				return nil
			}

			numDice, numDiceErr := strconv.Atoi(parts[0])
			sides, sidesErr := strconv.Atoi(parts[1])
			if numDiceErr != nil || sidesErr != nil || numDice <= 0 || sides <= 0 {
				c.Reply(AteLuningningErrorPrefix + "Invalid dice format. N and X must be positive numbers (e.g., `1d6`).")
				return nil
			}
			if numDice > 100 || sides > 1000 {
				c.Reply(AteLuningningErrorPrefix + "Oh my, that's too many dice or too many sides! (Max 100d1000 in my kitchen!)")
				return nil
			}

			total := 0
			rolls := make([]int, 0, numDice)
			for i := 0; i < numDice; i++ {
				roll := rand.Intn(sides) + 1
				total += roll
				rolls = append(rolls, roll)
			}

			if numDice == 1 {
				c.Reply(fmt.Sprintf("Time to shake things up a bit! Ate Luningning is rolling the dice for you... Rolled %s: **%d**", rollStr, total))
			} else {
				c.Reply(fmt.Sprintf("Time to shake things up a bit! Ate Luningning is rolling the dice for you... Rolled %s: %v = **%d**", rollStr, rolls, total))
			}
			return nil
		},
	}
}
