package cc3d

// Convert a level to C2M
import (
	"fmt"
	"sort"

	"github.com/magical/cc3d/c2m"
)

// Maps a CC3D tile ID to a psuedo-layer number,
// with lower layer numbers being at the bottom of the stack and higher layer numbers being at the top,
// such that no two tiles on the same layer should appear in the same position
// (with the exception of panel walls).
//
// This should also roughly correspond to the order tiles should be written to C2M files.
//
// This does NOT match the layers in the xml file.
var cc3d_layers = [...]int8{
	// Base tiles
	// These can't have a tile under them in c2m, so they have to be the bottommost tile.
	1:   1, //01 (1) = Floor Tile
	2:   1, //02 (2) = Wall
	3:   1, //03 (3) = Ice
	4:   1, //04 (4) = Ice Corner
	5:   1, //05 (5) = Ice Corner
	6:   1, //06 (6) = Ice Corner
	7:   1, //07 (7) = Ice Corner
	8:   1, //08 (8) = Water
	9:   1, //09 (9) = Fire
	10:  1, //0a (10) = Force floor
	11:  1, //0b (11) = Force floor
	12:  1, //0c (12) = Force floor
	13:  1, //0d (13) = Force floor
	14:  1, //0e (14) = Closed toggle door
	15:  1, //0f (15) = Open toggle door
	16:  1, //10 (16) = Red teleport
	17:  1, //11 (17) = Blue teleport
	20:  1, //14 (20) = Exit
	21:  1, //15 (21) = Slime
	30:  1, //1e (30) = Gravel
	31:  1, //1f (31) = Toggle door control
	32:  1, //20 (32) = Blue Golem control
	34:  1, //22 (34) = Red door
	35:  1, //23 (35) = Blue door
	36:  1, //24 (36) = Yellow door
	37:  1, //25 (37) = Green door
	44:  1, //2c (44) = F.I.S.H. Door
	45:  1, //2d (45) = Push up wall
	46:  1, //2e (46) = Appearing wall
	49:  1, //31 (49) = False blue wall
	50:  1, //32 (50) = Dirt
	57:  1, //39 (57) = Clone machine switch
	63:  1, //3f (63) = Security Gate Tools
	65:  1, //41 (65) = Trap
	66:  1, //42 (66) = Trap Control
	68:  1, //44 (68) = Clone machine
	70:  1, //46 (70) = Force floor random
	100: 1, //64 (100) = Yellow Golem control
	138: 1, //8a (138) = Security Gate Keys
	141: 1, //8d (141) = TURTLE
	198: 1, //c6 (198) = Sand
	199: 1, //c7 (199) = Red F.I.S.H. Door
	75:  1, //4b (75) = Laser Controller
	76:  1, //4c (76) = Laser Shooter

	154: 1, //9a (154) = Blue Push Control
	155: 1, //9b (155) = Green Push Control
	156: 1, //9c (156) = Red Push Control
	157: 1, //9d (157) = Yellow Push Control
	158: 1, //9e (158) = Toggle Blue Control
	159: 1, //9f (159) = Toggle Red Control
	160: 1, //a0 (160) = Toggle Yellow Control
	165: 1, //a5 (165) = Toggle Blue Door Closed
	166: 1, //a6 (166) = Toggle Red Door Closed
	167: 1, //a7 (167) = Toggle Yellow Door Closed
	168: 1, //a8 (168) = Toggle Blue Door Open
	169: 1, //a9 (169) = Toggle Red Door Open
	170: 1, //aa (170) = Toggle Yellow Door Open
	175: 1, //af (175) = Push Green Door Closed
	176: 1, //b0 (176) = Push Blue Door Closed
	177: 1, //b1 (177) = Push Red Door Closed
	178: 1, //b2 (178) = Push Yellow Door Closed

	// Items
	38:  2, //26 (38) = Red key
	39:  2, //27 (39) = Blue key
	40:  2, //28 (40) = Yellow key
	41:  2, //29 (41) = Green key
	42:  2, //2a (42) = F.I.S.H.
	43:  2, //2b (43) = EXTRA F.I.S.H.
	59:  2, //3b (59) = Ice orb
	60:  2, //3c (60) = Force Field orb
	61:  2, //3d (61) = Fire orb
	62:  2, //3e (62) = Water orb
	64:  2, //40 (64) = Red bomb
	144: 2, //90 (144) = Speed orb

	// Player / Blocks / Monsters
	22: 3, //16 (22) = Woop
	//
	23:  3, //17 (23) = Dirt block
	26:  3, //1a (26) = Ice block
	161: 3, //a1 (161) = Blue Block
	162: 3, //a2 (162) = Green Block
	163: 3, //a3 (163) = Red Block
	164: 3, //a4 (164) = Yellow Block
	184: 3, //b8 (184) = Reflector LU
	185: 3, //b9 (185) = Reflector DL
	186: 3, //ba (186) = Reflector UR
	187: 3, //bb (187) = Reflector RD
	72:  3, //48 (72) = Regular Security Bot
	73:  3, //49 (73) = Rotating Security Bot
	74:  3, //4a (74) = Multidirectional Security Bot
	190: 3, //be (190) = RotatingCC Security Bot
	191: 3, //bf (191) = Kickstarter BLock
	192: 3, //c0 (192) = Developer Support BLock
	//
	24:  3, //18 (24) = Walker
	25:  3, //19 (25) = Blinky
	33:  3, //21 (33) = Blue Golem
	51:  3, //33 (51) = Limpa
	52:  3, //34 (52) = Limpy
	53:  3, //35 (53) = Bouncer
	54:  3, //36 (54) = Omni
	55:  3, //37 (55) = Snappy
	56:  3, //38 (56) = Screamer
	87:  3, //57 (87) = Nibble
	99:  3, //63 (99) = Yellow Golem
	194: 3, //c2 (194) = Baby Blinky
	195: 3, //c3 (195) = Baby Screamer
	196: 3, //c4 (196) = Legs Green
	197: 3, //c5 (197) = Legs Red

	// Panel walls
	147: 4, //93 (147) = Panel Up
	148: 4, //94 (148) = Panel Right
	149: 4, //95 (149) = Panel Down
	150: 4, //96 (150) = Panel Left
}

