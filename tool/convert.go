package main

import (
	"flag"
	"log"
	"os"

	"github.com/magical/cc3d"
	"github.com/magical/cc3d/c2m"
)

func convertMain() {
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
	err := doConvert(filename, outputFlag)
	if err != nil {
		log.Fatal(err)
	}
}

func doConvert(filename, outname string) (err error) {
	f := os.Stdin
	if filename != "-" {
		f, err = os.Open(filename)
		if err != nil {
			return err
		}
		defer f.Close()
	}
	origMap, err := cc3d.ReadLevel(f)
	if err != nil {
		return err
	}
	convertedMap, err := cc3d.Convert(origMap)
	if err != nil {
		return err
	}
	//pretty.Println(convertedMap)
	out, err := os.Create(outname)
	if err != nil {
		return err
	}
	defer out.Close()
	err = c2m.Encode(out, convertedMap)
	if err != nil {
		return err
	}
	return out.Close()
}
