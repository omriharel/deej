package deej

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/omriharel/deej/pkg/deej/util"
)

const (
	crashlogFilename        = "deej-crash-%s.log"
	crashlogTimestampFormat = "2006.01.02-15.04.05"

	crashMessage = `-----------------------------------------------------------------
                        deej crashlog
-----------------------------------------------------------------
Unfortunately, deej has crashed. This really shouldn't happen!
If you've just encountered this, please contact @omriharel and attach this error log.
You can also join the deej Discord server at https://discord.gg/nf88NJu.
-----------------------------------------------------------------
Time: %s
Panic occurred: %s
Stack trace:
%s
-----------------------------------------------------------------
`
)

func (d *Deej) recoverFromPanic() {
	r := recover()

	if r == nil {
		return
	}

	// if we got here, we're recovering from a panic!
	now := time.Now()

	// that would suck
	if err := util.EnsureDirExists(logDirectory); err != nil {
		panic(fmt.Errorf("ensure crashlog dir exists: %w", err))
	}

	crashlogBytes := bytes.NewBufferString(fmt.Sprintf(crashMessage, now.Format(crashlogTimestampFormat), r, debug.Stack()))
	crashlogPath := filepath.Join(logDirectory, fmt.Sprintf(crashlogFilename, now.Format(crashlogTimestampFormat)))

	// that would REALLY suck
	if err := ioutil.WriteFile(crashlogPath, crashlogBytes.Bytes(), os.ModePerm); err != nil {
		panic(fmt.Errorf("can't even write the crashlog file contents: %w", err))
	}

	d.logger.Errorw("Encountered and logged panic, crashing",
		"crashlogPath", crashlogPath,
		"error", r)

	d.notifier.Notify("Unexpected crash occurred...",
		fmt.Sprintf("More details in %s", crashlogPath))

	// bye :(
	d.signalStop()
	d.logger.Errorw("Quitting", "exitCode", 1)
	os.Exit(1)
}