func Convert(m *Map) (*c2m.Map, error) {
	// Rotate 90deg ccw as we convert.
	// We actually *have* to in order to get the clone connections to work right

	var out c2m.Map
	out.Title = m.Name
	out.Author = m.Author

	all := make([]Tile, 0, len(m.Player)+len(m.Tiles)+len(m.Walls)+len(m.Objects)+len(m.Enemies)+len(m.Blocks)+len(m.Switches))
	all = append(all, m.Player...)
	all = append(all, m.Tiles...)
	all = append(all, m.Objects...)
	all = append(all, m.Enemies...)
	all = append(all, m.Blocks...)
	all = append(all, m.Walls...)
	all = append(all, m.Switches...)
	sort.Slice(all, func(i, j int) bool {
		if all[i].X != all[j].X {
			return all[i].X < all[j].X
		}
		if all[i].Y != all[j].Y {
			return all[i].Y < all[j].Y
		}
		l0, l1 := all[i].layer(), all[j].layer()
		if l0 != l1 {
			return l0 < l1
		}
		return all[i].Type < all[j].Type // tiebreaker
	})

	w, h := m.Width, m.Height
	w, h = h, w // rotate

	tiles := make([][]c2m.Tile, w*h)
	panel := make([]uint8, w*h)
	// accumulate tiles for each coordinate
	for _, t := range all {
		x := t.X / 64
		y := t.Y / 64

		// Rotate -90deg
		x, y = y, h-x-1

		if x < 0 || x >= w || y < 0 || y >= h {
			return nil, fmt.Errorf("tile (%d,%d) out of bounds for level size %dx%d", t.X/64, t.Y/64, m.Width, m.Height)
		}

		i := y*w + x

		// if it's a panel wall, combine it into the panel masks
		if t.isPanel() {
			d := t.Direction
			d = (d + 3) % 4 // rotate
			panel[i] |= 1 << d
			continue
		}

		id := t.Type
		mod := uint32(0)
		// Special cased stuff
		switch id {
		case 2: // Wall
			id = 0x30 // Blue Wall
		case 0x41: // Trap
			id = 0x42 // Trap
		case 0x42: // Trap control
			id = 0x3A // Brown button
		case 0xc2, 0xc4:
			//  c2 (194) Baby Blinky -> 19 glider
			//  c4 (196) Legs Green -> 19 glider
			id = 0x19
		case 0xc3:
			//  c3 (195) Baby Screamer -> 38 fireball
			//  c5 (197) Legs Red -> 38 fireball
			id = 0x38
		case 0xc6:
			id = 0x1E //  c6 (198) Sand -> 1E gravel
		case 0xc7:
			id = 0x2C //  c7 (199) Red F.I.S.H. Door -> 2C socket
		case 0x9a, 0x9b, 0x9c, 0x9d:
			//  9a (154) Blue Push Control -> F2 sokoban button
			//  9b (155) Green Push Control -> F2 sokoban button
			//  9c (156) Red Push Control -> F2 sokoban button
			//  9d (157) Yellow Push Control -> F2 sokoban button
			colorMod := []uint32{1, 3, 0, 2} // red, blue, yellow, green
			id = 0xF2
			mod = colorMod[t.Type-0x9a]
		case 0xa1, 0xa2, 0xa3, 0xa4:
			//  a1 (161) Blue Block -> F1 sokoban block
			//  a2 (162) Green Block -> F1 sokoban block
			//  a3 (163) Red Block -> F1 sokoban block
			//  a4 (164) Yellow Block -> F1 sokoban block
			colorMod := []uint32{1, 3, 0, 2} // red, blue, yellow, green
			id = 0xF1
			mod = colorMod[t.Type-0xa1]
		case 0xaf, 0xb0, 0xb1, 0xb2:
			//  af (175) Push Green Door Closed -> F3 sokoban floor
			//  b0 (176) Push Blue Door Closed -> F3 sokoban floor
			//  b1 (177) Push Red Door Closed -> F3 sokoban floor
			//  b2 (178) Push Yellow Door Closed -> F3 sokoban floor
			colorMod := []uint32{3, 1, 0, 2} // red, blue, yellow, green
			id = 0xF3
			mod = colorMod[t.Type-0xaf]

		case 0x48, 0x49, 0x4a, 0x4b, 0x4c, 0xb8, 0xb9, 0xba, 0xbb, 0xbe:
			// Unsupported elements:
			//  48 (72) Regular Security Bot -> nothing
			//  49 (73) Rotating Security Bot -> nothing
			//  4a (74) Multidirectional Security Bot -> nothing
			//  4b (75) Laser Controller -> nothing
			//  4c (76) Laser Shooter -> nothing
			//  b8 (184) Reflector LU
			//  b9 (185) Reflector DL
			//  ba (186) Reflector UR
			//  bb (187) Reflector RD
			//  be (190) RotatingCC Security Bot -> nothing
			return nil, fmt.Errorf("tile %d (%s) not supported in C2M", t.Type, t.Attributes.Name)
		}
		// TODO:
		//  bf (191) Kickstarter BLock -> F1 sokoban block
		//  c0 (192) Developer Support BLock -> F1 sokoban block
		// Unsupported:
		//  9e (158) Toggle Blue Control
		//  9f (159) Toggle Red Control
		//  a0 (160) Toggle Yellow Control
		//  a5 (165) Toggle Blue Door Closed
		//  a6 (166) Toggle Red Door Closed
		//  a7 (167) Toggle Yellow Door Closed
		//  a8 (168) Toggle Blue Door Open
		//  a9 (169) Toggle Red Door Open
		//  aa (170) Toggle Yellow Door Open

		// TODO: rotate force floors, ice corners, reflectors

		dir := (t.Direction + 3) % 4
		v := c2m.Tile{
			ID:    uint8(id),
			Dir:   uint8(dir),
			Flags: mod,
		}
		tiles[i] = append(tiles[i], v)
	}

	// Add panel walls
	for i := range tiles {
		if panel[i] != 0 {
			tiles[i] = append(tiles[i], c2m.Tile{
				ID:    0x6D,
				Flags: uint32(panel[i]),
			})
		}
	}

	// copy tiles to c2m.Map
	out.Width = w
	out.Height = h
	out.Tiles = tiles

	return &out, nil
}

