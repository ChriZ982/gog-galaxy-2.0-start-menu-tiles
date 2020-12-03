package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

const dbLocation = "storage/galaxy-2.0.db"
const versionFile = "selfupdate.json"
const supportedVersion = "2.0.25.2"

// Settings holds all passed command line arguments or their default values
type Settings struct {
	startMenuDir string
	groupName    string
	loglevel     string
	tagName      string
	gogDir       string
	tileSize     int
	height       int
	width        int
	hideName     bool
	force        bool
	yes          bool
}

var args Settings

func main() {
	parseArgs()
	configLogger()
	cmdLine := "\"" + strings.Join(os.Args[1:], "\" \"") + "\""
	log.Debug(cmdLine)

	checkArgs()
	checkVersion()

	games := listGames()
	createStartMenu(games)
	updateRegistry()

	saveSettings(cmdLine)
	log.Info("Program finished!")
}

func parseArgs() {
	flag.StringVar(&args.startMenuDir, "startDir", "/Appdata/Roaming/Microsoft/Windows/Start Menu/Programs/GOG.com/GameTiles/", "Path for game shortcuts and image data.")
	flag.StringVar(&args.groupName, "groupName", "", "Name of the Start Menu group.")
	flag.StringVar(&args.loglevel, "level", "INFO", "Defines log level.")
	flag.StringVar(&args.tagName, "tagName", "StartMenuTiles", "Define a custom tag that defines games to be added to the Start Menu. You can also set it to INSTALLED or ALL to add installed or all games to the StartMenu.")
	flag.StringVar(&args.gogDir, "gogDir", "C:/ProgramData/GOG.com/Galaxy/", "Path to GOG Galaxy 2.0 data directory.")
	flag.IntVar(&args.tileSize, "tileSize", 2, "Size of the individual game tiles (1 or 2).")
	flag.IntVar(&args.height, "height", 7, "Defines the rows per group Start Menu Layout.")
	flag.IntVar(&args.width, "width", 3, "Defines the tile count per row in the Start Menu Layout (3 or 4).")
	flag.BoolVar(&args.hideName, "hideName", false, "Show name of game on Start Menu Tile.")
	flag.BoolVar(&args.force, "force", false, "Force re-download of images.")
	flag.BoolVar(&args.yes, "y", false, "Always confirm creation of Start Layout.")
	flag.Parse()
}

func configLogger() {
	log.SetFormatter(&log.TextFormatter{ForceColors: true})

	level, err := log.ParseLevel(args.loglevel)
	if err != nil {
		log.Fatalf("Log level '%s' not recognized. %s", args.loglevel, err)
	}
	log.SetLevel(level)
}

func checkArgs() {
	if runtime.GOOS != "windows" {
		log.Fatal("This App does only support Windows 10!")
	}
	if args.width != 3 && args.width != 4 {
		log.Fatal("width has to be 3 or 4.")
	}
	if args.tileSize != 1 && args.tileSize != 2 {
		log.Fatal("tileSize has to be 1 or 2.")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Could not find users home directory. ", err)
	}
	args.startMenuDir = path.Join(home, args.startMenuDir)
}

func checkVersion() {
	filename := path.Join(args.gogDir, versionFile)
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("Error while opening version file (%s). %s", filename, err)
	}

	var objmap map[string]json.RawMessage
	err = json.Unmarshal(data, &objmap)
	if err != nil {
		log.Fatalf("Error while parsing version file (%s). %s", filename, err)
	}

	var version string
	err = json.Unmarshal(objmap["prefetched__desktop-galaxy-clientVersion"], &version)
	if err != nil {
		log.Fatalf("Error while parsing version file (%s). %s", filename, err)
	}

	if version != supportedVersion {
		log.Warnf("Detected GOG Galaxy version %s. Supported version is %s. You might experience bugs and unwanted behavior!",
			version, supportedVersion)
	}
}

func saveSettings(cmdLine string) {
	exePath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Saving executable file with current parameters to startmenu.")
	writeFile(path.Join(args.startMenuDir, "RUN_StartMenuTiles.bat"), "\""+exePath+"\" "+cmdLine)
	execPowershell("$WScriptShell = New-Object -ComObject WScript.Shell\n" +
		fmt.Sprintf(powershellCreateShortcut, path.Join(args.startMenuDir, "RUN StartMenuTiles again"),
			path.Join(args.startMenuDir, "RUN_StartMenuTiles")))
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func writeFile(filePath string, contents string) {
	err := ioutil.WriteFile(filePath, []byte(contents), 0666)
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
