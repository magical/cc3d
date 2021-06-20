package main

import (
	"flag"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/magical/cc3d"
	"github.com/nfnt/resize"
)

func mapMain() {
	filename := flag.Arg(0)
	if flag.NArg() == 0 {
		filename = "-"
	}
	if flag.NArg() > 1 {
		log.Fatal("too many arguments")
	}
	err := doMap(filename, outputFlag)
	if err != nil {
		log.Fatal(err)
	}
}

func doMap(filename, outname string) (err error) {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	m, err := cc3d.ReadLevel(f)
	if err != nil {
		return err
	}
	im, err := makeMap(m)
	if err != nil {
		return err
	}
	out, err := os.Create(outname)
	if err != nil {
		return err
	}
	defer out.Close()
	err = png.Encode(out, im)
	if err != nil {
		return err
	}
	return out.Close()
}

func makeMap(m *cc3d.Map) (*image.RGBA, error) {
	const tileSize = 48
	h := loadTiles(tileSize)
	// A note about coordinate systems:
	// In CC3D levels, the Y axis goes from the left side of the screen to the right side
	// and the X axis goes from the bottom to the top of the screen (or really, towards/away from the viewer)
	// This is standard(-ish) for 3D coordinates (Z points down?) but backwards from the usual convention for 2D graphics.
	dy := m.Width * tileSize
	dx := m.Height * tileSize
	im := image.NewRGBA(image.Rect(0, 0, dx, dy))
	drawTiles := func(tiles []cc3d.Tile) {
		for _, t := range tiles {
			y := dy - t.X/64*tileSize - tileSize
			x := t.Y / 64 * tileSize
			src := h.WarnTileImage(t)
			if src != nil {
				draw.Over.Draw(im, image.Rect(x, y, x+tileSize, y+tileSize), src, image.ZP)
			}
		}
	}
	// Colored blocks are in the Tiles layer, Clone machines are in the Walls layer
	drawTiles(m.Walls)
	drawTiles(m.Tiles)
	drawTiles(m.Switches)
	drawTiles(m.Objects)
	drawTiles(m.Enemies)
	drawTiles(m.Blocks)
	drawTiles(m.Player)
	return im, nil
}

type ImageMap map[string]image.Image

var warned = make(map[int]bool)

