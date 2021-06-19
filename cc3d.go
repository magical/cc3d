package cc3d

import (
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// <map author="supernewton" name="Outside Port" height="12" width="11" background="2">

type Map struct {
	Author     string `xml:"author,attr"`
	Name       string `xml:"name,attr"`
	Height     int    `xml:"height,attr"`
	Width      int    `xml:"width,attr"`
	Background int    `xml:"background,attr"`
	Player     []Tile `xml:"player>tile"`
	Tiles      []Tile `xml:"tiles>tile"`
	Objects    []Tile `xml:"objects>tile"`
	Enemies    []Tile `xml:"enemies>tile"`
	Blocks     []Tile `xml:"blocks>tile"`
	Walls      []Tile `xml:"walls>tile"`
	Switches   []Tile `xml:"switches>tile"`

	ExtraElem []xml.Name `xml:",any"`
}

// <tile image_index="22" x="384" y="384" direction="3" type="22">

type Tile struct {
	ImageIndex int        `xml:"image_index,attr"`
	X          int        `xml:"x,attr"`
	Y          int        `xml:"y,attr"`
	Direction  int        `xml:"direction,attr"`
	Type       int        `xml:"type,attr"`
	Attributes Attributes `xml:"attributes"`

	Extra []xml.Attr `xml:",any,attr"`
}

// <attributes flags="67657728" editor_category="1" name="Woop" />

type Attributes struct {
	Flags          uint64 `xml:"flags,attr"`
	EditorCategory int    `xml:"editor_category,attr"`
	Name           string `xml:"name,attr"`

	// obsolete attrs?
	FirstFrame   int    `xml:"first_frame,attr"`
	CurrentFrame int    `xml:"current_frame,attr"`
	EditFrame    int    `xml:"edit_frame,attr"`
	TotalFrames  int    `xml:"total_frames,attr"`
	FramesPerDir int    `xml:"frames_per_dir,attr"`
	MapChar      string `xml:"map_char,attr"`

	Extra []xml.Attr `xml:",any,attr"`
}

func (t Tile) String() string {
	var extra string
	for _, attr := range t.Extra {
		extra += "," + attr.Name.Local + "=" + attr.Value
	}
	for _, attr := range t.Attributes.Extra {
		extra += "," + attr.Name.Local + "=" + attr.Value
	}
	return fmt.Sprintf("(%d,%d)%s%s", t.X/64, t.Y/64, t.Attributes.Name, extra)
}

func (a Attributes) Equal(x Attributes) bool {
	return a.Flags == x.Flags &&
		a.EditorCategory == x.EditorCategory &&
		a.Name == x.Name &&
		a.FirstFrame == x.FirstFrame &&
		a.EditFrame == x.EditFrame &&
		a.TotalFrames == x.TotalFrames &&
		a.FramesPerDir == x.FramesPerDir &&
		a.MapChar == x.MapChar
}

type NamedReader interface {
	io.Reader
	Name() string
}

func SearchLevel(m *Map, levelid string) {
	found := false
	var names []string
	countTiles := func(tiles []Tile) {
		for _, t := range tiles {
			if t.Type == 0xbf || t.Type == 0xc0 {
				names = append(names, t.Attributes.Name)
				found = true
			}
		}
	}
	countTiles(m.Player)
	countTiles(m.Tiles)
	countTiles(m.Objects)
	countTiles(m.Enemies)
	countTiles(m.Blocks)
	countTiles(m.Walls)
	countTiles(m.Switches)

	if found {
		fmt.Println(levelid, names)
	}
}

func ReadLevel(r NamedReader) (*Map, error) {
	var d *xml.Decoder
	if strings.HasSuffix(r.Name(), ".gz") {
		zr, err := gzip.NewReader(r)
		if err != nil {
			return nil, err
		}
		d = xml.NewDecoder(zr)
	} else {
		d = xml.NewDecoder(r)
	}

	d.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		// levels are declared as encoding="utf-16" but are in fact ascii
		// so install a passthrough CharsetReader function
		return input, nil
	}
	var m *Map
	if err := d.Decode(&m); err != nil {
		return nil, err
	}

	return m, nil
}

// Check a level for validity.
// Returns a list of problems found.
func Check(m *Map) []string {
	var warnings []string
	warn := func(msg string, args ...interface{}) {
		warnings = append(warnings, fmt.Sprintf(msg, args...))
	}
	//warn("test")
	if !(m.Background == 0 || m.Background == 2) {
		warn("invalid background: %d", m.Background)
	}
	checkTiles := func(tiles []Tile) {
		for _, t := range tiles {
			if !(0 <= t.X && t.X < m.Width*64) {
				warn("tile x pos is out of range: x=%d, width=%d", t.X, m.Width)
			}
			if !(t.X%64 == 0) {
				warn("tile x pos is not a multiple of 64: x=%d", t.X)
			}
			if !(0 <= t.Y && t.Y < m.Height*64) {
				warn("tile y pos is out of range: y=%d, height=%d", t.Y, m.Height)
			}
			if !(t.Y%64 == 0) {
				warn("tile y pos is not a multiple of 64: y=%d", t.Y)
			}
			if !(0 <= t.Direction && t.Direction <= 3) {
				warn("invalid direction %d", t.Direction)
			}
		}
	}
	checkTiles(m.Player)
	checkTiles(m.Tiles)
	checkTiles(m.Objects)
	checkTiles(m.Enemies)
	checkTiles(m.Blocks)
	checkTiles(m.Walls)
	checkTiles(m.Switches)

	for _, name := range m.ExtraElem {
		warn("ignored top-level element <%s>", name.Local)
	}

	return warnings
}
