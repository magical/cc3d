package main

import (
	"flag"
	"log"
)

var outputFlag string

func main() {
	log.SetFlags(0)
	flag.StringVar(&outputFlag, "o", "", "output file for -map")
	listFlag := flag.Bool("info", false, "list info for one or more levels")
	mapFlag := flag.Bool("map", false, "convert a level into an image")
	flag.Parse()
	if *listFlag {
		if *mapFlag {
			log.Fatal("cannot use -list and -map together")
		}
		listMain()
	} else if *mapFlag {
		if *listFlag {
			log.Fatal("cannot use -map and -list together")
		}
		mapMain()
	}
}
