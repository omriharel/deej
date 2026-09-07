# deej

deej is an **open-source hardware volume mixer** for Windows and Linux PCs. It lets you use real-life sliders (like a DJ!) to **seamlessly control the volumes of different apps** (such as your music player, the game you're playing and your voice chat session) without having to stop what you're doing.

**Join the [deej Discord server](https://discord.gg/nf88NJu) if you need help or have any questions!**

[![Discord](https://img.shields.io/discord/702940502038937667?logo=discord)](https://discord.gg/nf88NJu)

> **_New:_** [work-in-progress deej FAQ](./docs/faq/faq.md)!

deej consists of a [lightweight desktop client](#features) written in Go, and an Arduino-based hardware setup that's simple and cheap to build. [**Check out some versions built by members of our community!**](./community.md)

**[Download the latest release](https://github.com/omriharel/deej/releases/latest) | [Video demonstration](https://youtu.be/VoByJ4USMr8) | [Build video by Tech Always](https://youtu.be/x2yXbFiiAeI)**

![deej](assets/build-3d-annotated.png)

> _**Psst!** [No 3D printer? No problem!](./assets/build-shoebox.jpg)_ You can build deej on some cardboard, a shoebox or even a breadboard :)

## Table of contents

- [Features](#features)
- [How it works](#how-it-works)
  - [Hardware](#hardware)
    - [Schematic](#schematic)
  - [Software](#software)
- [Slider mapping (configuration)](#slider-mapping-configuration)
- [Build your own!](#build-your-own)
  - [FAQ](#faq)
  - [Build video](#build-video)
  - [Bill of Materials](#bill-of-materials)
  - [Thingiverse collection](#thingiverse-collection)
  - [Build procedure](#build-procedure)
- [How to run](#how-to-run)
  - [Requirements](#requirements)
  - [Download and installation](#download-and-installation)
  - [Building from source](#building-from-source)
- [Community](#community)
- [License](#license)

## Features

deej is written in Go and [distributed](https://github.com/omriharel/deej/releases/latest) as a portable (no installer needed) executable.

- Bind apps to different sliders
  - Bind multiple apps per slider (i.e. one slider for all your games)
  - Bind the master channel
  - Bind "system sounds" (on Windows)
  - Bind specific audio devices by name (on Windows)
  - Bind currently active app (on Windows)
  - Bind all other unassigned apps
- Control your microphone's input level
- Lightweight desktop client, consuming around 10MB of memory
- Runs from your system tray
- Helpful notifications to let you know if something isn't working

> **Looking for the older Python version?** It's no longer maintained, but you can always find it in the [`legacy-python` branch](https://github.com/omriharel/deej/tree/legacy-python).

## How it works

### Hardware

- The sliders are connected to 5 (or as many as you like) analog pins on an Arduino Nano/Uno board. They're powered from the board's 5V output (see schematic)
- The board connects via a USB cable to the PC

#### Schematic

![Hardware schematic](assets/schematic.png)

### Software

- The code running on the Arduino board is a [C program](./arduino/deej-5-sliders-vanilla/deej-5-sliders-vanilla.ino) constantly writing current slider values over its serial interface
- The PC runs a lightweight [Go client](./pkg/deej/cmd/main.go) in the background. This client reads the serial stream and adjusts app volumes according to the given configuration file

## Slider mapping (configuration)

deej uses a simple YAML-formatted configuration file named [`config.yaml`](./config.yaml), placed alongside the deej executable.

The config file determines which applications (and devices) are mapped to which sliders, and which parameters to use for the connection to the Arduino board, as well as other user preferences.

**This file auto-reloads when its contents are changed, so you can change application mappings on-the-fly without restarting deej.**

It looks like this:

```yaml
slider_mapping:
  0: master
  1: chrome.exe
  2: spotify.exe
  3:
    - pathofexile_x64.exe
    - rocketleague.exe
  4: discord.exe

# set this to true if you want the controls inverted (i.e. top is 0%, bottom is 100%)
invert_sliders: false

# settings for connecting to the arduino board
com_port: COM4
baud_rate: 9600

# adjust the amount of signal noise reduction depending on your hardware quality
# supported values are "low" (excellent hardware), "default" (regular hardware) or "high" (bad, noisy hardware)
noise_reduction: default
```

- `master` is a special option to control the master volume of the system _(uses the default playback device)_
- `mic` is a special option to control your microphone's input level _(uses the default recording device)_
- `deej.unmapped` is a special option to control all apps that aren't bound to any slider ("everything else")
- On Windows, `deej.current` is a special option to control whichever app is currently in focus
- On Windows, you can specify a device's full name, i.e. `Speakers (Realtek High Definition Audio)`, to bind that device's level to a slider. This doesn't conflict with the default `master` and `mic` options, and works for both input and output devices.
  - Be sure to use the full device name, as seen in the menu that comes up when left-clicking the speaker icon in the tray menu
- `system` is a special option on Windows to control the "System sounds" volume in the Windows mixer
- All names are case-**in**sensitive, meaning both `chrome.exe` and `CHROME.exe` will work
- You can create groups of process names (using a list) to either:
    - control more than one app with a single slider
    - choose whichever process in the group that's currently running (i.e. to have one slider control any game you're playing)

## Build your own!

Building deej is very simple. You only need a few relatively cheap parts - it's an excellent starter project (and my first Arduino project, personally). Remember that if you need any help or have a question that's not answered here, you can always [join the deej Discord server](https://discord.gg/nf88NJu).

Build deej for yourself, or as an awesome gift for your gaming buddies!

### FAQ

I've started a highly focused effort of writing a proper FAQ page for deej, covering many basic and advanced topics.

It is still _very much a work-in-progress_, but I'm happy to [share it in its current state](./docs/faq/faq.md) in hopes that it at least covers some questions you might have.

FAQ feedback in our [community Discord](https://discord.gg/nf88NJu) is strongly encouraged :)

### Build video

In case you prefer watching to reading, Charles from the [**Tech Always**](https://www.youtube.com/c/TechAlways) YouTube channel has made [**a fantastic video**](https://youtu.be/x2yXbFiiAeI) that covers the basics of building deej for yourself, including parts, costs, assembly and software. I highly recommend checking it out!

### Bill of Materials

- An Arduino Nano, Pro Micro or Uno board
  - I officially recommend using a Nano or a Pro Micro for their smaller form-factor, friendlier USB connectors and more analog pins. Plus they're cheaper
  - You can also use any other development board that has a Serial over USB interface
- A few slider potentiometers, up to your number of free analog pins (the cheaper ones cost around 1-2 USD each, and come with a standard 10K Ohm variable resistor. These _should_ work just fine for this project)
  - **Important:** make sure to get **linear** sliders, not logarithmic ones! Check the product description
  - You can also use circular knobs if you like
- Some wires
- Any kind of box to hold everything together. **You don't need a 3D printer for this project!** It works fantastically with just a piece of cardboard or a shoebox. That being said, if you do have one, read on...

### Thingiverse collection

With many different 3D-printed designs being added to our [community showcase](./community.md), it felt right to gather all of them in a Thingiverse collection for you to browse. If you have access to a 3D printer, feel free to use one of the designs in your build.

**[Visit our community-created design collection on Thingiverse!](https://thingiverse.com/omriharel/collections/deej)**

> You can also [submit your own](https://discord.gg/nf88NJu) design to be added to the collection. Regardless, if you do upload your design to Thingiverse, _please add a `deej` tag to it so that others can find it more easily_.


### Build procedure

- Connect everything according to the [schematic](#schematic)
- Test with a multimeter to be sure your sliders are hooked up correctly
- Flash the Arduino chip with the sketch in [`arduino\deej-5-sliders-vanilla`](./arduino/deej-5-sliders-vanilla/deej-5-sliders-vanilla.ino)
  - _Important:_ If you have more or less than 5 sliders, you must edit the sketch to match what you have
- After flashing, check the serial monitor. You should see a constant stream of values separated by a pipe (`|`) character, e.g. `0|240|1023|0|483`
  - When you move a slider, its corresponding value should move between 0 and 1023
- Congratulations, you're now ready to run the deej executable!

## How to run

### Requirements

#### Windows

- Windows. That's it

#### Linux

- Install `libgtk-3-dev`, `libappindicator3-dev` and `libwebkit2gtk-4.0-dev` for system tray support. Pre-built Linux binaries aren't currently released, so you'll need to [build from source](#building-from-source). If there's demand for pre-built binaries, please [let me know](https://discord.gg/nf88NJu)!

### Download and installation

- Head over to the [releases page](https://github.com/omriharel/deej/releases) and download the [latest version](https://github.com/omriharel/deej/releases/latest)'s executable and configuration file (`deej.exe` and `config.yaml`)
- Place them in the same directory anywhere on your machine
- (Optional, on Windows) Create a shortcut to `deej.exe` and copy it to `%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup` to have deej run on boot

### Building from source

If you'd rather not download a compiled executable, or want to extend deej or modify it to your needs, feel free to clone the repository and build it yourself. All you need is a Go 1.14 (or above) environment on your machine. If you go this route, make sure to check out the [developer scripts](./pkg/deej/scripts).

Like other Go packages, you can also use the `go get` tool: `go get -u github.com/omriharel/deej`. Please note that the package code now resides in the `pkg/deej` directory, and needs to be imported from there if used inside another project.

If you need any help with this, please [join our Discord server](https://discord.gg/nf88NJu).

## Community

[![Discord](https://img.shields.io/discord/702940502038937667?logo=discord)](https://discord.gg/nf88NJu)

deej is a relatively new project, but a vibrant and awesome community is rapidly growing around it. Come hang out with us in the [deej Discord server](https://discord.gg/nf88NJu), or check out a whole bunch of cool and creative builds made by our members in the [community showcase](./community.md).

The server is also a great place to ask questions, suggest features or report bugs (but of course, feel free to use GitHub if you prefer).

### Donations

If you love deej and want to show your support for this project, you can do so using the link below. Please don't feel obligated to donate - building the project and telling your friends about it goes a very long way! Thank you very much.

[![ko-fi](https://www.ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/omriharel)

### Contributing

Please see [`docs/CONTRIBUTING.md`](./docs/CONTRIBUTING.md).

## License

deej is released under the [MIT license](./LICENSE).


## 🌐 Web Resources & Interactive Index
- [PHOTO BLOCK JOURNEY](https://eduquestses.pages.dev/photo-block-journey.html)
- [CATEGORY MOBILE2 112](https://eduquestkr.pages.dev/category-mobile2-112.html)
- [MUSCLE CHALLENGE](https://eduquestses.pages.dev/muscle-challenge.html)
- [PURRFECT SCOOPS](https://brainquestskr.pages.dev/purrfect-scoops.html)
- [LAST TO LEAVE CIRCLE OBBY](https://quizzesarena.onrender.com/last-to-leave-circle-obby.html)
- [DIRTY MONEY THE RICH GET RICH](https://eduquestkr.pages.dev/dirty-money-the-rich-get-rich.html)
- [EMERGENCY JAM](https://quizzesarena.github.io/emergency-jam.html)
- [DEEP FISHING](https://eduquestses.pages.dev/deep-fishing.html)
- [RUNIC BLOCK COLLAPSE](https://quizzesarena.github.io/runic-block-collapse.html)
- [ROYAL GARDEN MATCH 2](https://eduquestses.pages.dev/royal-garden-match-2.html)
- [DINO HUNTER KING](https://eduquestses.pages.dev/dino-hunter-king.html)
- [TOY CLAW SIMULATOR](https://eduquestses.pages.dev/toy-claw-simulator.html)
- [CHICKEN WILD RUN](https://eduquestses.pages.dev/chicken-wild-run.html)
- [WAVE DASH GEOMETRY ARROW](https://quizzesarena.github.io/wave-dash-geometry-arrow.html)
- [TRANSFORMERS BATTLE FOR THE CITY](https://quizzesarena.github.io/transformers-battle-for-the-city.html)
- [ITALIAN BRAINROT BOMB 2PLAYER](https://quizzesarena.github.io/italian-brainrot-bomb-2player.html)
- [HUGGY WUGGY ESCAPE](https://quizzesarena.github.io/huggy-wuggy-escape.html)
- [REVERSI](https://eduquestses.pages.dev/reversi.html)
- [BACTERIA LIFE DEATH](https://eduquestses.pages.dev/bacteria-life-death.html)
- [PET RUNNER](https://eduquestses.pages.dev/pet-runner.html)
- [SUDO TETROID DAILY](https://eduquestses.pages.dev/sudo-tetroid-daily.html)
- [COSMO PET STARRY CARE](https://eduquestses.pages.dev/cosmo-pet-starry-care.html)
- [CITY BIKE RACING CHAMPION](https://quizzesarena.github.io/city-bike-racing-champion.html)
- [XYTRIAN RUNNER](https://quizzesarena.github.io/xytrian-runner.html)
- [TROLL THE TEACHER](https://quizzesarena.onrender.com/troll-the-teacher.html)
- [MINERS FURY](https://quizzesarena.github.io/miners-fury.html)
- [POTTERY MASTER](https://quizzesarena.github.io/pottery-master.html)
- [LUDO STAR](https://eduquestses.pages.dev/ludo-star.html)
- [BEACH CLUB](https://quizzesarena.github.io/beach-club.html)
- [CATEGORY WORLD CUP17](https://quizzesarena.onrender.com/category-world-cup17.html)
- [SAMURAI LEGACY](https://eduquestkr.pages.dev/samurai-legacy.html)
- [PLANT GIRL DEFENSE ZOMBIE](https://quizzesarena.github.io/plant-girl-defense-zombie.html)
- [AXE THROW](https://quizzesarena.github.io/axe-throw.html)
- [CATEGORY FLASH 2](https://quizzesarena.onrender.com/category-flash-2.html)
- [2048 MERGE CIRCLE](https://eduquestses.pages.dev/2048-merge-circle.html)
- [BOXING FIGHTER](https://quizzesarena.onrender.com/boxing-fighter.html)
- [HALLOWEEN CHALLENGE](https://eduquestses.pages.dev/halloween-challenge.html)
- [MINE FPS SHOOTER NOOB ARENA](https://quizzesarena.github.io/mine-fps-shooter-noob-arena.html)
- [FRUIT NINJA](https://eduquestses.pages.dev/fruit-ninja.html)
- [CATEGORY ARCHERY52](https://quizzesarena.onrender.com/category-archery52.html)
- [COLOR COCKTAIL](https://quizzesarena.github.io/color-cocktail.html)
- [TIMBERLAND ARRANGE PUZZLE GAME](https://quizzesarena.github.io/timberland-arrange-puzzle-game.html)
- [LOOP GHOST](https://quizzesarena.github.io/loop-ghost.html)
- [SOLITAIRE MAHJONG FARM 2](https://eduquestses.pages.dev/solitaire-mahjong-farm-2.html)
- [CATEGORY COLOR197](https://quizzesarena.github.io/category-color197.html)
- [HIDDEN KITTY](https://eduquestsjp.pages.dev/hidden-kitty.html)
- [SPOOKY CHAINS](https://eduquestkr.pages.dev/spooky-chains.html)
- [ANIMAL TRANSFORM RACE](https://quizzesarena.github.io/animal-transform-race.html)
- [CATEGORY 2D1 060](https://quizzesarena.onrender.com/category-2d1-060.html)
- [LITTLE ALCHEMY 2](https://quizzesarena.github.io/little-alchemy-2.html)
- [BUBBLY LAB](https://eduquestses.pages.dev/bubbly-lab.html)
- [MAGIC SOLITAIRE](https://quizzesarena.github.io/magic-solitaire.html)
- [HIDDEN OBJECTS STORY](https://quizzesarena.github.io/hidden-objects-story.html)
- [JUST DICE RANDOM TOWER DEFENCE](https://eduquestsjp.pages.dev/just-dice-random-tower-defence.html)
- [SUSHI PUZZLE](https://eduquestses.pages.dev/sushi-puzzle.html)
- [TILES MATCHING](https://quizzesarena.github.io/tiles-matching.html)
- [BLOCK BLAST 2048](https://eduquestses.pages.dev/block-blast-2048.html)
- [INDEX18](https://quizzesarena.onrender.com/index18.html)
- [BOMBER FRIENDS](https://eduquestsjp.pages.dev/bomber-friends.html)
- [ZOMBIE SIEGEIO](https://eduquestses.pages.dev/zombie-siegeio.html)
- [JEWEL LINK](https://eduquestses.pages.dev/jewel-link.html)
- [MUSKETEERS GUNPOWDER VS STEEL](https://eduquestkr.pages.dev/musketeers-gunpowder-vs-steel.html)
- [JEWELS BLITZ LEGENDS](https://quizzesarena.onrender.com/jewels-blitz-legends.html)
- [FAR ORION NEW WORLDS](https://eduquestses.pages.dev/far-orion-new-worlds.html)
- [BUBBLE SHOOTER HAWAII](https://eduquestses.pages.dev/bubble-shooter-hawaii.html)
- [CONSTRUCTION SET 3D BUILDER](https://eduquestsjp.pages.dev/construction-set-3d-builder.html)
- [OM NOM RUN](https://eduquestsjp.pages.dev/om-nom-run.html)
- [CHESS DUEL](https://quizzesarena.github.io/chess-duel.html)
- [SHANGHAI TOWN](https://quizzesarena.github.io/shanghai-town.html)
- [STICKMAN DOORS AND ISLAND](https://quizzesarena.github.io/stickman-doors-and-island.html)
- [BOUNCY BOMB BUDDIES](https://eduquestsjp.pages.dev/bouncy-bomb-buddies.html)
- [ROLLING BALLS SEA RACE](https://quizzesarena.github.io/rolling-balls-sea-race.html)
- [HAPPY JUMP](https://eduquestkr.pages.dev/happy-jump.html)
- [DISASSEMBLE THE PICTURE PUZZLE](https://quizzesarena.onrender.com/disassemble-the-picture-puzzle.html)
- [ITALIAN BRAINROT OBBY PARKOUR](https://quizzesarena.onrender.com/italian-brainrot-obby-parkour.html)
- [CITY BANANA MAN AGENT](https://eduquestkr.pages.dev/city-banana-man-agent.html)
- [ANIMAL IN RAILS](https://eduquestsjp.pages.dev/animal-in-rails.html)
- [CATEGORY ART32](https://brainquestspt.pages.dev/category-art32.html)
- [CHESS ONLINE](https://quizzesarena.web.app/chess-online.html)
- [HIDDEN OBJECT ROOMS EXPLORATION](https://quizzesarena.github.io/hidden-object-rooms-exploration.html)
- [CYBERPUNK AGENT](https://eduquestkr.pages.dev/cyberpunk-agent.html)
- [CATEGORY TOWER DEFENSE 2](https://eduquestsjp.pages.dev/category-tower-defense-2.html)
- [CANNON BLAST THE LAST STAND](https://quizzesarena.web.app/cannon-blast-the-last-stand.html)
- [REAL IMPOSSIBLE SKY TRACKS CAR DRIVING](https://quizzesarena.web.app/real-impossible-sky-tracks-car-driving.html)
- [DRAW ONE PART BRAIN PUZZLE](https://eduquestses.pages.dev/draw-one-part-brain-puzzle.html)
- [CITY BIKE RACING CHAMPION](https://quizzesarena.web.app/city-bike-racing-champion.html)
- [CATEGORY CUTE62](https://quizzesarena.onrender.com/category-cute62.html)
- [SHIP CONTROL 3D](https://eduquestses.pages.dev/ship-control-3d.html)
- [FURRY KUNG FU](https://quizzesarena.web.app/furry-kung-fu.html)
- [CATEGORY SANDBOX](https://eduquestkr.pages.dev/category-sandbox.html)
- [FISH OUT OF WATER](https://quizzesarena.web.app/fish-out-of-water.html)
- [TAP BLOCK PUZZLE SMASH GAME](https://eduquestkr.pages.dev/tap-block-puzzle-smash-game.html)
- [CASTLE CRAFT](https://quizzesarena.github.io/castle-craft.html)
- [FILL SORT PUZZLE](https://eduquestses.pages.dev/fill-sort-puzzle.html)
- [AUTUMN GLAM GALA](https://quizzesarena.web.app/autumn-glam-gala.html)
- [COLOSSATRON](https://eduquestses.pages.dev/colossatron.html)
- [TAP CAR](https://quizzesarena.web.app/tap-car.html)
- [ICE CUBE](https://eduquestsjp.pages.dev/ice-cube.html)
- [OBBY MODES ONLINE MINI GAMES](https://eduquestses.pages.dev/obby-modes-online-mini-games.html)
- [CUBE KING](https://quizzesarena.web.app/cube-king.html)
- [BALLISTIC BREAKTHROUGH](https://eduquestses.pages.dev/ballistic-breakthrough.html)
- [SNOW RACE 3D FUN RACING](https://eduquestkr.pages.dev/snow-race-3d-fun-racing.html)
- [TRICKY ARROW 2](https://eduquestses.pages.dev/tricky-arrow-2.html)
- [CATEGORY SHOOTER 2](https://eduquestsjp.pages.dev/category-shooter-2.html)
- [THE BRANCH RUNNER](https://eduquestses.pages.dev/the-branch-runner.html)
- [CATEGORY PLATFORM260](https://quizzesarena.github.io/category-platform260.html)
- [TATTOO MASTER](https://eduquestses.pages.dev/tattoo-master.html)
- [PERFECT ASMR CLEANING](https://eduquestses.pages.dev/perfect-asmr-cleaning.html)
- [RACING MASTER 3D](https://eduquestses.pages.dev/racing-master-3d.html)
- [CATEGORY STUNT128](https://quizzesarena.onrender.com/category-stunt128.html)
- [CATEGORY CUTE](https://quizzesarena.github.io/category-cute.html)
- [BLOCK MINE FUSE TNT](https://quizzesarena.web.app/block-mine-fuse-tnt.html)
- [THE LOST CITY MATCH 3](https://eduquestses.pages.dev/the-lost-city-match-3.html)
- [CATEGORY MYSTERY45](https://eduquestsjp.pages.dev/category-mystery45.html)
- [ESCAPE FROM THE PORTAL](https://quizzesarena.web.app/escape-from-the-portal.html)
- [HALLOWEEN STORE SORT](https://eduquestses.pages.dev/halloween-store-sort.html)
- [MERMAIDS SPOT THE DIFFERENCES](https://eduquestkr.pages.dev/mermaids-spot-the-differences.html)
- [JUMP BALL CLASSIC](https://eduquestkr.pages.dev/jump-ball-classic.html)
- [WORM ESCAPE](https://eduquestses.pages.dev/worm-escape.html)
- [CATEGORY SKILL256](https://eduquestkr.pages.dev/category-skill256.html)
- [ARROW CUBE ESCAPE](https://eduquestkr.pages.dev/arrow-cube-escape.html)
- [HEIST DEFENDER](https://eduquestsjp.pages.dev/heist-defender.html)
- [INDEX36](https://quizzesarena.onrender.com/index36.html)
- [SKILLFUL FINGER](https://eduquestses.pages.dev/skillful-finger.html)
- [POWERFUL PUNCH](https://eduquestsjp.pages.dev/powerful-punch.html)
- [CATEGORY TOWER DEFENSE 2](https://quizzesarena.onrender.com/category-tower-defense-2.html)
- [SUMMON TRIBE](https://eduquestses.pages.dev/summon-tribe.html)
- [GIRLFRIEND FROM HELL](https://eduquestses.pages.dev/girlfriend-from-hell.html)
- [ALPHABET LORE MAZE](https://quizzesarena.github.io/alphabet-lore-maze.html)
- [CATEGORY INCREMENTAL](https://quizzesarena.github.io/category-incremental.html)