func (t Tile) isPanel() bool {
	return 0x93 <= t.Type && t.Type <= 0x96
}

func (t Tile) layer() int {
	if t.Type >= 0 && t.Type < len(cc3d_layers) {
		return int(cc3d_layers[t.Type])
	}
	return 0
}

// LL Sokoban blocks:
//
//    F1,D,+:color sokoban_block
//    F2:color sokoban_button
//    F3:color sokoban_wall

// Relevant CC2 tiles:
//
// 01:w    floor   CC1
// 02    wall    CC1
// 03    ice     CC1
// 04    ice wall ne     CC1
// 05    ice wall se     CC1
// 06    ice wall nw     CC1
// 07    ice wall sw     CC1
// 08    water   CC1
// 09    fire    CC1
// 0A    force floor n   CC1
// 0B    force floor e   CC1
// 0C    force floor s   CC1
// 0D    force floor w   CC1
// 0E    green toggle wall       CC1
// 0F    green toggle floor      CC1
// 10:w    red teleport
// 11:w    blue teleport   CC1
// 14    exit    CC1
// 15    toxic floor
// 16,D,+        chip    CC1
// 17,D,+        dirt block      CC1
// 18,D,+        walker  CC1
// 19,D,+        glider  CC1
// 1A,D,+        ice block       CC1
// 1E    gravel  CC1
// 1F    green button    CC1
// 20    blue button     CC1
// 21,D,+        tank    CC1
// 22    red door        CC1
// 23    blue door       CC1
// 24    yellow door     CC1
// 25    green door      CC1
// 26,+  red key CC1
// 27,+  blue key        CC1
// 28,+  yellow key      CC1
// 29,+  green key       CC1
// 2A,+  ic chip CC1
// 2B,+  extra chip      CC1
// 2C    chip socket     CC1
// 2D    popup wall      CC1
// 2E    invisible wall  CC1
// 2F    invisible wall (temp)   CC1
// 30    blue wall       CC1
// 31    blue floor      CC1
// 32    dirt    CC1
// 33,D,+        bug     CC1
// 34,D,+        centipede       CC1
// 35,D,+        ball    CC1
// 36,D,+        blob    CC1
// 37,D,+        red teeth       CC1
// 38,D,+        fireball        CC1
// 39    red button      CC1
// 3A    brown button    CC1
// 3B,+    ice boots     CC1
// 3C,+  magnet boots    CC1
// 3D,+  fire boots      CC1
// 3E,+  flippers        CC1
// 3F    boot thief      CC1
// 40,+  red bomb        CC1
// 41    open trap
// 42    trap    CC1
// 44:d    clone machine
// 45    hint    CC1
// 46    force floor random      CC1
// 57,D,+    blue teeth
// 63,D,+    yellow tank
// 64    yellow tank button
// 6B:s    custom floor
// 6D,P,+    thin wall
// 70:s    custom wall
// 76,m,+  modifier
// 77,mm,+  modifier
// 78,mmmm,+  modifier
// 8A    key thief
// 8D    turtle
// 90,+  speed boots
// 92,+  hook

