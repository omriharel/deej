package deej

import (
	"fmt"
	"sync"
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

// Length Returns the Number of Sliders in the ConfigMap
func (m *SliderMap) Length() int {
	return len(m.m)
}

func (m *SliderMap) iterate(f func(int, []string)) {
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
