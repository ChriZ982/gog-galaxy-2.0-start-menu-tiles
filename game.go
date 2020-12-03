package main

import (
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

var disallowedChars = regexp.MustCompile("[^A-Za-z0-9 +_#!()=-]+")

// Game contains important data of one game that is needed for tiles
type Game struct {
	FileName     string
	Title        string
	ReleaseKey   string
	IconFileName string
}

// Sanitize cleans disallowed characters from all strings
func (g *Game) Sanitize() {
	g.FileName = disallowedChars.ReplaceAllString(g.ReleaseKey, "")
	g.Title = disallowedChars.ReplaceAllString(strings.SplitN(g.Title, ":", 2)[1], "")
	g.IconFileName = strings.ReplaceAll(g.IconFileName, ".webp", ".png")
}

// ExistsIn searches for current game in a list
func (g *Game) ExistsIn(games []Game) bool {
	for _, other := range games {
		if g.Title == other.Title {
			log.Warnf("'%s' (%s) already exists with ReleaseKey '%s'. Hide one of them in your games library.", other.Title, g.ReleaseKey, other.ReleaseKey)
			return true
		}
	}
	return false
}