// Special mappings:
//  02 Wall -> blue wall (real)
//  41 Trap -> 42 Trap
//  42 Trap control -> 3A brown button
//
//  93 (147) Panel Up -> 6D thin wall
//  94 (148) Panel Right -> 6D thin wall
//  95 (149) Panel Down -> 6D thin wall
//  96 (150) Panel Left -> 6D thin wall
//  9a (154) Blue Push Control -> F2 sokoban button
//  9b (155) Green Push Control -> F2 sokoban button
//  9c (156) Red Push Control -> F2 sokoban button
//  9d (157) Yellow Push Control -> F2 sokoban button
//  9e (158) Toggle Blue Control
//  9f (159) Toggle Red Control
//  a0 (160) Toggle Yellow Control
//  a1 (161) Blue Block -> F1 sokoban block
//  a2 (162) Green Block -> F1 sokoban block
//  a3 (163) Red Block -> F1 sokoban block
//  a4 (164) Yellow Block -> F1 sokoban block
//  a5 (165) Toggle Blue Door Closed
//  a6 (166) Toggle Red Door Closed
//  a7 (167) Toggle Yellow Door Closed
//  a8 (168) Toggle Blue Door Open
//  a9 (169) Toggle Red Door Open
//  aa (170) Toggle Yellow Door Open
//  af (175) Push Green Door Closed -> F3 sokoban floor
//  b0 (176) Push Blue Door Closed -> F3 sokoban floor
//  b1 (177) Push Red Door Closed -> F3 sokoban floor
//  b2 (178) Push Yellow Door Closed -> F3 sokoban floor
//  bf (191) Kickstarter BLock -> F1 sokoban block
//  c0 (192) Developer Support BLock -> F1 sokoban block
//  c2 (194) Baby Blinky -> 19 glider
//  c3 (195) Baby Screamer -> 38 fireball
//  c4 (196) Legs Green -> 19 glider
//  c5 (197) Legs Red -> 38 fireball
//  c6 (198) Sand -> 1E gravel
//  c7 (199) Red F.I.S.H. Door -> 2C socket

// Unsupported elements:
//  48 (72) Regular Security Bot -> nothing
//  49 (73) Rotating Security Bot -> nothing
//  4a (74) Multidirectional Security Bot -> nothing
//  4b (75) Laser Controller -> nothing
//  4c (76) Laser Shooter -> nothing
//  b8 (184) Reflector LU
//  b9 (185) Reflector DL
//  ba (186) Reflector UR
//  bb (187) Reflector RD
//  be (190) RotatingCC Security Bot -> nothing
