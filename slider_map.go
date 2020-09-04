package deej

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/thoas/go-funk"
)

// SliderMap Holder of the slider map
type SliderMap struct {
	m    map[int][]string
	lock sync.Locker
}

func newSliderMap() *SliderMap {
	return &SliderMap{
		m:    make(map[int][]string),
		lock: &sync.Mutex{},
	}
}

func sliderMapFromConfigs(userMapping map[string][]string, internalMapping map[string][]string) *SliderMap {
	resultMap := newSliderMap()

	// copy targets from user config, ignoring empty values
	for sliderIdxString, targets := range userMapping {
		sliderIdx, _ := strconv.Atoi(sliderIdxString)

		resultMap.set(sliderIdx, funk.FilterString(targets, func(s string) bool {
			return s != ""
		}))
	}

	// add targets from internal configs, ignoring duplicate or empty values
	for sliderIdxString, targets := range internalMapping {
		sliderIdx, _ := strconv.Atoi(sliderIdxString)

		existingTargets, ok := resultMap.Get(sliderIdx)
		if !ok {
			existingTargets = []string{}
		}

		filteredTargets := funk.FilterString(targets, func(s string) bool {
			return (!funk.ContainsString(existingTargets, s)) && s != ""
		})

		existingTargets = append(existingTargets, filteredTargets...)
		resultMap.set(sliderIdx, existingTargets)
	}

	return resultMap
}

//Iterate through each value in the map
func (m *SliderMap) Iterate(f func(int, []string)) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for key, value := range m.m {
		f(key, value)
	}
}

// Get Returns a Key at a certin Index
func (m *SliderMap) Get(key int) ([]string, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	value, ok := m.m[key]
	return value, ok
}

func (m *SliderMap) set(key int, value []string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.m[key] = value
}

func (m *SliderMap) String() string {
	m.lock.Lock()
	defer m.lock.Unlock()

	sliderCount := 0
	targetCount := 0

	for _, value := range m.m {
		sliderCount++
		targetCount += len(value)
	}

	return fmt.Sprintf("<%d sliders mapped to %d targets>", sliderCount, targetCount)
}
