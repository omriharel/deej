// ch340probe shows which driver the CH340 is using and, with -read, prints the
// slider lines coming from it over WinUSB.
//
// Listing only reads Windows' device database; it never talks to the board or
// its driver, so it's safe to run while the vendor driver is still installed.
// -read refuses to open the board unless it is bound to WinUSB.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/omriharel/deej/pkg/deej/ch340"
)

func main() {
	read := flag.Duration("read", 0, "open the CH340 via WinUSB and print lines for this long, e.g. -read 10s")
	baud := flag.Int("baud", 9600, "baud rate for -read")
	raw := flag.Bool("raw", false, "with -read, print raw bytes instead of lines")
	dtr := flag.Bool("dtr", false, "assert DTR/RTS on open (resets the Arduino)")
	linux := flag.Bool("linux", false, "use the Linux driver's init sequence")
	flag.Parse()
	ch340.AssertDTR = *dtr
	ch340.LinuxInit = *linux

	devs, err := ch340.Find()
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
	if len(devs) == 0 {
		fmt.Println("No CH340/CH341 plugged in.")
		os.Exit(2)
	}

	for _, d := range devs {
		fmt.Printf("Device:  %s\n", d.Description)
		fmt.Printf("ID:      %s\n", d.InstanceID)
		fmt.Printf("Driver:  %s\n", orNone(d.Service))
		if d.UsesWinUSB() {
			fmt.Println("Status:  ready for deej (com_port: winusb)")
		} else {
			fmt.Println("Status:  NOT on WinUSB - deej won't touch it (see docs/winusb.md)")
		}
		for _, p := range d.Paths {
			fmt.Printf("Path:    %s\n", p)
		}
		fmt.Println()
	}

	if *read <= 0 {
		return
	}

	dev, err := ch340.Open(*baud)
	if err != nil {
		fmt.Println("open failed:", err)
		os.Exit(3)
	}
	fmt.Printf("Opened %s, reading for %s...\n", dev, *read)

	go func() {
		time.Sleep(*read)
		dev.Close()
	}()

	if *raw {
		buf := make([]byte, 4096)
		total := 0
		for {
			n, err := dev.Read(buf)
			if err != nil {
				fmt.Printf("\nDone, %d bytes (%v).\n", total, err)
				return
			}
			total += n
			fmt.Printf("%d bytes: %q\n", n, buf[:n])
		}
	}

	r := bufio.NewReader(dev)
	count := 0
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if err == ch340.ErrClosed {
				fmt.Printf("Done, %d lines.\n", count)
				return
			}
			fmt.Println("read error:", err)
			os.Exit(4)
		}
		count++
		fmt.Print(line)
	}
}

func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}
