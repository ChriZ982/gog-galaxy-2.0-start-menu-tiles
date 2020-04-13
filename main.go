package main

import (
	"bytes"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

const sqlTaggedGames = `SELECT UserReleaseTags.releaseKey, WebCacheResources.filename, GamePieces.value FROM UserReleaseTags
LEFT JOIN WebCacheResources ON UserReleaseTags.releaseKey = WebCacheResources.releaseKey
LEFT JOIN GamePieces ON UserReleaseTags.releaseKey = GamePieces.releaseKey
WHERE UserReleaseTags.tag = 'StartMenuTiles' AND WebCacheResources.webCacheResourceTypeId = 2 AND GamePieces.gamePieceTypeId = 11 AND GamePieces.userId <> 0`

// Uses a partial Start Layout (https://docs.microsoft.com/en-us/windows/configuration/customize-and-export-start-layout#configure-a-partial-start-layout)
const partialStartLayoutBegin = `<LayoutModificationTemplate xmlns:defaultlayout="http://schemas.microsoft.com/Start/2014/FullDefaultLayout" xmlns:start="http://schemas.microsoft.com/Start/2014/StartLayout" Version="1" xmlns="http://schemas.microsoft.com/Start/2014/LayoutModification">
  <LayoutOptions StartTileGroupCellWidth="8" />
  <DefaultLayoutOverride LayoutCustomizationRestrictionType="OnlySpecifiedGroups">
    <StartLayoutCollection>
	  <defaultlayout:StartLayout GroupCellWidth="8">
		<start:Group Name="">`

const startLayoutTile = `
          <start:DesktopApplicationTile Size="2x2" Column="%d" Row="%d" DesktopApplicationLinkPath="%%APPDATA%%\Microsoft\Windows\Start Menu\Programs\GOG.com\GameTiles\%s.lnk" />`

const partialStartLayoutEnd = `
        </start:Group>
      </defaultlayout:StartLayout>
    </StartLayoutCollection>
  </DefaultLayoutOverride>
</LayoutModificationTemplate>`

const powershellCreateShortcut = `Invoke-WebRequest -Uri https://images.gog.com/%s?namespace=gamesdb -OutFile "%s.png"
$WScriptShell = New-Object -ComObject WScript.Shell
$Shortcut = $WScriptShell.CreateShortcut("%s.lnk")
$Shortcut.TargetPath = "%s.bat"
$Shortcut.Save()`

const powershellApplyStartLayout = `$fileName = "$(Get-Location)\data\PartialStartLayout.xml"
Export-StartLayout -Path "$(Get-Location)\data\StartLayoutBackup.xml"

$WindowsUpdateRegKey = "HKCU:\Software\Policies\Microsoft\Windows\Explorer"
if(-not (Test-Path $WindowsUpdateRegKey))
{
	New-Item -Path $WindowsUpdateRegKey -Force
}
Set-ItemProperty -Path $WindowsUpdateRegKey -Name StartLayoutFile -Value "$fileName" -Type ExpandString
Set-ItemProperty -Path $WindowsUpdateRegKey -Name LockedStartLayout -Value 1 -Type DWord
Stop-Process -ProcessName explorer
sleep 10
Set-ItemProperty -Path $WindowsUpdateRegKey -Name LockedStartLayout -Value 0 -Type DWord
Stop-Process -ProcessName explorer`

const visualElements = `<?xml version="1.0" encoding="utf-8"?>
<Application xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
	<VisualElements ShowNameOnSquare150x150Logo="on" Square150x150Logo="VisualElements\MediumIcon%[1]s.png" Square70x70Logo="VisualElements\MediumIcon%[1]s.png" ForegroundText="light" BackgroundColor="#5A391B" />
</Application>`

func execPowershell(cmdText string, args ...interface{}) {
	fileName := "tmp.ps1"
	writeFile(fileName, fmt.Sprintf(cmdText, args...))
	cmd := exec.Command("powershell", "-noexit", "& .\\"+fileName)
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
			log.Debug(strings.TrimRight(line, " "))
		}
	}
	os.Remove(fileName)
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
	database, _ := sql.Open("sqlite3", "./data/galaxy-2.0.db?mode=ro")
	rows, _ := database.Query(sqlTaggedGames)
	database.Close()

	log.Info("Parsing games...")
	var games map[string]map[string]string = make(map[string]map[string]string)
	var releaseKey string
	var iconFileName string
	var title string
	reg, err := regexp.Compile("[^A-Za-z0-9 ]+")
	if err != nil {
		log.Fatal(err)
	}
	for rows.Next() {
		rows.Scan(&releaseKey, &iconFileName, &title)
		if _, ok := games[releaseKey]; !ok {
			games[releaseKey] = make(map[string]string)
		}
		games[releaseKey]["iconFileName"] = strings.ReplaceAll(iconFileName, ".webp", ".png")
		games[releaseKey]["title"] = reg.ReplaceAllString(strings.Split(title, ":")[1], "")
	}

	log.Info("Creating shortcuts...")
	home, _ := os.UserHomeDir()
	startMenuPath := home + "\\Appdata\\Roaming\\Microsoft\\Windows\\Start Menu\\Programs\\GOG.com\\GameTiles\\"
	err = os.MkdirAll(startMenuPath+"VisualElements", os.ModePerm)
	if err != nil {
		log.Fatal("Error while creating Start Menu folder:", err)
	}

	log.Info("Creating Start Menu...")
	appendText := partialStartLayoutBegin
	tileCount := 0
	for key, val := range games {
		appendText += fmt.Sprintf(startLayoutTile,
			(tileCount%4)*2,
			(tileCount-(tileCount%4))/4,
			val["title"])
		tileCount++
		writeFile(startMenuPath+key+".bat",
			`"C:\Program Files (x86)\GOG Galaxy\GalaxyClient.exe" /command=runGame /gameId=`+key)
		writeFile(startMenuPath+key+".VisualElementsManifest.xml", fmt.Sprintf(visualElements, key))

		execPowershell(powershellCreateShortcut,
			val["iconFileName"], startMenuPath+"VisualElements\\MediumIcon"+key, startMenuPath+val["title"], startMenuPath+key)
	}

	appendText += partialStartLayoutEnd
	writeFile("data\\PartialStartLayout.xml", appendText)

	log.Info("Updating Start Menu...")
	execPowershell(powershellApplyStartLayout)
	log.Info("Program finished!")
}
