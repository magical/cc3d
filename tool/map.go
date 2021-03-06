package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/magical/cc3d"
	"github.com/nfnt/resize"
)

var flipFlag = flag.Bool("flip", false, "flip map coordinates")

func mapMain() {
	filename := flag.Arg(0)
	if flag.NArg() == 0 {
		filename = "-"
	}
	if flag.NArg() > 1 {
		log.Fatal("too many arguments")
	}
	if outputFlag == "" {
		log.Fatal("missing -o option")
	}
	err := doMap(filename, outputFlag)
	if err != nil {
		log.Fatal(err)
	}
}

func doMap(filename, outname string) (err error) {
	f := os.Stdin
	if filename != "-" {
		f, err = os.Open(filename)
		if err != nil {
			return err
		}
		defer f.Close()
	}
	m, err := cc3d.ReadLevel(f)
	if err != nil {
		return err
	}
	tileset := loadTiles(tileSize)
	im, err := makeMap(m, tileset, *flipFlag)
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

func loadTiles(size int) Tileset {
	exeDir := "." // TODO: relative to executable?
	dir := filepath.Join(exeDir, "ChucksChallengeImages")
	h0 := LoadTileDirectory(dir, size)
	h1, err := LoadTileImage("tworld.png", size)
	if err != nil {
		panic(err)
	}
	return FallbackTileset{h0, h1}
}

const tileSize = 48

func makeMap(m *cc3d.Map, tileset Tileset, flip bool) (*image.RGBA, error) {
	// A note about coordinate systems:
	// Levels are displayed in CC3D rotated 90 degrees ccw from how they are actually stored
	// (assuming a normal coordinate system with X going right and Y going down).
	// We can rotate the coordinate system to match the game but that actually messes
	// up directional tiles like force floors, which are consistent with the original
	// coordinate system, not the rotated one. So we don't do that by default.
	dx := m.Width * tileSize
	dy := m.Height * tileSize
	if flip {
		dx, dy = dy, dx
	}
	im := image.NewRGBA(image.Rect(0, 0, dx, dy))
	base := make(map[image.Point]bool)
	drawTiles := func(tiles []cc3d.Tile) {
		for _, t := range tiles {
			x := t.X / 64 * tileSize
			y := t.Y / 64 * tileSize
			if flip {
				x = t.Y / 64 * tileSize
				y = dy - (t.X/64+1)*tileSize
			}
			src := tileset.TileImage(t)
			warnMissingTileImage(t, src)
			var mask image.Image
			if src != nil {
				if isMostlyOpaque(src) && base[image.Pt(t.X, t.Y)] {
					mask = image.NewUniform(color.Alpha{0x80})
					//log.Println("masking tile %s", t.Attributes.Name)
				}
				draw.DrawMask(im, image.Rect(x, y, x+tileSize, y+tileSize), src, src.Bounds().Min, mask, image.ZP, draw.Over)
			}
			if dir := tileset.Direction(t); dir != nil {
				draw.DrawMask(im, image.Rect(x, y, x+tileSize, y+tileSize), dir, image.ZP, mask, image.ZP, draw.Over)
			}
			// Mark this coord as having a base tile drawn
			// unless it's a Floor, in which case we don't care about drawing over it
			if t.Type != 1 {
				base[image.Pt(t.X, t.Y)] = true
			}
		}
	}
	// 16287: Colored blocks are in the Tiles layer, Clone machines are in the Walls layer
	// 16287: Colored blocks in Tiles layer on top of toggle walls in the Walls layer
	// 1366: Panel walls in both the Blocks and Walls layers
	// TODO: experiment with drawing objects on top of blocks+enemies
	drawTiles(m.Switches)
	drawTiles(m.Tiles)
	drawTiles(m.Walls)
	drawTiles(m.Objects)
	drawTiles(m.Blocks)
	drawTiles(m.Enemies)
	drawTiles(m.Player)
	//im = resize.Resize(uint(dx/2), uint(dy/2), im, resize.NearestNeighbor).(*image.RGBA)
	return im, nil
}

func isMostlyOpaque(m image.Image) bool {
	if p, ok := m.(*image.RGBA); ok {
		alpha := 1.0
		n := 0
		i0, i1 := 3, p.Rect.Dx()*4
		for y := p.Rect.Min.Y; y < p.Rect.Max.Y; y++ {
			for i := i0; i < i1; i += 4 {
				if p.Pix[i] != 0xff {
					alpha += float64(p.Pix[i]) / 0xff
					n++
				}
			}
			i0 += p.Stride
			i1 += p.Stride
		}
		return alpha/float64(n) > 0.5
	} else if m, ok := m.(interface {
		Opaque() bool
	}); ok {
		return m.Opaque()
	}
	return true
}

type Tileset interface {
	// Returns an arrow indicating the direction for a tile
	// or nil if the tile is not a creature.
	Direction(t cc3d.Tile) image.Image

	// Returns the image for a tile.
	TileImage(t cc3d.Tile) image.Image
}

type ImageMap map[string]image.Image

func (h ImageMap) Direction(t cc3d.Tile) image.Image {
	switch t.Type {
	case 22, // Woop
		24,  // Walker
		25,  // Blinky
		33,  // Blue Golem
		51,  // Limpa
		52,  // Limpy
		53,  // Bouncer
		54,  // Omni
		55,  // Snappy
		56,  // Screamer
		68,  // Clone machine
		72,  // Regular Security Bot
		73,  // Rotating Security Bot
		74,  // Multidirectional Security Bot
		75,  // Laser Controller
		76,  // Laser Shooter
		87,  // Nibble
		99,  // Yellow Golem
		190, // RotatingCC Security Bot
		194, // Baby Blinky
		195, // Baby Screamer
		196, // Legs Green
		197: // Legs Red
		switch t.Direction {
		case 0:
			return h["ArrowN"]
		case 1:
			return h["ArrowE"]
		case 2:
			return h["ArrowS"]
		case 3:
			return h["ArrowW"]
		}
	}
	return nil
}

var (
	warned = make(map[int]bool)
	warnMu sync.RWMutex
)

func warnMissingTileImage(t cc3d.Tile, im image.Image) image.Image {
	if im == nil {
		warnMu.RLock()
		done := warned[t.Type]
		warnMu.RUnlock()
		if !done {
			warnMu.Lock()
			if !warned[t.Type] {
				log.Printf("warning: missing tile image for %d %s", t.Type, t.Attributes.Name)
				warned[t.Type] = true
			}
			warnMu.Unlock()
		}
	}
	return im
}

func (h ImageMap) TileImage(t cc3d.Tile) image.Image {
	return h.tileImage(t)
}

func (h ImageMap) tileImage(t cc3d.Tile) image.Image {
	switch t.Type {
	case 1:
		//01 (1) = Floor Tile
		return h["Floor2"]
	case 2:
		// 02 (2) = Wall
		return h["Wall"]
	case 3:
		// 03 (3) = Ice
		return h["Ice"]
	case 4, 5, 6, 7:
		// 04 (4) = Ice Corner
		// 05 (5) = Ice Corner
		// 06 (6) = Ice Corner
		// 07 (7) = Ice Corner
		break // TODO
	case 8:
		// 08 (8) = Water
		return h["Water2"]
	case 9:
		// 09 (9) = Fire
		return h["Lava"]
	case 10:
		// 0a (10) = Force floor
		return h["ConveyorNorth"]
	case 11:
		// 0b (11) = Force floor
		return h["ConveyorEast"]
	case 12:
		// 0c (12) = Force floor
		return h["ConveyorSouth"]
	case 13:
		// 0d (13) = Force floor
		return h["ConveyorWest"]
	case 14:
		// 0e (14) = Closed toggle door
		return h["PushGateGreen"]
	case 15:
		// 0f (15) = Open toggle door
		return h["PushGateGreenOpen"]
	case 16, 17:
		// 10 (16) = Red teleport
		// 11 (17) = Blue teleport
		return h["Teleports"] // TODO
	case 20:
		// 14 (20) = Exit
		return h["Exit"]
	case 21:
		// 15 (21) = Slime
		return h["Slime"]
	case 22:
		// 16 (22) = Woop
		return h["WoopCentered"]
	case 23:
		// 17 (23) = Dirt block
		return h["Mound"]
	case 24:
		// 18 (24) = Walker
		return h["LegsBlue"]
	case 25:
		// 19 (25) = Blinky
		return h["BlinkyCentered"]
	case 26:
		// 1a (26) = Ice block
		return h["IceGem"]
	case 30:
		// 1e (30) = Gravel
		return h["Gravel"]
	case 31:
		// 1f (31) = Toggle door control
		return h["PushButtonGreen"]
	case 32:
		// 20 (32) = Blue Golem control
		return h["GolemBlueSwitch"]
	case 33:
		// 21 (33) = Blue Golem
		return h["GolemBlueCentered"]
	case 34:
		// 22 (34) = Red door
		return h["RedDoor"]
	case 35:
		// 23 (35) = Blue door
		return h["Doors"]
	case 36:
		// 24 (36) = Yellow door
		return h["YellowDoor"]
	case 37:
		// 25 (37) = Green door
		return h["GreenDoor"]
	case 38:
		// 26 (38) = Red key
		return h["RedKey"]
	case 39:
		// 27 (39) = Blue key
		return h["BlueKey"]
	case 40:
		// 28 (40) = Yellow key
		return h["YellowKey"]
	case 41:
		// 29 (41) = Green key
		return h["GreenKey"]
	case 42, 43:
		// 2a (42) = F.I.S.H.
		// 2b (43) = EXTRA F.I.S.H.
		return h["FISHCentered"]
	case 44:
		// 2c (44) = F.I.S.H. Door
		return h["FISHDoorBlue"]
	case 45:
		// 2d (45) = Push up wall
		break // TODO
	case 46:
		// 2e (46) = Appearing wall
		return h["InvisibleWalls"]
	case 49:
		// 31 (49) = False blue wall
		return h["FakeWalls"]
	case 50:
		// 32 (50) = Dirt
		return h["Mud"]
	case 51:
		// 33 (51) = Limpa
		return h["LimpaL"]
	case 52:
		// 34 (52) = Limpy
		return h["LimpyR"]
	case 53:
		// 35 (53) = Bouncer
		return h["BouncerCentered"]
	case 54:
		// 36 (54) = Omni
		return h["Omni"]
	case 55:
		// 37 (55) = Snappy
		return h["SnappyCentered"]
	case 56:
		// 38 (56) = Screamer
		return h["ScreamerCentered"]
	case 57:
		// 39 (57) = Clone machine switch
		return h["CloneButton"]
	case 59, 60, 61, 62:
		// 3b (59) = Ice orb
		// 3c (60) = Force Field orb
		// 3d (61) = Fire orb
		// 3e (62) = Water orb
		break
	case 63:
		// 3f (63) = Security Gate Tools
		return h["SecurityGateBlue"]
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
		return h["Gear"]
	case 72, 73, 74:
		// 48 (72) = Regular Security Bot
		// 49 (73) = Rotating Security Bot
		// 4a (74) = Multidirectional Security Bot
		return h["SquishyCentered"]
	case 75:
		// 4b (75) = Laser Controller
		return h["SpitterButton"]
	case 76:
		// 4c (76) = Laser Shooter
		return h["Spitter"]
	case 87:
		// 57 (87) = Nibble
		return h["NibblesCentered"]
	case 99:
		// 63 (99) = Yellow Golem
		return h["GolemYellowCentered"]
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
		// Tile ID doesn't actually matter, only the direction matters
		// Compare levels 18620 and 1366 for example
		dir := t.Direction % 4
		panels := []string{"PanelE", "ThinWalls", "PanelW", "PanelN"}
		return h[panels[dir]]
	case 154:
		// 9a (154) = Blue Push Control
		return h["PressurePadBlue"]
	case 155:
		// 9b (155) = Green Push Control
		return h["PressurePadGreen"]
	case 156:
		// 9c (156) = Red Push Control
		return h["PressurePad"]
	case 157:
		// 9d (157) = Yellow Push Control
		return h["PressurePadYellow"]
	case 158:
		// 9e (158) = Toggle Blue Control
		return h["PushButtonBlue"]
	case 159:
		// 9f (159) = Toggle Red Control
		return h["PushButtonRed"]
	case 160:
		// a0 (160) = Toggle Yellow Control
		return h["PushButton"]
	case 161:
		// a1 (161) = Blue Block
		return h["BlueBlock"]
	case 162:
		// a2 (162) = Green Block
		return h["GreenBlock"]
	case 163:
		// a3 (163) = Red Block
		return h["RedBlock"]
	case 164:
		// a4 (164) = Yellow Block
		return h["ColouredBlock"]
	case 165:
		// a5 (165) = Toggle Blue Door Closed
		return h["PushGateBlue"]
	case 166:
		// a6 (166) = Toggle Red Door Closed
		return h["PushGateRed"]
	case 167:
		// a7 (167) = Toggle Yellow Door Closed
		return h["PushGate"]
	case 168:
		// a8 (168) = Toggle Blue Door Open
		return h["PushGateBlueOpen"]
	case 169:
		// a9 (169) = Toggle Red Door Open
		return h["PushGateRedOpen"]
	case 170:
		// aa (170) = Toggle Yellow Door Open
		return h["PushGateYellowOpen"]
	case 175:
		// af (175) = Push Green Door Closed
		return h["PressureGateGreen"]
	case 176:
		// b0 (176) = Push Blue Door Closed
		return h["PressureGateBlue"]
	case 177:
		// b1 (177) = Push Red Door Closed
		return h["PressureGate"]
	case 178:
		// b2 (178) = Push Yellow Door Closed
		return h["PressureGateYellow"]
	case 184:
		// b8 (184) = Reflector LU
		return h["ReflectorLU"]
	case 185:
		// b9 (185) = Reflector DL
		return h["ReflectorDL"]
	case 186:
		// ba (186) = Reflector UR
		return h["ReflectorUR"]
	case 187:
		// bb (187) = Reflector RD
		return h["ReflectorRD"]
	case 190:
		// be (190) = RotatingCC Security Bot
		return h["SquishyCentered"]
	case 191, 192:
		// bf (191) = Kickstarter BLock
		// c0 (192) = Developer Support BLock
		return h["RedBlock"] // TODO
	case 193:
		// Ben 10: Slime
		// TODO: why does this have a different id?
		return h["Slime"]
	case 194:
		// c2 (194) = Baby Blinky
		return h["BlinkyCentered"] // TODO
	case 195:
		// c3 (195) = Baby Screamer
		return h["ScreamerCentered"] // TODO
	case 196:
		// c4 (196) = Legs Green
		return h["LegsGreen"]
	case 197:
		// c5 (197) = Legs Red
		return h["LegsRed"]
	case 198:
		// c6 (198) = Sand
		break // TODO
	case 199:
		// c7 (199) = Red F.I.S.H. Door
		return h["FISHDoorRed"]
	}
	return nil
}

// Load a tileset from a directory containing PNG files for individual tiles.
func LoadTileDirectory(directory string, size int) ImageMap {
	tileMap := make(ImageMap)
	files, _ := filepath.Glob(filepath.Join(directory, "*.png"))
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
		//if !isMostlyOpaque(im) {
		//	log.Printf("tile %q is not opaque", name)
		//}
		tileMap[name] = im
	}
	return tileMap
}

