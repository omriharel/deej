package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/omriharel/deej/pkg/deej"
)

var (
	gitCommit  string
	versionTag string
	buildType  string

	verbose bool
)

func init() {
	flag.BoolVar(&verbose, "verbose", false, "show verbose logs (useful for debugging serial)")
	flag.BoolVar(&verbose, "v", false, "shorthand for --verbose")
	flag.Parse()
}

func main() {
	for {
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("Recovered from panic: %v\n", r)
				}
			}()

			// first we need a logger
			logger, err := deej.NewLogger(buildType)
			if err != nil {
				panic(fmt.Sprintf("Failed to create logger: %v", err))
			}

			named := logger.Named("main")
			named.Debug("Created logger")

			named.Infow("Version info",
				"gitCommit", gitCommit,
				"versionTag", versionTag,
				"buildType", buildType)

			// provide a fair warning if the user's running in verbose mode
			if verbose {
				named.Debug("Verbose flag provided, all log messages will be shown")
			}

			// create the deej instance
			d, err := deej.NewDeej(logger, verbose)
			if err != nil {
				named.Fatalw("Failed to create deej object", "error", err)
				return
			}

			// if injected by build process, set version info to show up in the tray
			if buildType != "" && (versionTag != "" || gitCommit != "") {
				identifier := gitCommit
				if versionTag != "" {
					identifier = versionTag
				}

				versionString := fmt.Sprintf("Version %s-%s", buildType, identifier)
				d.SetVersion(versionString)
			}

			// onwards, to glory
			if err = d.Initialize(); err != nil {
				named.Errorw("Failed to initialize deej", "error", err)
				return
			}
		}()

		// If we get here, something failed or panicked. Wait and retry.
		fmt.Println("Deej crashed or disconnected. Restarting in 5 seconds...")
		time.Sleep(5 * time.Second)
	}
}
