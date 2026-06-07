// Package stats provides a small thread-safe record of cron occurrences that is
// shared between the scheduler module (writer) and the api module (reader).
package stats

import (
	"sync"
	"time"
)

// ScheduleStat is the public view of a single schedule's activity.
type ScheduleStat struct {
	Ticks    int64     `json:"ticks"`
	LastTick time.Time `json:"last_tick,omitempty"`
}

// Store tracks how many times each named cron schedule has fired.
type Store struct {
	mu     sync.RWMutex
	counts map[string]int64
	last   map[string]time.Time
}

// NewStore returns an empty Store.
func NewStore() *Store {
	return &Store{
		counts: make(map[string]int64),
		last:   make(map[string]time.Time),
	}
}

// Record increments the tick counter for a named schedule, stamps the time, and
// returns the new count. It is safe for concurrent use.
func (s *Store) Record(schedule string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counts[schedule]++
	s.last[schedule] = time.Now().UTC()
	return s.counts[schedule]
}

// Snapshot returns a copy of the current counters keyed by schedule name.
func (s *Store) Snapshot() map[string]ScheduleStat {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]ScheduleStat, len(s.counts))
	for name, c := range s.counts {
		out[name] = ScheduleStat{Ticks: c, LastTick: s.last[name]}
	}
	return out
}
