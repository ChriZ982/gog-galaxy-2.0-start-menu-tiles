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
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/shiena/ansicolor"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/windows/registry"
)

const sqlTaggedGames = `SELECT UserReleaseTags.releaseKey, WebCacheResources.filename, GamePieces.value FROM UserReleaseTags
LEFT JOIN WebCacheResources ON UserReleaseTags.releaseKey = WebCacheResources.releaseKey
LEFT JOIN GamePieces ON UserReleaseTags.releaseKey = GamePieces.releaseKey
WHERE UserReleaseTags.tag = ? AND WebCacheResources.webCacheResourceTypeId = 2 AND GamePieces.gamePieceTypeId = 11 AND GamePieces.userId <> 0`

const sqlInstalledGames = `SELECT Installed.releaseKey, WebCacheResources.filename, GamePieces.value FROM
	(SELECT 'gog_' || productId as releaseKey FROM InstalledBaseProducts
	UNION ALL
	SELECT Platforms.name || '_' || productId as releaseKey FROM InstalledExternalProducts
	JOIN Platforms ON InstalledExternalProducts.platformId = Platforms.id) as Installed
LEFT JOIN WebCacheResources ON Installed.releaseKey = WebCacheResources.releaseKey
LEFT JOIN GamePieces ON Installed.releaseKey = GamePieces.releaseKey
WHERE WebCacheResources.webCacheResourceTypeId = 2 AND GamePieces.gamePieceTypeId = 11 AND GamePieces.userId <> 0`

const sqlAllGames = `SELECT OwnedGames.releaseKey, WebCacheResources.filename, GamePieces.value FROM OwnedGames
LEFT JOIN WebCacheResources ON OwnedGames.releaseKey = WebCacheResources.releaseKey
LEFT JOIN GamePieces ON OwnedGames.releaseKey = GamePieces.releaseKey
WHERE WebCacheResources.webCacheResourceTypeId = 2 AND GamePieces.gamePieceTypeId = 11 AND OwnedGames.userId <> 0 AND GamePieces.userId <> 0`

// Uses a partial Start Layout (https://docs.microsoft.com/en-us/windows/configuration/customize-and-export-start-layout#configure-a-partial-start-layout)
const partialStartLayoutBegin = `<LayoutModificationTemplate xmlns:defaultlayout="http://schemas.microsoft.com/Start/2014/FullDefaultLayout" xmlns:start="http://schemas.microsoft.com/Start/2014/StartLayout" Version="1" xmlns="http://schemas.microsoft.com/Start/2014/LayoutModification">
  <LayoutOptions StartTileGroupCellWidth="8" />
  <DefaultLayoutOverride LayoutCustomizationRestrictionType="OnlySpecifiedGroups">
    <StartLayoutCollection>
      <defaultlayout:StartLayout GroupCellWidth="8">
        <start:Group Name="%s">`

const startLayoutTile = `
          <start:DesktopApplicationTile Size="%[1]dx%[1]d" Column="%[2]d" Row="%[3]d" DesktopApplicationLinkPath="%%APPDATA%%\Microsoft\Windows\Start Menu\Programs\GOG.com\GameTiles\%[4]s.lnk" />`

const partialStartLayoutEnd = `
        </start:Group>
      </defaultlayout:StartLayout>
    </StartLayoutCollection>
  </DefaultLayoutOverride>
</LayoutModificationTemplate>`

const newGroup = `
        </start:Group>
        <start:Group Name="%s">`

const powershellDownloadImage = `Invoke-WebRequest -Uri https://images.gog.com/%s?namespace=gamesdb -OutFile "%s.png"`

const powershellCreateShortcut = `$Shortcut = $WScriptShell.CreateShortcut("%s.lnk")
$Shortcut.TargetPath = "%s.bat"
$Shortcut.Save()`

const visualElements = `<?xml version="1.0" encoding="utf-8"?>
<Application xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
	<VisualElements ShowNameOnSquare150x150Logo="%[2]s" Square150x150Logo="VisualElements\MediumIcon%[1]s.png" Square70x70Logo="VisualElements\MediumIcon%[1]s.png" ForegroundText="light" BackgroundColor="#5A391B" />
</Application>`

var disallowedChars = regexp.MustCompile("[^A-Za-z0-9 +_#!()=-]+")

var loglevel = flag.String("level", "INFO", "Defines log level.")
var databaseFile = flag.String("database", "C:/ProgramData/GOG.com/Galaxy/storage/galaxy-2.0.db", "Path to GOG Galaxy 2.0 database.")
var startMenuDir = flag.String("startFolder", "/Appdata/Roaming/Microsoft/Windows/Start Menu/Programs/GOG.com/GameTiles/", "Path for game shortcuts and image data.")
var width = flag.Int("width", 3, "Defines the tile count per row in the Start Menu Layout (3 or 4).")
var height = flag.Int("height", 7, "Defines the rows per group Start Menu Layout.")
var tileSize = flag.Int("tileSize", 2, "Size of the individual game tiles (1 or 2).")
var yes = flag.Bool("y", false, "Always confirm creation of Start Layout.")
var groupName = flag.String("groupName", "", "Name of the Start Menu group.")
var tagName = flag.String("tagName", "StartMenuTiles", "Define a custom tag that defines games to be added to the Start Menu. You can also set it to INSTALLED or ALL to add installed or all games to the StartMenu.")
var hideName = flag.Bool("hideName", false, "Show name of game on Start Menu Tile.")
var force = flag.Bool("force", false, "Force re-download of images.")

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
	updateRegistry()

	log.Info("Program finished!")
}

