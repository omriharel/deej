package deej

import (
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/jax-b/deej/util"
)

// CanonicalConfig provides application-wide access to configuration fields,
// as well as loading/file watching logic for deej's configuration file
type CanonicalConfig struct {
	SliderMapping *SliderMap

	// renamed from ProcessRefreshFrequency, key left as is in yaml-config for backwards compatibility
	SessionRefreshThreshold time.Duration

	ConnectionInfo struct {
		COMPort  string
		BaudRate int
	}

	InvertSliders bool

	logger             *zap.SugaredLogger
	notifier           Notifier
	stopWatcherChannel chan bool
	sliderMapMutex     sync.Locker

	reloadConsumers []chan bool
}

type marshalledConfig struct {
	SliderMapping           map[int]interface{} `yaml:"slider_mapping"`
	ProcessRefreshFrequency int                 `yaml:"process_refresh_frequency"`
	COMPort                 string              `yaml:"com_port"`
	BaudRate                int                 `yaml:"baud_rate"`
	InvertSliders           bool                `yaml:"invert_sliders"`
}

const (
	configFilepath = "config.yaml"

	defaultProcessRefreshFrequency = 5 * time.Second
	defaultCOMPort                 = "COM4"
	defaultBaudRate                = 9600
)

var defaultSliderMapping = func() *SliderMap {
	emptyMap := newSliderMap()
	emptyMap.set(0, []string{masterSessionName})
	return emptyMap
}()

// NewConfig creates a config instance for the deej object
func NewConfig(logger *zap.SugaredLogger, notifier Notifier) (*CanonicalConfig, error) {
	logger = logger.Named("config")

	cc := &CanonicalConfig{
		logger:             logger,
		notifier:           notifier,
		reloadConsumers:    []chan bool{},
		stopWatcherChannel: make(chan bool),
	}

	logger.Debug("Created config instance")

	return cc, nil
}

// Load reads a config file from disk and tries to parse it
func (cc *CanonicalConfig) Load() error {
	cc.logger.Debugw("Loading config", "path", configFilepath)

	// make sure it exists
	if !util.FileExists(configFilepath) {
		cc.logger.Warnw("Config file not found", "path", configFilepath)
		cc.notifier.Notify("Can't find configuration!",
			fmt.Sprintf("%s must be in the same directory as deej. Please re-launch", configFilepath))

		return fmt.Errorf("config file doesn't exist: %s", configFilepath)
	}

	// open->read->close the file
	configBytes, err := ioutil.ReadFile(configFilepath)
	if err != nil {
		cc.logger.Warnw("Failed to read config file", "error", err)
		return fmt.Errorf("read config file: %w", err)
	}

	// unmarshall it into the yaml-aware struct
	mc := &marshalledConfig{}
	if err := yaml.Unmarshal(configBytes, mc); err != nil {
		cc.logger.Warnw("Failed to unmarhsal config into struct", "error", err)
		cc.notifier.Notify("Invalid configuration!",
			"Please make sure config.yaml is a valid YAML format.")

		return fmt.Errorf("unmarshall yaml config: %w", err)
	}

	// canonize it
	if err := cc.populateFromMarshalled(mc); err != nil {
		cc.logger.Warnw("Failed to populate config fields from marshalled struct", "error", err)
		return fmt.Errorf("populate config fields: %w", err)
	}

	cc.logger.Info("Loaded config successfully")
	cc.logger.Infow("Config values",
		"sliderMapping", cc.SliderMapping,
		"sessionRefreshThreshold", cc.SessionRefreshThreshold,
		"connectionInfo", cc.ConnectionInfo,
		"invertSliders", cc.InvertSliders)

	return nil
}

// SubscribeToChanges allows external components to receive updates when the config is reloaded
func (cc *CanonicalConfig) SubscribeToChanges() chan bool {
	c := make(chan bool)
	cc.reloadConsumers = append(cc.reloadConsumers, c)

	return c
}

