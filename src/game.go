package main

import (
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Game contains important data of one game that is needed for tiles
type Game struct {
	FileName     string
	Title        string
	ReleaseKey   string
	IconFileName string
}

// Sanitize cleans disallowed characters from all strings
func (g *Game) Sanitize(disallowedChars *regexp.Regexp) {
	g.FileName = disallowedChars.ReplaceAllString(g.ReleaseKey, "")
	g.Title = disallowedChars.ReplaceAllString(strings.SplitN(g.Title, ":", 2)[1], "")
	g.IconFileName = strings.ReplaceAll(g.IconFileName, ".webp", ".png")
}

// ExistsIn searches for current game in a list
func (g *Game) ExistsIn(games []Game) bool {
	for _, game := range games {
		if g.Title == game.Title {
			log.Warnf("'%s' (%s) already exists with ReleaseKey '%s'. Hide one of them in your games library.", game.Title, g.ReleaseKey, game.ReleaseKey)
			return true
		}
	}
	return false
}
