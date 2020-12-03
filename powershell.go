package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/windows/registry"
)

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

func createStartMenu(games []Game) {
	log.Info("Creating shortcuts...")
	err := os.MkdirAll(path.Join(args.startMenuDir, "VisualElements"), os.ModePerm)
	if err != nil {
		log.Fatal("Error while creating Start Menu folder. ", err)
	}

	log.Info("Creating Start Menu layout...")
	partialStartLayout := fmt.Sprintf(partialStartLayoutBegin, args.groupName)
	maxTiles := args.width * (3 - args.tileSize) * args.height * (3 - args.tileSize)
	actualWidth := args.width * (3 - args.tileSize)
	nameOnLogo := "on"
	if args.hideName {
		nameOnLogo = "off"
	}

	allPowershellCreateShortcut := "$WScriptShell = New-Object -ComObject WScript.Shell\n"
	allPowershellDownloadImage := ""
	tileCount := 0
	for _, game := range games {
		if tileCount >= maxTiles {
			tileCount = 0
			partialStartLayout += fmt.Sprintf(newGroup, args.groupName)
		}
		partialStartLayout += fmt.Sprintf(startLayoutTile, args.tileSize,
			(tileCount%actualWidth)*(args.tileSize),
			(tileCount-(tileCount%actualWidth))/actualWidth*(args.tileSize),
			game.Title)
		tileCount++

		gamePath := path.Join(args.startMenuDir, game.FileName)
		iconPath := path.Join(args.startMenuDir, "VisualElements\\MediumIcon"+game.FileName)
		writeFile(gamePath+".bat", `"C:\Program Files (x86)\GOG Galaxy\GalaxyClient.exe" /command=runGame /gameId=`+game.ReleaseKey)
		writeFile(gamePath+".VisualElementsManifest.xml", fmt.Sprintf(visualElements, game.FileName, nameOnLogo))
		if !fileExists(iconPath+".png") || args.force {
			allPowershellDownloadImage += fmt.Sprintf(powershellDownloadImage+"\n", game.IconFileName, iconPath)
		}
		allPowershellCreateShortcut += fmt.Sprintf(powershellCreateShortcut+"\n", path.Join(args.startMenuDir, game.Title), gamePath)
	}

	if len(allPowershellDownloadImage) > 0 {
		execPowershell(allPowershellDownloadImage)
	}
	execPowershell(allPowershellCreateShortcut)

	partialStartLayout += partialStartLayoutEnd
	writeFile("PartialStartLayout.xml", partialStartLayout)
}

func updateRegistry() {
	time.Sleep(8 * time.Second) // Wait for shortcuts to be recoginzed
	log.Info("Updating Start Menu...")
	execPowershell("Export-StartLayout -Path StartLayoutBackup.xml")

	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Policies\Microsoft\Windows\Explorer`, registry.ALL_ACCESS)
	defer key.Close()
	if err != nil {
		log.Fatal("Could not create Registry Key. You might have to run the program with admin rights. ", err)
	}

	values, err := key.ReadValueNames(0)
	if err != nil {
		log.Fatal("Could not list Registry values. You might have to run the program with admin rights.")
	}
	if find(values, "StartLayoutFile") || find(values, "LockedStartLayout") {
		log.Warn("Registry Value 'StartLayoutFile' or 'LockedStartLayout' exists. There might have been a Start Layout previously applied! This would be removed entirely!")
	}
	if !args.yes {
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

	err = key.SetExpandStringValue("StartLayoutFile", wd+"\\PartialStartLayout.xml")
	err2 := key.SetDWordValue("LockedStartLayout", 1)
	if err != nil || err2 != nil {
		log.Fatal("Could not set registry value. You might have to run the program with admin rights. ", err)
	}

	execPowershell("Stop-Process -ProcessName explorer")
	time.Sleep(5 * time.Second)
	err = key.SetDWordValue("LockedStartLayout", 0)
	if err != nil {
		log.Fatal("Could not set registry value. You might have to run the program with admin rights. ", err)
	}

	execPowershell("Stop-Process -ProcessName explorer")
	time.Sleep(3 * time.Second)
	err = key.DeleteValue("StartLayoutFile")
	err2 = key.DeleteValue("LockedStartLayout")
	if err != nil || err2 != nil {
		log.Fatal("Could not delete registry value. You might have to run the program with admin rights. ", err)
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