var twTileInfo = []struct {
	Type int
	X    int
	Y    int
}{
	{1, 0, 0},     // Floor
	{2, 0, 1},     // Wall
	{42, 0, 2},    // IC Chip
	{8, 0, 3},     // Water
	{9, 0, 4},     // Fire
	{23, 0, 10},   // Dirt Block
	{35, 1, 6},    // Blue Door
	{34, 1, 7},    // Red Door
	{37, 1, 8},    // Green Door
	{36, 1, 9},    // Yellow Door
	{45, 2, 14},   // Popup wall
	{39, 6, 4},    // Blue Key
	{38, 6, 5},    // Red Key
	{41, 6, 6},    // Green Key
	{40, 6, 7},    // Yellow Key
	{0x3e, 6, 8},  // Flipper
	{0x3d, 6, 9},  // Fire boots
	{0x3b, 6, 10}, // Skates
	{0x3c, 6, 11}, // Suction boots
	{4, 1, 13},    // Ice corner SW
	{5, 1, 10},    // Ice corner NW
	{6, 1, 11},    // Ice corner NE
	{7, 1, 12},    // Ice corner SE
}

type SpriteMap struct {
	sheet subimager
	size  int
}

type subimager interface {
	image.Image
	SubImage(r image.Rectangle) image.Image
}

