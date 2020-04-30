package main

import (
	"github.com/omriharel/deej"
)

func main() {
	d := deej.Deej{}

	d.Initialize()
	go d.Run()
}
