package main

import (
	"log"

	"github.com/omriharel/deej"
)

func main() {
	d, err := deej.NewDeej()
	if err != nil {
		log.Fatalf("create deej object: %v", err)
	}

	if err = d.Initialize(); err != nil {
		log.Fatalf("initialize deej: %v", err)
	}
}
