package main

import (
	"bytes"
	"database/sql"
	"flag"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/go-vgo/robotgo"
	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

func execPowershell(cmds ...string) {
	for _, cmdText := range cmds {
		cmd := exec.Command("powershell", "-noexit", "& "+cmdText)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err != nil {
			log.Error(err)
		}
		for _, line := range strings.Split(stderr.String(), "\r\n") {
			if len(line) > 0 {
				log.Warn(strings.TrimRight(line, " "))
			}
		}
		for _, line := range strings.Split(stdout.String(), "\r\n") {
			if len(line) > 0 {
				log.Info(strings.TrimRight(line, " "))
			}
		}
	}
}

func writeFile(filePath string, contents string) {
	f, _ := os.Create(filePath)
	f.WriteString(contents)
	f.Close()
}

func main() {
	loglevel := flag.String("level", "INFO", "log level")
	flag.Parse()
	level, _ := log.ParseLevel(*loglevel)
	log.SetLevel(level)

	if runtime.GOOS != "windows" {
		log.Fatal("This App does only support Windows 10!")
	}

	log.Info("Reading GOG Galaxy 2.0 database...")
	database, _ := sql.Open("sqlite3", "./data/galaxy-2.0.db")
	database.Exec("PRAGMA wal_checkpoint")
	rows, _ := database.Query(`
		SELECT UserReleaseTags.releaseKey, WebCacheResources.filename, GamePieces.value FROM UserReleaseTags
		LEFT JOIN WebCacheResources ON UserReleaseTags.releaseKey = WebCacheResources.releaseKey
		LEFT JOIN GamePieces ON UserReleaseTags.releaseKey = GamePieces.releaseKey
		WHERE UserReleaseTags.tag = 'StartMenuTiles' AND WebCacheResources.webCacheResourceTypeId = 2 AND GamePieces.gamePieceTypeId = 11 AND GamePieces.userId <> 0`)
	database.Close()

	log.Info("Parsing games...")
	var games map[string]map[string]string = make(map[string]map[string]string)
	var releaseKey string
	var iconFileName string
	var title string
	for rows.Next() {
		rows.Scan(&releaseKey, &iconFileName, &title)
		if _, ok := games[releaseKey]; !ok {
			games[releaseKey] = make(map[string]string)
		}
		games[releaseKey]["iconFileName"] = strings.ReplaceAll(iconFileName, ".webp", ".png")
		games[releaseKey]["title"] = strings.Trim(strings.Split(title, ":")[1], "\"{}")
	}

	log.Info("Creating shortcuts...")
	home, _ := os.UserHomeDir()
	startMenuPath := home + "\\Appdata\\Roaming\\Microsoft\\Windows\\Start Menu\\Programs\\GOG.com\\GameTiles"
	err := os.MkdirAll(startMenuPath+"\\VisualElements", os.ModePerm)
	if err != nil {
		log.Fatal("Error while creating Start Menu folder:", err)
	}

	for key, val := range games {
		linkPath := startMenuPath + "\\"
		writeFile(linkPath+key+".bat",
			`"C:\Program Files (x86)\GOG Galaxy\GalaxyClient.exe" /command=runGame /gameId=`+key)
		writeFile(linkPath+key+".VisualElementsManifest.xml",
			`<?xml version="1.0" encoding="utf-8"?>
<Application xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
	<VisualElements ShowNameOnSquare150x150Logo="on" Square150x150Logo="VisualElements\MediumIcon`+key+`.png" Square70x70Logo="VisualElements\SmallIcon`+key+`.png" ForegroundText="light" BackgroundColor="#5A391B" />
</Application>`)
		writeFile("create_shortcut.ps1",
			`Invoke-WebRequest -Uri https://images.gog.com/`+val["iconFileName"]+`?namespace=gamesdb -OutFile "`+linkPath+"VisualElements\\MediumIcon"+key+`.png"
Copy-Item "`+linkPath+"VisualElements\\MediumIcon"+key+`.png" "`+linkPath+"VisualElements\\SmallIcon"+key+`.png"
$SourceFileLocation = "`+linkPath+key+`.bat"
$ShortcutLocation = "`+linkPath+val["title"]+`.lnk"
$WScriptShell = New-Object -ComObject WScript.Shell
$Shortcut = $WScriptShell.CreateShortcut("$ShortcutLocation")
$Shortcut.TargetPath = "$SourceFileLocation"
$Shortcut.Save()`)
		execPowershell(".\\create_shortcut.ps1")
		os.Remove("create_shortcut.ps1")
	}
	robotgo.KeyTap("cmd")
	robotgo.Sleep(1)
	robotgo.KeyTap("down")
	robotgo.Sleep(1)
	robotgo.TypeStr("G")
	log.Info(len(games))
}
