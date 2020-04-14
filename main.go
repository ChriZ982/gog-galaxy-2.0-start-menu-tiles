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
	"github.com/shiena/ansicolor"
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
          <start:DesktopApplicationTile Size="%[1]dx%[1]d" Column="%[2]d" Row="%[3]d" DesktopApplicationLinkPath="%%APPDATA%%\Microsoft\Windows\Start Menu\Programs\GOG.com\GameTiles\%[4]s.lnk" />`

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

const powershellApplyStartLayout = `$fileName = "$(Get-Location)\PartialStartLayout.xml"
Export-StartLayout -Path "StartLayoutBackup.xml"
sleep 5

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

var alphanum = regexp.MustCompile("[^A-Za-z0-9 ]+")

var loglevel = flag.String("level", "INFO", "Defines log level.")
var databaseFile = flag.String("database", "C:/ProgramData/GOG.com/Galaxy/storage/galaxy-2.0.db", "Path to GOG Galaxy 2.0 database.")
var startMenuDir = flag.String("startFolder", "/Appdata/Roaming/Microsoft/Windows/Start Menu/Programs/GOG.com/GameTiles/", "Path for game shortcuts and image data.")
var width = flag.Int("layoutWidth", 3, "Defines the tile count per row in the Start Menu Layout (3 or 4).")
var tileSize = flag.Int("tileSize", 2, "Size of the individual game tiles (1 or 2).")

func main() {
	flag.Parse()

	log.SetFormatter(&log.TextFormatter{ForceColors: true})
	log.SetOutput(ansicolor.NewAnsiColorWriter(os.Stdout))
	level, err := log.ParseLevel(*loglevel)
	if err != nil {
		log.Fatalf("Log level '%s' not recognized. %s", *loglevel, err)
	}
	log.SetLevel(level)

	if runtime.GOOS != "windows" {
		log.Fatal("This App does only support Windows 10!")
	}
	if *width != 3 && *width != 4 {
		log.Fatal("layoutWidth has to be 3 or 4.")
	}
	if *tileSize != 1 && *tileSize != 2 {
		log.Fatal("tileSize has to be 1 or 2.")
	}

	games := listGames()
	createStartMenu(games)
}

func listGames() *map[string]map[string]string {
	log.Info("Reading GOG Galaxy 2.0 database...")
	database, err := sql.Open("sqlite3", *databaseFile+"?mode=ro")
	if err != nil {
		log.Fatalf("Error while trying to open GOG Galaxy 2.0 database at '%s'. %s", *databaseFile, err)
	}
	rows, err := database.Query(sqlTaggedGames)
	if err != nil {
		log.Fatal("Error while running query on database. ", err)
	}
	database.Close()

	log.Info("Parsing games...")
	var games map[string]map[string]string = make(map[string]map[string]string)
	var releaseKey string
	var iconFileName string
	var title string
	for rows.Next() {
		rows.Scan(&releaseKey, &iconFileName, &title)
		extractedTitle := alphanum.ReplaceAllString(strings.Split(title, ":")[1], "")
		alreadyExisting := false
		for _, val := range games {
			if val["title"] == extractedTitle {
				alreadyExisting = true
				break
			}
		}
		if alreadyExisting {
			continue
		}
		if _, ok := games[releaseKey]; !ok {
			games[releaseKey] = make(map[string]string)
		}
		games[releaseKey]["iconFileName"] = strings.ReplaceAll(iconFileName, ".webp", ".png")
		games[releaseKey]["title"] = extractedTitle
	}
	return &games
}

func createStartMenu(games *map[string]map[string]string) {
	log.Info("Creating shortcuts...")
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Could not find users home directory. ", err)
	}
	startMenuPath := home + *startMenuDir
	err = os.MkdirAll(startMenuPath+"VisualElements", os.ModePerm)
	if err != nil {
		log.Fatal("Error while creating Start Menu folder. ", err)
	}

	log.Info("Creating Start Menu layout...")
	partialStartLayout := partialStartLayoutBegin
	tileCount := 0
	actualWidth := *width * (3 - *tileSize)
	for key, val := range *games {
		partialStartLayout += fmt.Sprintf(startLayoutTile,
			*tileSize,
			(tileCount%actualWidth)*(*tileSize),
			(tileCount-(tileCount%actualWidth))/actualWidth*(*tileSize),
			val["title"])
		tileCount++
		writeFile(startMenuPath+key+".bat",
			`"C:\Program Files (x86)\GOG Galaxy\GalaxyClient.exe" /command=runGame /gameId=`+key)
		writeFile(startMenuPath+key+".VisualElementsManifest.xml", fmt.Sprintf(visualElements, key))

		execPowershell(powershellCreateShortcut,
			val["iconFileName"], startMenuPath+"VisualElements\\MediumIcon"+key, startMenuPath+val["title"], startMenuPath+key)
	}

	partialStartLayout += partialStartLayoutEnd
	writeFile("PartialStartLayout.xml", partialStartLayout)

	log.Info("Updating Start Menu...")
	execPowershell(powershellApplyStartLayout)
	log.Info("Program finished!")
}

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
	f, err := os.Create(filePath)
	if err != nil {
		log.Fatalf("Could not open or create file '%s'. %s", filePath, err)
	}
	_, err = f.WriteString(contents)
	if err != nil {
		log.Fatal("Could not write to file. ", err)
	}
	f.Close()
}
