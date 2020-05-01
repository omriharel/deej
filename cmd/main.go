package main

import (
	"fmt"

	"github.com/omriharel/deej"
)

func main() {

	// first we need a logger
	logger, err := deej.NewLogger()
	if err != nil {
		panic(fmt.Sprintf("Failed to create logger: %v", err))
	}

	named := logger.Named("main")
	named.Debug("Created logger")

	// create the deej instance
	d, err := deej.NewDeej(logger)
	if err != nil {
		named.Fatalw("Failed to create deej object", "error", err)
	}

	// onwards, to glory
	if err = d.Initialize(); err != nil {
		named.Fatalw("Failed to initialize deej", "error", err)
	}
}
