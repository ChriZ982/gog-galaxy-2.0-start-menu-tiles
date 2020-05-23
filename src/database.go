package main

import (
	"database/sql"
	"fmt"
	"sort"

	log "github.com/sirupsen/logrus"
)

var sqlTaggedGames = fmt.Sprintf(gameInfoJoins, "SELECT urt.releaseKey, wcr.filename, gp.value FROM UserReleaseTags urt",
	"urt.releaseKey", "AND urt.tag = ?")

var sqlInstalledGames = fmt.Sprintf(gameInfoJoins,
	`SELECT Installed.releaseKey, wcr.filename, gp.value FROM
	(SELECT 'gog_' || ibp.productId as releaseKey FROM InstalledBaseProducts ibp
	UNION ALL
	SELECT p.name || '_' || iep.productId as releaseKey FROM InstalledExternalProducts iep
	JOIN Platforms p ON iep.platformId = p.id WHERE iep.platformId <> 85) as Installed`, // Excluding Rockstar games as they seem to be always "installed"
	"Installed.releaseKey", "")

var sqlAllGames = fmt.Sprintf(gameInfoJoins, "SELECT lr.releaseKey, wcr.filename, gp.value FROM LibraryReleases lr",
	"lr.releaseKey", "AND lr.userId <> 0")

const gameInfoJoins = `%[1]s
LEFT JOIN WebCache wc ON %[2]s = wc.releaseKey
LEFT JOIN WebCacheResources wcr ON wc.id = wcr.webCacheId
LEFT JOIN WebCacheResourceTypes wcrt ON wcrt.id = wcr.webCacheResourceTypeId
LEFT JOIN GamePieces gp ON %[2]s = gp.releaseKey
LEFT JOIN GamePieceTypes gpt ON gpt.id = gp.gamePieceTypeId
LEFT JOIN UserReleaseProperties urp ON %[2]s = urp.releaseKey
WHERE wcrt.type = 'squareIcon' AND gpt.type = 'title' AND urp.isHidden = 0 AND gp.userId <> 0 %[3]s`

const lastCacheUpdate = `SELECT updateDate FROM GamePieceCacheUpdateDates WHERE userId <> 0`

func listGames() *[]Game {
	log.Info("Reading GOG Galaxy 2.0 database...")
	database, err := sql.Open("sqlite3", *gogDir+dbLocation+"?mode=ro")
	defer database.Close()
	if err != nil {
		log.Fatalf("Error while trying to open GOG Galaxy 2.0 database at '%s'. %s", *gogDir+dbLocation, err)
	}
	var cacheUpdate string
	err = database.QueryRow(lastCacheUpdate).Scan(&cacheUpdate)
	if err != nil {
		log.Fatal("Error while trying to get latest cache update. ", err)
	}
	log.Infof("Last cache update was on '%s'", cacheUpdate)
	var rows *sql.Rows
	switch *tagName {
	case "INSTALLED":
		rows, err = database.Query(sqlInstalledGames)
	case "ALL":
		rows, err = database.Query(sqlAllGames)
	default:
		rows, err = database.Query(sqlTaggedGames, *tagName)
	}
	if err != nil {
		log.Fatal("Error while running query on database. ", err)
	}

	log.Info("Parsing games...")
	var games []Game
	for rows.Next() {
		var game Game
		rows.Scan(&game.ReleaseKey, &game.IconFileName, &game.Title)
		game.Sanitize(disallowedChars)
		if !game.ExistsIn(games) {
			games = append(games, game)
		}
	}
	sort.Slice(games, func(i, j int) bool {
		return games[i].Title < games[j].Title
	})

	switch {
	case len(games) == 0:
		log.Fatal("No games found.")
	case len(games) > 150:
		log.Fatalf("Adding too many tiles causes unexpected behaviour. %d tiles can not be created safely.", len(games))
	case len(games) > 80:
		log.Warnf("Adding too many tiles causes unexpected behaviour. %d tiles will be added.", len(games))
	}
	return &games
}
