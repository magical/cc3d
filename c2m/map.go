package c2m

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

type Map struct {
	Title  string
	Author string
	Width  int
	Height int
	Tiles  [][]Tile // list of tile stacks
}

type Tile struct {
	ID    uint8
	Dir   uint8
	Flags uint32
}

type Chunk struct {
	Name [4]byte
	Size uint32
	Data []byte
}

func writeChunk(w io.Writer, name string, data []byte) (int64, error) {
	written := int64(0)
	if len(name) != 4 {
		return 0, fmt.Errorf("c2m chunk name must be 4 bytes long: %q", name)
	}
	if int64(len(data)) > math.MaxUint32 {
		return 0, fmt.Errorf("c2m chunk %s is too long: %d > 4GB", name, len(data))
	}
	n, err := io.WriteString(w, name)
	written += int64(n)
	if err != nil {
		return written, err
	}
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(len(data)))
	n, err = w.Write(buf[:4])
	written += int64(n)
	if err != nil {
		return written, err
	}
	n, err = w.Write(data)
	written += int64(n)
	return written, err
}

// Write a chunk whose contents is a string followed by a NUL byte.
func writeChunkString(w io.Writer, name string, data string) (int64, error) {
	// TODO: optimize allocations
	return writeChunk(w, name, []byte(data+"\x00"))
}

func encodeMap(tiles [][]Tile, w, h int) ([]byte, error) {
	b := new(bytes.Buffer)
	if w < 10 {
		return nil, fmt.Errorf("map size %dx%d out of range: %[1]d < 10", w, h)
	}
	if w > 100 {
		return nil, fmt.Errorf("map size %dx%d out of range: %[1]d > 100", w, h)
	}
	if h < 7 {
		return nil, fmt.Errorf("map size %dx%d out of range: %[2]d < 7", w, h)
	}
	if h > 100 {
		return nil, fmt.Errorf("map size %dx%d out of range: %[2]d > 100", w, h)
	}
	b.WriteByte(uint8(w))
	b.WriteByte(uint8(h))
	for _, stack := range tiles {
		// iterate backwards; our stacks go from bottom to top
		for j := len(stack) - 1; j >= 0; j-- {
			t := stack[j]
			encodeTile(b, t)
			if j == 0 && t.hasLower() {
				// Insert a virtual Floor tile if the bottommost tile is allowed to have a tile underneath
				b.WriteByte(0x1)
			}
			if j != 0 && !t.hasLower() {
				// TODO: better error
				return nil, fmt.Errorf("tile should be at bottom of stack: %v", t)
			}
		}
	}
	return b.Bytes(), nil
}

func encodeTile(b *bytes.Buffer, t Tile) {
	if t.Flags != 0 && !t.hasExtra() {
		if t.Flags <= math.MaxUint8 {
			b.WriteByte(0x76)
			b.WriteByte(uint8(t.Flags))
		} else if t.Flags <= math.MaxUint16 {
			b.WriteByte(0x77)
			binary.Write(b, binary.LittleEndian, uint16(t.Flags))
		} else {
			b.WriteByte(0x78)
			binary.Write(b, binary.LittleEndian, uint32(t.Flags))
		}
	}
	b.WriteByte(t.ID)
	if t.hasDir() {
		b.WriteByte(t.Dir)
	}
	if t.hasExtra() {
		// TODO: check overflow
		b.WriteByte(uint8(t.Flags))
	}
}

func Encode(w io.Writer, m *Map) error {
	mapdata, err := encodeMap(m.Tiles, m.Width, m.Height)
	if err != nil {
		return err
	}
	if _, err := writeChunkString(w, "CC2M", "5\x00"); err != nil {
		return err
	}
	if _, err := writeChunkString(w, "TITL", m.Title); err != nil {
		return err
	}
	if _, err := writeChunkString(w, "AUTH", m.Author); err != nil {
		return err
	}
	if _, err := writeChunkString(w, "NOTE", "Written by github.com/magical/cc3d/c2m"); err != nil {
		return err
	}
	optn := make([]byte, 3) // Time: 0, Viewport: 10x10
	if _, err := writeChunk(w, "OPTN", optn); err != nil {
		return err
	}
	// TODO: PACK
	if _, err := writeChunk(w, "MAP ", mapdata); err != nil {
		return err
	}
	if _, err := writeChunk(w, "END ", nil); err != nil {
		return err
	}
	return nil
}

const (
	hasDir = 1 << iota
	hasExtra
	hasLower
)

type meta struct {
	ID    int
	Mod   int8
	Flags uint8
	Name  string
}

