package metrics

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Counters is a thread-safe in-memory counter store.
type Counters struct {
	mu     sync.RWMutex
	values map[string]uint64
}

func NewCounters() *Counters {
	return &Counters{
		values: make(map[string]uint64),
	}
}

func (c *Counters) Inc(name string) uint64 {
	return c.Add(name, 1)
}

func (c *Counters) Add(name string, delta uint64) uint64 {
	normalized := normalizeName(name)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.values[normalized] += delta
	return c.values[normalized]
}

func (c *Counters) Get(name string) uint64 {
	normalized := normalizeName(name)

	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.values[normalized]
}

func (c *Counters) Snapshot() map[string]uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	snapshot := make(map[string]uint64, len(c.values))
	for k, v := range c.values {
		snapshot[k] = v
	}

	return snapshot
}

// PlainText renders counters in a stable, sorted plain text format.
func (c *Counters) PlainText() string {
	snapshot := c.Snapshot()
	if len(snapshot) == 0 {
		return "no_counters 0\n"
	}

	names := make([]string, 0, len(snapshot))
	for name := range snapshot {
		names = append(names, name)
	}
	sort.Strings(names)

	var b strings.Builder
	for _, name := range names {
		fmt.Fprintf(&b, "%s %d\n", name, snapshot[name])
	}

	return b.String()
}

func normalizeName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return "unnamed_counter"
	}

	var b strings.Builder
	b.Grow(len(name))

	for i := 0; i < len(name); i++ {
		ch := name[i]
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_' {
			b.WriteByte(ch)
			continue
		}
		b.WriteByte('_')
	}

	return b.String()
}