var _ subimager = &image.RGBA{}

func (_ SpriteMap) Direction(t cc3d.Tile) image.Image { return nil }

func (h SpriteMap) TileImage(t cc3d.Tile) image.Image {
	for _, info := range twTileInfo {
		if info.Type == t.Type {
			x, y := info.X*h.size, info.Y*h.size
			r := image.Rect(x, y, x+h.size, y+h.size)
			return h.sheet.SubImage(r)
		}
	}
	return nil
}

// Load a tileset from an image in Tile World's small format.
func LoadTileImage(path string, size int) (*SpriteMap, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error loading tileset: %w", err)
	}
	im, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("error loading tileset %q: %w", path, err)
	}
	// Transparentify
	if im, ok := im.(*image.RGBA); ok && im.Opaque() {
		//log.Print("transparentizing")
		dim := im.Rect.Size()
		magenta := []byte{0xff, 0, 0xff, 0xff}
		for y := 0; y < dim.Y; y++ {
			i := im.Stride * y
			for x := 0; x < dim.X; x++ {
				if bytes.Equal(im.Pix[i:i+4], magenta) {
					im.Pix[i+0] = 0
					im.Pix[i+2] = 0
					im.Pix[i+3] = 0 // transparent
				}
				i += 4
			}
		}
	}
	sim := im.(subimager)
	// TODO: check size
	return &SpriteMap{sim, size}, nil
}

type FallbackTileset []Tileset

func (h FallbackTileset) Direction(t cc3d.Tile) image.Image {
	return h[0].Direction(t)
}

func (h FallbackTileset) TileImage(t cc3d.Tile) image.Image {
	for i := range h {
		if im := h[i].TileImage(t); im != nil {
			return im
		}
	}
	return nil
}
