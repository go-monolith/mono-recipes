package scheduler

import (
	"sync"
	"time"

	"github.com/example/cron-scheduler-demo/modules/scheduler/contract"
)

// statsStore is the scheduler's private, thread-safe record of cron
// occurrences. Other modules read it only through the contract.GetStats
// service, never by sharing the store.
type statsStore struct {
	mu     sync.RWMutex
	counts map[string]int64
	last   map[string]time.Time
}

// newStatsStore returns an empty store.
func newStatsStore() *statsStore {
	return &statsStore{
		counts: make(map[string]int64),
		last:   make(map[string]time.Time),
	}
}

// Record increments the tick counter for a named schedule, stamps the time, and
// returns the new count. It is safe for concurrent use.
func (s *statsStore) Record(schedule string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counts[schedule]++
	s.last[schedule] = time.Now().UTC()
	return s.counts[schedule]
}

// Snapshot returns a copy of the current counters keyed by schedule name.
func (s *statsStore) Snapshot() map[string]contract.ScheduleStat {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]contract.ScheduleStat, len(s.counts))
	for name, c := range s.counts {
		// Copy the value before taking its address so the pointer doesn't alias
		// into the map (and so each snapshot entry is independent).
		ts := s.last[name]
		out[name] = contract.ScheduleStat{Ticks: c, LastTick: &ts}
	}
	return out
}