func (t Tile) hasLower() bool {
	if int(t.ID) < len(tilespec) {
		return tilespec[t.ID].Flags&hasLower != 0
	}
	return false
}
func (t Tile) hasDir() bool {
	if int(t.ID) < len(tilespec) {
		return tilespec[t.ID].Flags&hasDir != 0
	}
	return false
}
func (t Tile) hasExtra() bool {
	if int(t.ID) < len(tilespec) {
		return tilespec[t.ID].Flags&hasExtra != 0
	}
	return false
}

func (t Tile) HasDir() bool { return t.hasDir() } // FIXME: don't export

func (t Tile) String() string {
	if int(t.ID) < len(tilespec) && tilespec[t.ID].Name != "" {
		return fmt.Sprintf("%d %s", t.ID, tilespec[t.ID].Name)
	}
	return fmt.Sprintf("%d Unknown tile", t.ID)
}

var tilespec = [...]meta{
	1:   {ID: 0x1, Mod: 'w', Flags: 0, Name: "floor"},
	2:   {ID: 0x2, Mod: 0, Flags: 0, Name: "wall"},
	3:   {ID: 0x3, Mod: 0, Flags: 0, Name: "ice"},
	4:   {ID: 0x4, Mod: 0, Flags: 0, Name: "ice wall ne"},
	5:   {ID: 0x5, Mod: 0, Flags: 0, Name: "ice wall se"},
	6:   {ID: 0x6, Mod: 0, Flags: 0, Name: "ice wall nw"},
	7:   {ID: 0x7, Mod: 0, Flags: 0, Name: "ice wall sw"},
	8:   {ID: 0x8, Mod: 0, Flags: 0, Name: "water"},
	9:   {ID: 0x9, Mod: 0, Flags: 0, Name: "fire"},
	10:  {ID: 0xa, Mod: 0, Flags: 0, Name: "force floor n"},
	11:  {ID: 0xb, Mod: 0, Flags: 0, Name: "force floor e"},
	12:  {ID: 0xc, Mod: 0, Flags: 0, Name: "force floor s"},
	13:  {ID: 0xd, Mod: 0, Flags: 0, Name: "force floor w"},
	14:  {ID: 0xe, Mod: 0, Flags: 0, Name: "green toggle wall"},
	15:  {ID: 0xf, Mod: 0, Flags: 0, Name: "green toggle floor"},
	16:  {ID: 0x10, Mod: 'w', Flags: 0, Name: "red teleport"},
	17:  {ID: 0x11, Mod: 'w', Flags: 0, Name: "blue teleport"},
	18:  {ID: 0x12, Mod: 0, Flags: 0, Name: "yellow teleport"},
	19:  {ID: 0x13, Mod: 0, Flags: 0, Name: "green teleport"},
	20:  {ID: 0x14, Mod: 0, Flags: 0, Name: "exit"},
	21:  {ID: 0x15, Mod: 0, Flags: 0, Name: "toxic floor"},
	22:  {ID: 0x16, Mod: 0, Flags: hasDir | hasLower, Name: "chip"},
	23:  {ID: 0x17, Mod: 0, Flags: hasDir | hasLower, Name: "dirt block"},
	24:  {ID: 0x18, Mod: 0, Flags: hasDir | hasLower, Name: "walker"},
	25:  {ID: 0x19, Mod: 0, Flags: hasDir | hasLower, Name: "glider"},
	26:  {ID: 0x1a, Mod: 0, Flags: hasDir | hasLower, Name: "ice block"},
	27:  {ID: 0x1b, Mod: 0, Flags: hasLower, Name: "thin wall s"},
	28:  {ID: 0x1c, Mod: 0, Flags: hasLower, Name: "thin wall e"},
	29:  {ID: 0x1d, Mod: 0, Flags: hasLower, Name: "thin wall se"},
	30:  {ID: 0x1e, Mod: 0, Flags: 0, Name: "gravel"},
	31:  {ID: 0x1f, Mod: 0, Flags: 0, Name: "green button"},
	32:  {ID: 0x20, Mod: 0, Flags: 0, Name: "blue button"},
	33:  {ID: 0x21, Mod: 0, Flags: hasDir | hasLower, Name: "tank"},
	34:  {ID: 0x22, Mod: 0, Flags: 0, Name: "red door"},
	35:  {ID: 0x23, Mod: 0, Flags: 0, Name: "blue door"},
	36:  {ID: 0x24, Mod: 0, Flags: 0, Name: "yellow door"},
	37:  {ID: 0x25, Mod: 0, Flags: 0, Name: "green door"},
	38:  {ID: 0x26, Mod: 0, Flags: hasLower, Name: "red key"},
	39:  {ID: 0x27, Mod: 0, Flags: hasLower, Name: "blue key"},
	40:  {ID: 0x28, Mod: 0, Flags: hasLower, Name: "yellow key"},
	41:  {ID: 0x29, Mod: 0, Flags: hasLower, Name: "green key"},
	42:  {ID: 0x2a, Mod: 0, Flags: hasLower, Name: "ic chip"},
	43:  {ID: 0x2b, Mod: 0, Flags: hasLower, Name: "extra chip"},
	44:  {ID: 0x2c, Mod: 0, Flags: 0, Name: "chip socket"},
	45:  {ID: 0x2d, Mod: 0, Flags: 0, Name: "popup wall"},
	46:  {ID: 0x2e, Mod: 0, Flags: 0, Name: "invisible wall"},
	47:  {ID: 0x2f, Mod: 0, Flags: 0, Name: "invisible wall (temp)"},
	48:  {ID: 0x30, Mod: 0, Flags: 0, Name: "blue wall"},
	49:  {ID: 0x31, Mod: 0, Flags: 0, Name: "blue floor"},
	50:  {ID: 0x32, Mod: 0, Flags: 0, Name: "dirt"},
	51:  {ID: 0x33, Mod: 0, Flags: hasDir | hasLower, Name: "bug"},
	52:  {ID: 0x34, Mod: 0, Flags: hasDir | hasLower, Name: "centipede"},
	53:  {ID: 0x35, Mod: 0, Flags: hasDir | hasLower, Name: "ball"},
	54:  {ID: 0x36, Mod: 0, Flags: hasDir | hasLower, Name: "blob"},
	55:  {ID: 0x37, Mod: 0, Flags: hasDir | hasLower, Name: "red teeth"},
	56:  {ID: 0x38, Mod: 0, Flags: hasDir | hasLower, Name: "fireball"},
	57:  {ID: 0x39, Mod: 0, Flags: 0, Name: "red button"},
	58:  {ID: 0x3a, Mod: 0, Flags: 0, Name: "brown button"},
	59:  {ID: 0x3b, Mod: 0, Flags: hasLower, Name: "ice boots"},
	60:  {ID: 0x3c, Mod: 0, Flags: hasLower, Name: "magnet boots"},
	61:  {ID: 0x3d, Mod: 0, Flags: hasLower, Name: "fire boots"},
	62:  {ID: 0x3e, Mod: 0, Flags: hasLower, Name: "flippers"},
	63:  {ID: 0x3f, Mod: 0, Flags: 0, Name: "boot thief"},
	64:  {ID: 0x40, Mod: 0, Flags: hasLower, Name: "red bomb"},
	65:  {ID: 0x41, Mod: 0, Flags: 0, Name: "open trap"},
	66:  {ID: 0x42, Mod: 0, Flags: 0, Name: "trap"},
	67:  {ID: 0x43, Mod: 'd', Flags: 0, Name: "clone machine"},
	68:  {ID: 0x44, Mod: 'd', Flags: 0, Name: "clone machine"},
	69:  {ID: 0x45, Mod: 0, Flags: 0, Name: "hint"},
	70:  {ID: 0x46, Mod: 0, Flags: 0, Name: "force floor random"},
	71:  {ID: 0x47, Mod: 0, Flags: 0, Name: "gray button"},
	72:  {ID: 0x48, Mod: 0, Flags: 0, Name: "revolving door sw"},
	73:  {ID: 0x49, Mod: 0, Flags: 0, Name: "revolving door nw"},
	74:  {ID: 0x4a, Mod: 0, Flags: 0, Name: "revolving door ne"},
	75:  {ID: 0x4b, Mod: 0, Flags: 0, Name: "revolving door se"},
	76:  {ID: 0x4c, Mod: 0, Flags: hasLower, Name: "time bonus"},
	77:  {ID: 0x4d, Mod: 0, Flags: hasLower, Name: "time toggle"},
	78:  {ID: 0x4e, Mod: 'w', Flags: 0, Name: "transmogrifier"},
	79:  {ID: 0x4f, Mod: 't', Flags: 0, Name: "railroad"},
	80:  {ID: 0x50, Mod: 'w', Flags: 0, Name: "steel wall"},
	81:  {ID: 0x51, Mod: 0, Flags: hasLower, Name: "time bomb"},
	82:  {ID: 0x52, Mod: 0, Flags: hasLower, Name: "helmet"},
	86:  {ID: 0x56, Mod: 0, Flags: hasDir | hasLower, Name: "melinda"},
	87:  {ID: 0x57, Mod: 0, Flags: hasDir | hasLower, Name: "blue teeth"},
	89:  {ID: 0x59, Mod: 0, Flags: hasLower, Name: "hiking boots"},
	90:  {ID: 0x5a, Mod: 0, Flags: 0, Name: "male-only"},
	91:  {ID: 0x5b, Mod: 0, Flags: 0, Name: "female-only"},
	92:  {ID: 0x5c, Mod: 'g', Flags: 0, Name: "logic gate"},
	94:  {ID: 0x5e, Mod: 'w', Flags: 0, Name: "pink button"},
	95:  {ID: 0x5f, Mod: 0, Flags: 0, Name: "flame jet"},
	96:  {ID: 0x60, Mod: 0, Flags: 0, Name: "flame jet"},
	97:  {ID: 0x61, Mod: 0, Flags: 0, Name: "orange button"},
	98:  {ID: 0x62, Mod: 0, Flags: hasLower, Name: "lightning"},
	99:  {ID: 0x63, Mod: 0, Flags: hasDir | hasLower, Name: "yellow tank"},
	100: {ID: 0x64, Mod: 0, Flags: 0, Name: "yellow tank button"},
	101: {ID: 0x65, Mod: 0, Flags: hasDir | hasLower, Name: "chip mimic"},
	102: {ID: 0x66, Mod: 0, Flags: hasDir | hasLower, Name: "melinda mimic"},
	104: {ID: 0x68, Mod: 0, Flags: hasLower, Name: "bowling ball"},
	105: {ID: 0x69, Mod: 0, Flags: hasDir | hasLower, Name: "rover"},
	106: {ID: 0x6a, Mod: 0, Flags: hasLower, Name: "time down"},
	107: {ID: 0x6b, Mod: 's', Flags: 0, Name: "custom floor"},
	109: {ID: 0x6d, Mod: 0, Flags: hasExtra | hasLower, Name: "thin wall"},
	111: {ID: 0x6f, Mod: 0, Flags: hasLower, Name: "rr sign"},
	112: {ID: 0x70, Mod: 's', Flags: 0, Name: "custom wall"},
	113: {ID: 0x71, Mod: 'G', Flags: 0, Name: "symbol"},
	114: {ID: 0x72, Mod: 0, Flags: 0, Name: "purple toggle floor"},
	115: {ID: 0x73, Mod: 0, Flags: 0, Name: "purple toggle wall"},
	122: {ID: 0x7a, Mod: 0, Flags: hasLower, Name: "10 point flag"},
	123: {ID: 0x7b, Mod: 0, Flags: hasLower, Name: "100 point flag"},
	124: {ID: 0x7c, Mod: 0, Flags: hasLower, Name: "1000 point flag"},
	125: {ID: 0x7d, Mod: 0, Flags: 0, Name: "green wall"},
	126: {ID: 0x7e, Mod: 0, Flags: 0, Name: "green floor"},
	127: {ID: 0x7f, Mod: 0, Flags: hasLower, Name: "no sign"},
	128: {ID: 0x80, Mod: 0, Flags: hasLower, Name: "double points flag"},
	129: {ID: 0x81, Mod: 0, Flags: hasDir | hasExtra | hasLower, Name: "direction block"},
	130: {ID: 0x82, Mod: 0, Flags: hasDir | hasLower, Name: "floor monster"},
	131: {ID: 0x83, Mod: 0, Flags: hasLower, Name: "green bomb"},
	132: {ID: 0x84, Mod: 0, Flags: hasLower, Name: "green chip"},
	135: {ID: 0x87, Mod: 'w', Flags: 0, Name: "black button"},
	136: {ID: 0x88, Mod: 'w', Flags: 0, Name: "off switch"},
	137: {ID: 0x89, Mod: 'w', Flags: 0, Name: "on switch"},
	138: {ID: 0x8a, Mod: 0, Flags: 0, Name: "key thief"},
	139: {ID: 0x8b, Mod: 0, Flags: hasDir | hasLower, Name: "ghost"},
	140: {ID: 0x8c, Mod: 0, Flags: hasLower, Name: "foil"},
	141: {ID: 0x8d, Mod: 0, Flags: 0, Name: "turtle"},
	142: {ID: 0x8e, Mod: 0, Flags: hasLower, Name: "secret eye"},
	143: {ID: 0x8f, Mod: 0, Flags: hasLower, Name: "treasure"},
	144: {ID: 0x90, Mod: 0, Flags: hasLower, Name: "speed boots"},
	146: {ID: 0x92, Mod: 0, Flags: hasLower, Name: "hook"},
	// Lexy's Labyrinth extensions
	241: {ID: 0xf1, Mod: 'c', Flags: hasDir | hasLower, Name: "sokoban block"},
	242: {ID: 0xf2, Mod: 'c', Flags: 0, Name: "sokoban button"},
	243: {ID: 0xf3, Mod: 'c', Flags: 0, Name: "sokoban wall"},
}
