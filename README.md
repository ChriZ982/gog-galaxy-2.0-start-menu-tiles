# <img height="25" src="https://simpleicons.org/icons/gog-dot-com.svg"/> GOG Galaxy 2.0 Start Menu Tiles
[![GOG Galaxy 2.0](https://img.shields.io/badge/GOG-Galaxy%202.0-86328A?logo=data:https://simpleicons.org/icons/gog-dot-com.svg)](https://www.gogalaxy.com/en/) [![Build status](https://travis-ci.org/ChriZ982/gog-galaxy-2.0-start-menu-tiles.svg?branch=master)](https://travis-ci.org/github/ChriZ982/gog-galaxy-2.0-start-menu-tiles/branches) [![GitHub release (latest by date)](https://img.shields.io/github/v/release/ChriZ982/gog-galaxy-2.0-start-menu-tiles)](https://github.com/ChriZ982/gog-galaxy-2.0-start-menu-tiles/releases) ![GitHub stars](https://img.shields.io/github/stars/ChriZ982/gog-galaxy-2.0-start-menu-tiles) ![GitHub top language](https://img.shields.io/github/languages/top/ChriZ982/gog-galaxy-2.0-start-menu-tiles) [![PayPal.Me ChriZ98](https://img.shields.io/badge/PayPal.Me-ChriZ98-00457C?logo=paypal)](https://www.paypal.me/ChriZ98)

This script lets you create Start Menu Tiles of your favourite games in Windows 10! :video_game:

Simply download the `GOG_Galaxy_Start_Menu.exe` from the Releases section. Your GOG Galaxy 2.0 database will be read and a shortcut will be created in the Programs section. Additionally tiles can be automatically added to the Start Menu, providing a stylish and easy access.

**Disclaimer: If you have applied a Start Menu Layout previosly, all changes will be reverted and all tiles will be deleted! This can be the case for computers managed by organizations.**

#### Examples:
<table>
  <tr>
    <td><img alt="Startmenu Picture 1" src="examples/startmenu1.jpeg" /></td>
    <td><img alt="Startmenu Picture 2" src="examples/startmenu2.jpeg" /></td>
  </tr>
  <tr>
    <td><img alt="Startmenu Picture 3" src="examples/startmenu3.jpeg" /></td>
    <td></td>
  </tr>
</table>

## :sparkles: Planned Features
* [x] Add build pipeline
* [ ] Remember tiles and remove old image files
* [ ] Possibility to choose different icon image source
* [x] Add custom Start Menu group name setting
* [x] Test whether registry folder exists

## :hammer_and_wrench: Usage
```
Usage of GOG_Galaxy_Start_Menu.exe:
  -database string
        Path to GOG Galaxy 2.0 database. (default "C:/ProgramData/GOG.com/Galaxy/storage/galaxy-2.0.db")
  -force
        Force re-download of images.
  -groupName string
        Name of the Start Menu group.
  -height int
        Defines the rows per group Start Menu Layout. (default 7)
  -hideName
        Show name of game on Start Menu Tile.
  -level string
        Defines log level. (default "INFO")
  -startFolder string
        Path for game shortcuts and image data. (default "/Appdata/Roaming/Microsoft/Windows/Start Menu/Programs/GOG.com/GameTiles/")
  -tagName string
        Define a custom tag that defines games to be added to the Start Menu. You can also set it to INSTALLED or ALL to add installed or all games to the StartMenu. (default "StartMenuTiles")
  -tileSize int
        Size of the individual game tiles (1 or 2). (default 2)
  -width int
        Defines the tile count per row in the Start Menu Layout (3 or 4). (default 3)
  -y    Always confirm creation of Start Layout.
```

## :earth_africa: Contributing
If you find any issues or have some improvement ideas, please [create an issue](../../issues/new/choose). Also feel free to fork the repo and create a pull request when you have finished your implementation. :page_with_curl:

If your feature is a good addition to the project, it will be merged!

## :sparkling_heart: Support my projects
If you like the project and you want to support me - please consider to gift using the button below.

[![PayPal.Me ChriZ98](https://img.shields.io/badge/PayPal.Me-ChriZ98-00457C?logo=paypal)](https://www.paypal.me/ChriZ98)

Thanks! :heart:

## :scroll: License
<table>
  <tr>
    <td><a rel="license" href="http://creativecommons.org/licenses/by-nc-sa/4.0/"><img alt="Creative Commons License" style="border-width:0" width="160px" src="https://i.creativecommons.org/l/by-nc-sa/4.0/88x31.png" /></a></td>
    <td><span xmlns:dct="http://purl.org/dc/terms/" href="http://purl.org/dc/dcmitype/Text" property="dct:title" rel="dct:type">GOG Galaxy 2.0 Start Menu Tiles</span> by <a xmlns:cc="http://creativecommons.org/ns#" href="https://github.com/ChriZ982" property="cc:attributionName" rel="cc:attributionURL">ChriZ982</a> is licensed under a <a rel="license" href="http://creativecommons.org/licenses/by-nc-sa/4.0/">Creative Commons Attribution-NonCommercial-ShareAlike 4.0 International License</a>.</td>
  </tr>
</table>