func listGames() *map[string]map[string]string {
	log.Info("Reading GOG Galaxy 2.0 database...")
	database, err := sql.Open("sqlite3", *databaseFile+"?mode=ro")
	if err != nil {
		log.Fatalf("Error while trying to open GOG Galaxy 2.0 database at '%s'. %s", *databaseFile, err)
	}
	var rows *sql.Rows
	if *tagName == "INSTALLED" {
		rows, err = database.Query(sqlInstalledGames)
	} else if *tagName == "ALL" {
		rows, err = database.Query(sqlAllGames)
	} else {
		rows, err = database.Query(sqlTaggedGames, *tagName)
	}
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
		extractedTitle := disallowedChars.ReplaceAllString(strings.Split(title, ":")[1], "")
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
	if len(games) == 0 {
		log.Fatal("No games found.")
	}
	if len(games) > 150 {
		log.Fatalf("Adding too many tiles causes unexpected behaviour. %d tiles can not be created safely.", len(games))
	}
	if len(games) > 80 {
		log.Warnf("Adding too many tiles causes unexpected behaviour. %d tiles will be added.", len(games))
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
	partialStartLayout := fmt.Sprintf(partialStartLayoutBegin, *groupName)
	tileCount := 0
	maxTiles := *width * (3 - *tileSize) * *height * (3 - *tileSize)
	actualWidth := *width * (3 - *tileSize)
	nameOnLogo := "on"
	if *hideName {
		nameOnLogo = "off"
	}
	allPowershellCreateShortcut := "$WScriptShell = New-Object -ComObject WScript.Shell\n"
	allPowershellDownloadImage := ""
	for key, val := range *games {
		partialStartLayout += fmt.Sprintf(startLayoutTile,
			*tileSize,
			(tileCount%actualWidth)*(*tileSize),
			(tileCount-(tileCount%actualWidth))/actualWidth*(*tileSize),
			val["title"])
		tileCount++
		if tileCount >= maxTiles {
			tileCount = 0
			partialStartLayout += fmt.Sprintf(newGroup, *groupName)
		}
		plainKey := disallowedChars.ReplaceAllString(key, "")
		writeFile(startMenuPath+plainKey+".bat", `"C:\Program Files (x86)\GOG Galaxy\GalaxyClient.exe" /command=runGame /gameId=`+key)
		writeFile(startMenuPath+plainKey+".VisualElementsManifest.xml", fmt.Sprintf(visualElements, plainKey, nameOnLogo))
		if !fileExists(startMenuPath+"VisualElements\\MediumIcon"+plainKey+".png") || *force {
			allPowershellDownloadImage += fmt.Sprintf(powershellDownloadImage+"\n", val["iconFileName"], startMenuPath+"VisualElements\\MediumIcon"+plainKey)
		}
		allPowershellCreateShortcut += fmt.Sprintf(powershellCreateShortcut+"\n", startMenuPath+val["title"], startMenuPath+plainKey)
	}
	if len(allPowershellDownloadImage) > 0 {
		execPowershell(allPowershellDownloadImage)
	}
	execPowershell(allPowershellCreateShortcut)

	partialStartLayout += partialStartLayoutEnd
	writeFile("PartialStartLayout.xml", partialStartLayout)
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func updateRegistry() {
	time.Sleep(8 * time.Second) // Wait for shortcuts to be recoginzed
	log.Info("Updating Start Menu...")
	execPowershell("Export-StartLayout -Path StartLayoutBackup.xml")
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Policies\Microsoft\Windows\Explorer`, registry.ALL_ACCESS)
	if err != nil {
		log.Fatal("Could not create Registry Key. ", err)
	}
	defer key.Close()
	values, err := key.ReadValueNames(0)
	if err != nil {
		log.Fatal("Could not list Registry values")
	}
	if find(values, "StartLayoutFile") || find(values, "LockedStartLayout") {
		log.Warn("Registry Value 'StartLayoutFile' or 'LockedStartLayout' exists. There might have been a Start Layout previously applied! This would be removed entirely!")
	}
	if !*yes {
		log.Warn("The script will now create registry values to modify the Start Menu. The groups in your Start Menu will probably be reordered. If there was a custom Start Layout .xml file applied before, all tiles will be removed! Use at own risk!\nDo you want to proceed? [yN]")
		var proceed string
		fmt.Scanln(&proceed)
		if proceed != "y" {
			log.Info("Script cancelled by user.")
			return
		}
	}
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal("Not able to get working directory. ", err)
	}
	err1 := key.SetExpandStringValue("StartLayoutFile", wd+"\\PartialStartLayout.xml")
	err2 := key.SetDWordValue("LockedStartLayout", 1)
	if err1 != nil || err2 != nil {
		log.Fatal("Could not set registry value. ", err)
	}
	execPowershell("Stop-Process -ProcessName explorer")
	time.Sleep(5 * time.Second)
	err = key.SetDWordValue("LockedStartLayout", 0)
	if err != nil {
		log.Fatal("Could not set registry value. ", err)
	}
	execPowershell("Stop-Process -ProcessName explorer")
	time.Sleep(3 * time.Second)
	err3 := key.DeleteValue("StartLayoutFile")
	err4 := key.DeleteValue("LockedStartLayout")
	if err3 != nil || err4 != nil {
		log.Fatal("Could not delete registry value. ", err)
	}
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
			log.Error(strings.TrimRight(line, " "))
			log.Warn("Script: ", fmt.Sprintf(cmdText, args...))
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

func find(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
