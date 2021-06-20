package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/magical/cc3d"
)

func listMain() {
	if filename := flag.Arg(0); flag.NArg() == 0 || flag.NArg() == 1 && filename == "-" {
		err := process("-")
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	divider := false
	for _, filename := range flag.Args() {
		if divider {
			fmt.Println()
			fmt.Println("---")
			fmt.Println()
			divider = false
		}
		err := process(filename)
		if err != nil {
			log.Println(err)
			continue
		}
		divider = true
	}
}
func process(filename string) error {
	var f *os.File
	if filename == "-" {
		f = os.Stdin
	} else {
		f, err := os.Open(filename)
		if err != nil {
			return err
		}
		defer f.Close()
	}
	m, err := cc3d.ReadLevel(f)
	if err != nil {
		return err
	}
	levelid, _, _ := cut(filepath.Base(filename), ".")
	cc3d.PrintInfo(m, levelid)
	return nil
}

func cut(s, sep string) (before, after string, found bool) {
	i := strings.Index(s, sep)
	if i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}