func (h ImageMap) WarnTileImage(t cc3d.Tile) image.Image {
	m := h.TileImage(t)
	if m == nil && !warned[t.Type] {
		log.Printf("warning: missing tile image for %d %s", t.Type, t.Attributes.Name)
		warned[t.Type] = true
	}
	return m
}
func (h ImageMap) TileImage(t cc3d.Tile) image.Image {
	switch t.Type {
	case 1:
		//01 (1) = Floor Tile
		return h["Floor"]
	case 2:
		// 02 (2) = Wall
		return h["Wall"]
	case 3, 4, 5, 6, 7:
		// 03 (3) = Ice
		// 04 (4) = Ice Corner
		// 05 (5) = Ice Corner
		// 06 (6) = Ice Corner
		// 07 (7) = Ice Corner
		return h["Ice"]
	case 8:
		// 08 (8) = Water
		return h["Water2"]
	case 9:
		// 09 (9) = Fire
		return h["Lava"]
	case 10, 11, 12, 13:
		// 0a (10) = Force floor
		// 0b (11) = Force floor
		// 0c (12) = Force floor
		// 0d (13) = Force floor
		return h["ConveyorBelt"]
	case 14, 15:
		// 0e (14) = Closed toggle door
		// 0f (15) = Open toggle door
		return h["PushGate"]
	case 16, 17:
		// 10 (16) = Red teleport
		// 11 (17) = Blue teleport
		return h["Teleports"]
	case 20:
		// 14 (20) = Exit
		return h["Exit"]
	case 21:
		// 15 (21) = Slime
		return h["Slime"]
	case 22:
		// 16 (22) = Woop
		return h["Woop"]
	case 23:
		// 17 (23) = Dirt block
		return h["Mound"]
	case 24:
		// 18 (24) = Walker
		return h["Legs"] // Blue
	case 25:
		// 19 (25) = Blinky
		return h["Blinky"]
	case 26:
		// 1a (26) = Ice block
		return h["IceGem"]
	case 30:
		// 1e (30) = Gravel
		return h["Gravel"]
	case 31:
		// 1f (31) = Toggle door control
		return h["PushButton"]
	case 32:
		// 20 (32) = Blue Golem control
		return h["GolemBlue"]
	case 33:
		// 21 (33) = Blue Golem
		return h["GolemBlueSwitch"]
	case 34, 35, 36, 37:
		// 22 (34) = Red door
		// 23 (35) = Blue door
		// 24 (36) = Yellow door
		// 25 (37) = Green door
		return h["Doors"]
	case 38, 39, 40, 41:
		// 26 (38) = Red key
		// 27 (39) = Blue key
		// 28 (40) = Yellow key
		// 29 (41) = Green key
		return h["Key"]
	case 42, 43:
		// 2a (42) = F.I.S.H.
		// 2b (43) = EXTRA F.I.S.H.
		return h["FISH"]
	case 44:
		// 2c (44) = F.I.S.H. Door
		return h["FISHDoor"]
	case 45: // ???
	// 2d (45) = Push up wall
	case 46:
		// 2e (46) = Appearing wall
		return h["InvisibleWalls"]
	case 49:
		// 31 (49) = False blue wall
		return h["FakeWalls"]
	case 50:
		// 32 (50) = Dirt
		return h["Mud"]
	case 51, 52:
		// 33 (51) = Limpa
		// 34 (52) = Limpy
		return h["LimpaLimpy"]
	case 53:
		// 35 (53) = Bouncer
		return h["Bouncer"]
	case 54:
		// 36 (54) = Omni
		return h["Omni"]
	case 55:
		// 37 (55) = Snappy
		return h["Snappy"]
	case 56:
		// 38 (56) = Screamer
		return h["Screamer"]
	case 57:
		// 39 (57) = Clone machine switch
		return h["CloneButton"]
	case 59, 60, 61, 62:
		// 3b (59) = Ice orb
		// 3c (60) = Force Field orb
		// 3d (61) = Fire orb
		// 3e (62) = Water orb
		return h["Orbs"]
	case 63:
		// 3f (63) = Security Gate Tools
		return h["SecurityGate"]
	case 64:
		// 40 (64) = Red bomb
		return h["Bomb"]
	case 65:
		// 41 (65) = Trap
		return h["Cage"]
	case 66:
		// 42 (66) = Trap Control
		return h["CageButton"]
	case 68:
		// 44 (68) = Clone machine
		return h["CloneMachine"]
	case 70:
		// 46 (70) = Force floor random
		return h["ConveyerBelt"]
	case 72, 73, 74:
		// 48 (72) = Regular Security Bot
		// 49 (73) = Rotating Security Bot
		// 4a (74) = Multidirectional Security Bot
		return h["Squishy"]
	case 75:
		// 4b (75) = Laser Controller
		return h["SpitterButton"]
	case 76:
		// 4c (76) = Laser Shooter
		return h["Spitter"]
	case 87:
		// 57 (87) = Nibble
		return h["Nibbles"]
	case 99:
		// 63 (99) = Yellow Golem
		return h["GolemYellow"]
	case 100:
		// 64 (100) = Yellow Golem control
		return h["GolemYellowSwitch"]
	case 138:
		// 8a (138) = Security Gate Keys
		return h["SecurityGate"]
	case 141:
		// 8d (141) = TURTLE
		return h["Bridge"]
	case 144:
		// 90 (144) = Speed orb
		return h["Orbs"]
	case 147, 148, 149, 150:
		// 93 (147) = Panel Up
		// 94 (148) = Panel Right
		// 95 (149) = Panel Down
		// 96 (150) = Panel Left
		return h["ThinWalls"]
	case 154, 155, 156, 157:
		// 9a (154) = Blue Push Control
		// 9b (155) = Green Push Control
		// 9c (156) = Red Push Control
		// 9d (157) = Yellow Push Control
		return h["PressurePad"]
	case 158, 159, 160:
		// 9e (158) = Toggle Blue Control
		// 9f (159) = Toggle Red Control
		// a0 (160) = Toggle Yellow Control
		return h["PushButton"]
	case 161, 162, 163, 164:
		// a1 (161) = Blue Block
		// a2 (162) = Green Block
		// a3 (163) = Red Block
		// a4 (164) = Yellow Block
		return h["ColouredBlock"]
	case 165, 166, 167, 168, 169, 170:
		// a5 (165) = Toggle Blue Door Closed/Open
		// a6 (166) = Toggle Red Door Closed/Open
		// a7 (167) = Toggle Yellow Door Closed/Open
		// a8 (168) = Toggle Blue Door Open/Closed
		// a9 (169) = Toggle Red Door Open/Closed
		// aa (170) = Toggle Yellow Door Open/Closed
		return h["PushGate"]
	case 175, 176, 177, 178:
		// af (175) = Push Green Door Closed
		// b0 (176) = Push Blue Door Closed
		// b1 (177) = Push Red Door Closed
		// b2 (178) = Push Yellow Door Closed
		return h["PressureGate"]
	case 184, 185, 186, 187:
		// b8 (184) = Reflector LU
		// b9 (185) = Reflector DL
		// ba (186) = Reflector UR
		// bb (187) = Reflector RD
		return h["Reflector"]
	case 190:
		// be (190) = RotatingCC Security Bot
		return h["Squishy"]
	case 194:
		// bf (191) = Kickstarter BLock
		// c0 (192) = Developer Support BLock
		// c2 (194) = Baby Blinky
		return h["Blinky"]
	case 195:
		// c3 (195) = Baby Screamer
		return h["Screamer"]
	case 196, 197:
		// c4 (196) = Legs Green
		// c5 (197) = Legs Red
		return h["Legs"]
		// c6 (198) = Sand
		// c7 (199) = Red F.I.S.H. Door
	}
	return nil
}

func loadTiles(size int) ImageMap {
	tileMap := make(ImageMap)
	exeDir := "." // TODO: relative to executable?
	dir := filepath.Join(exeDir, "ChucksChallengeImages")
	files, _ := filepath.Glob(filepath.Join(dir, "*.png"))
	for _, imgPath := range files {
		filename := filepath.Base(imgPath)
		name, _, _ := cut(filename, ".")
		name = strings.TrimSuffix(name, "CreatorThumbnail")
		f, err := os.Open(imgPath)
		if err != nil {
			log.Printf("error loading tile: %v", err)
			continue
		}
		im, err := png.Decode(f)
		if err != nil {
			log.Printf("error loading tile %q: %v", name, err)
		}
		im = resize.Resize(uint(size), uint(size), im, resize.Bilinear)
		tileMap[name] = im
	}
	return tileMap
}
