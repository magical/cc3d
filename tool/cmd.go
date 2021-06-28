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
	httpFlag := flag.Bool("http", false, "serve level maps over HTTP")
	convertFlag := flag.Bool("convert", false, "convert cc3d xml to c2m")
	flag.Parse()
	if *listFlag {
		if *httpFlag {
			log.Fatal("cannot use -http and -map together")
		}
		listMain()
	} else if *mapFlag {
		if *listFlag {
			log.Fatal("cannot use -map and -list together")
		}
		mapMain()
	} else if *httpFlag {
		if *listFlag || *mapFlag {
			log.Fatal("cannot use -http with -map or -list")
		}
		log.SetFlags(log.LstdFlags)
		httpMain()
	} else if *convertFlag {
		if *httpFlag || *listFlag || *mapFlag {
			log.Fatal("cannot use -convert with -http or -map or -list")
		}
		convertMain()
	}
}
