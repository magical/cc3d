package cc3d

import (
	"fmt"
	"log"
	"sort"
	"strconv"
)

func printlnf(msg string, args ...interface{}) {
	fmt.Println(fmt.Sprintf(msg, args...))
}

func PrintInfo(m *Map, levelid string) {
	fmt.Println(m.Name)
	fmt.Println(m.Author)
	fmt.Println(m.Width, "x", m.Height)
	fmt.Println("background", m.Background)
	fmt.Println("player", m.Player)
	fmt.Println("tiles   ", m.Tiles)
	fmt.Println("objects ", m.Objects)
	fmt.Println("enemies ", m.Enemies)
	fmt.Println("blocks  ", m.Blocks)
	fmt.Println("walls   ", m.Walls)
	fmt.Println("switches", m.Switches)

	for _, w := range Check(m) {
		log.Print(levelid, ": warning: ", w)
	}

	type point struct{ X, Y int }
	var ta = make(map[int][]Attributes)
	var grid = make(map[point][]Tile)
	countTiles := func(tiles []Tile) {
		for _, t := range tiles {
			ta[t.Type] = addAttrs(ta[t.Type], t.Attributes)
			//p := point{t.X / 64, t.Y / 64}
			//grid[p] = append(grid[p], t)
		}
	}
	countTiles(m.Player)
	countTiles(m.Tiles)
	countTiles(m.Objects)
	countTiles(m.Enemies)
	countTiles(m.Blocks)
	countTiles(m.Walls)
	countTiles(m.Switches)

	var keys []int
	for k := range ta {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		for _, v := range ta[k] {
			fmt.Println(fmt.Sprintf("%02[1]x (%[1]d)", k), "=", v.Name)
		}
	}
	for _, k := range keys {
		for _, v := range ta[k] {
			fmt.Printf("ATTRS %02x %s flags:%#x category:%d\n", k, v.Name, v.Flags, v.EditorCategory)
		}
	}

	if len(grid) > 0 {
		for y := 0; y < m.Height; y++ {
			for x := 0; x < m.Width; x++ {
				ts := grid[point{x, y}]
				if len(ts) == 0 {
					// TODO: move to check
					printlnf("GRID no tiles at %d,%d", x, y)
				}
				if len(ts) > 2 {
					printlnf("GRID %d,%d has %d tiles", x, y, len(ts))
				}
				for _, t := range ts {
					printlnf("TILE %2d,%d | %3d %-24s | dir=%d image_index=%d flags=%#x", x, y, t.Type, t.Attributes.Name, t.Direction, t.ImageIndex, t.Attributes.Flags)
				}
			}
		}
	}
}

func addAttrs(s []Attributes, attr Attributes) []Attributes {
	for _, v := range s {
		if attr.Equal(v) {
			return s
		}
	}
	return append(s, attr)
}

func formatFlagList(s []Attributes) string {
	var out []byte
	for _, attr := range s {
		if out != nil {
			out = append(out, ',')
		}
		out = append(out, "0x"...)
		out = strconv.AppendUint(out, attr.Flags, 16)
	}
	return string(out)
}