// WatchConfigFileChanges starts watching for configuration file changes
// and attempts reloading the config when they happen
func (cc *CanonicalConfig) WatchConfigFileChanges() {
	cc.logger.Debugw("Starting to watch config file for changes", "path", configFilepath)

	// set up the watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		cc.logger.Warnw("Failed to create filesystem watcher", "error", err)
	}

	// clean it up when done
	defer func() {
		if err := watcher.Close(); err != nil {
			cc.logger.Warnw("Failed to close filesystem watcher", "error", err)
		}
	}()

	// start watcher event loop
	go func() {
		const (
			minTimeBetweenReloadAttempts = time.Millisecond * 500
			delayBetweenEventAndReload   = time.Millisecond * 50
		)

		lastAttemptedReload := time.Now()

		// listen for watcher events
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// when we get a write event...
				if event.Op&fsnotify.Write == fsnotify.Write {

					now := time.Now()

					// ... check if it's not a duplicate (many editors will write to a file twice)
					if lastAttemptedReload.Add(minTimeBetweenReloadAttempts).Before(now) {

						// and attempt reload if appropriate
						cc.logger.Debugw("Config file modified, attempting reload", "event", event)

						// wait a bit to let the editor actually flush the new file contents
						<-time.After(delayBetweenEventAndReload)

						if err = cc.Load(); err != nil {
							cc.logger.Warnw("Failed to reload config file", "error", err)
						} else {
							cc.logger.Info("Reloaded config successfully")
							cc.notifier.Notify("Configuration reloaded!", "Your changes have been applied.")

							cc.onConfigReloaded()
						}

						// don't forget to update the time
						lastAttemptedReload = now
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}

				cc.logger.Warnw("Filesystem watcher encountered an error", "error", err)
			}
		}
	}()

	// add the config file path
	if err = watcher.Add(configFilepath); err != nil {
		cc.logger.Warnw("Failed to add config filepath to fs watcher", "error", err)
	}

	// wait till they stop us
	<-cc.stopWatcherChannel
	cc.logger.Debug("Stopping filesystem watcher")
}

// StopWatchingConfigFile signals our filesystem watcher to stop
func (cc *CanonicalConfig) StopWatchingConfigFile() {
	cc.stopWatcherChannel <- true
}

func (cc *CanonicalConfig) populateFromMarshalled(mc *marshalledConfig) error {

	// start by loading the slider mapping because it's the only failable part for now
	if mc.SliderMapping == nil {
		cc.logger.Warnw("Missing key in config, using default value",
			"key", "slider_mapping",
			"value", defaultSliderMapping)

		cc.SliderMapping = defaultSliderMapping
	} else {

		sliderMapping := newSliderMap()

		// this is where we need to parse out each value (which is an interface{} at this point),
		// and type-assert it into either a string or a list of strings
		for key, value := range mc.SliderMapping {
			switch typedValue := value.(type) {
			case string:
				if typedValue == "" {
					sliderMapping.set(key, []string{})
				} else {
					sliderMapping.set(key, []string{typedValue})
				}

			// silently ignore nil values and treat as no targets
			case nil:
				sliderMapping.set(key, []string{})

			// we can't directly type-assert to a []string, so we must check each item. yup, that sucks
			case []interface{}:
				sliderItems := []string{}

				for _, listItem := range typedValue {

					// silently ignore nil values
					if listItem == nil {
						continue
					}

					listItemStr, ok := listItem.(string)
					if !ok {
						cc.logger.Warnw("Non-string value in slider mapping list",
							"key", key,
							"value", listItem,
							"valueType", fmt.Sprintf("%t", listItem))

						return fmt.Errorf("invalid slider mapping for slider %d: got type %t, need string or []string", key, typedValue)
					}

					// ignore empty strings
					if listItemStr != "" {
						sliderItems = append(sliderItems, listItemStr)
					}
				}

				sliderMapping.set(key, sliderItems)
			default:
				cc.logger.Warnw("Invalid value for slider mapping key",
					"key", key,
					"value", typedValue,
					"valueType", fmt.Sprintf("%t", typedValue))

				return fmt.Errorf("invalid slider mapping for slider %d: got type %t, need string or []string", key, typedValue)
			}
		}

		cc.SliderMapping = sliderMapping
	}

	// for each config field, check if non-zero and populate its equivalent in our canonical config
	// for zero-value fields, log and use the default constant defined above
	// silently ignore invalid amount of seconds
	if mc.ProcessRefreshFrequency <= 0 {
		cc.logger.Warnw("Missing key in config, using default value",
			"key", "process_refresh_frequency",
			"value", defaultProcessRefreshFrequency)

		cc.SessionRefreshThreshold = defaultProcessRefreshFrequency
	} else {
		cc.SessionRefreshThreshold = time.Duration(mc.ProcessRefreshFrequency) * time.Second
	}

	if mc.COMPort == "" {
		cc.logger.Warnw("Missing key in config, using default value",
			"key", "com_port",
			"value", defaultCOMPort)

		cc.ConnectionInfo.COMPort = defaultCOMPort
	} else {
		cc.ConnectionInfo.COMPort = mc.COMPort
	}

	// silently ignore invalid baud rates
	if mc.BaudRate <= 0 {
		cc.logger.Warnw("Missing key in config, using default value",
			"key", "baud_rate",
			"value", defaultBaudRate)

		cc.ConnectionInfo.BaudRate = defaultBaudRate
	} else {
		cc.ConnectionInfo.BaudRate = mc.BaudRate
	}

	// if the key isn't found this will default to false, which is what we want
	cc.InvertSliders = mc.InvertSliders

	cc.logger.Debug("Populated config fields from marshalled config object")

	return nil
}

func (cc *CanonicalConfig) onConfigReloaded() {
	cc.logger.Debug("Notifying consumers about configuration reload")

	for _, consumer := range cc.reloadConsumers {
		consumer <- true
	}
}
