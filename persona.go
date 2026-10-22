package main

// Ate Luningning's persona strings. Keeping them in one place makes the bot's
// voice easy to tweak without hunting through command handlers.
const (
	AteLuningningName        = "Ate Luningning"
	AteLuningningGreeting    = "Oh, hello there, my dear! Ate Luningning is here.. "
	AteLuningningErrorPrefix = "Oh dear, something went wrong in Ate Luningning's kitchen! "
)

// validSigns is the set of zodiac signs the horoscope command accepts.
var validSigns = map[string]bool{
	"aries": true, "taurus": true, "gemini": true,
	"cancer": true, "leo": true, "virgo": true,
	"libra": true, "scorpio": true, "sagittarius": true,
	"capricorn": true, "aquarius": true, "pisces": true,
}
