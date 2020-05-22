package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"runtime"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/shiena/ansicolor"
	log "github.com/sirupsen/logrus"
)

var disallowedChars = regexp.MustCompile("[^A-Za-z0-9 +_#!()=-]+")

const dbLocation = "storage/galaxy-2.0.db"
const versionFile = "selfupdate.json"
const supportedVersion = "2.0.15.43"

var loglevel = flag.String("level", "INFO", "Defines log level.")
var gogDir = flag.String("database", "C:/ProgramData/GOG.com/Galaxy/", "Path to GOG Galaxy 2.0 database.")
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
	cmdLine := strings.Join(os.Args[1:], " ")
	log.Debug(cmdLine)

	checkParams()
	checkVersion()

	games := listGames()
	createStartMenu(games)
	updateRegistry()

	saveSettings(cmdLine)
	log.Info("Program finished!")
}

func checkParams() {
	if runtime.GOOS != "windows" {
		log.Fatal("This App does only support Windows 10!")
	}
	if *width != 3 && *width != 4 {
		log.Fatal("layoutWidth has to be 3 or 4.")
	}
	if *tileSize != 1 && *tileSize != 2 {
		log.Fatal("tileSize has to be 1 or 2.")
	}
	if !strings.HasSuffix(*gogDir, "/") && !strings.HasSuffix(*gogDir, "\\") {
		*gogDir = *gogDir + "/"
	}
	if !strings.HasPrefix(*startMenuDir, "/") && !strings.HasPrefix(*startMenuDir, "\\") {
		*startMenuDir = "/" + *startMenuDir
	}
	if !strings.HasSuffix(*startMenuDir, "/") && !strings.HasSuffix(*startMenuDir, "\\") {
		*startMenuDir = *startMenuDir + "/"
	}
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Could not find users home directory. ", err)
	}
	*startMenuDir = home + *startMenuDir
}

func checkVersion() {
	data, err := ioutil.ReadFile(*gogDir + versionFile)
	if err != nil {
		log.Fatalf("Error while opening version file (%s). %s", *gogDir+versionFile, err)
	}
	var objmap map[string]json.RawMessage
	err = json.Unmarshal(data, &objmap)
	if err != nil {
		log.Fatalf("Error while parsing version file (%s). %s", *gogDir+versionFile, err)
	}
	var version string
	err = json.Unmarshal(objmap["desktop-galaxy-clientVersion"], &version)
	if err != nil {
		log.Fatalf("Error while parsing version file (%s). %s", *gogDir+versionFile, err)
	}
	if version != supportedVersion {
		log.Warnf("Detected GOG Galaxy version %s. Supported version is %s. You might experience bugs and unwanted behavior!", version, supportedVersion)
	}
}

func saveSettings(cmdLine string) {
	batchPath := *startMenuDir + "RUN_StartMenuTiles.bat"
	ex, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Saving executable file with current parameters to startmenu.")
	writeFile(batchPath, "\""+ex+"\" "+cmdLine)
	createShortcut := "$WScriptShell = New-Object -ComObject WScript.Shell\n"
	createShortcut += fmt.Sprintf(powershellCreateShortcut, *startMenuDir+"RUN StartMenuTiles again", *startMenuDir+"RUN_StartMenuTiles")
	execPowershell(createShortcut)
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func writeFile(filePath string, contents string) {
	f, err := os.Create(filePath)
	defer f.Close()
	if err != nil {
		log.Fatalf("Could not open or create file '%s'. %s", filePath, err)
	}
	_, err = f.WriteString(contents)
	if err != nil {
		log.Fatal("Could not write to file. ", err)
	}
}

func find(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
