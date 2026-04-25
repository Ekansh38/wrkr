package drill

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Stats tracks drill performance across sessions, persisted to ~/.wrkr_drill.json.
type Stats struct {
	TotalCorrect  int             `json:"totalCorrect"`
	TotalWrong    int             `json:"totalWrong"`
	MissedCounts  map[string]int  `json:"missedCounts"` // "value:toBase" - all-time miss count
	LastSession   *SessionSummary `json:"lastSession,omitempty"`
	Streak        int             `json:"streak"`        // consecutive days drilled
	LastDrillDate string          `json:"lastDrillDate"` // YYYY-MM-DD
}

// SessionSummary records what happened in a single drill session.
type SessionSummary struct {
	Correct int    `json:"correct"`
	Wrong   int    `json:"wrong"`
	Game    string `json:"game"`
}

// MissEntry is one weak-spot entry returned by TopMissed.
type MissEntry struct {
	Display string // e.g. "182 → hex"
	Count   int
}

// StatsPath returns the path to the stats file.
func StatsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".wrkr_drill.json"
	}
	return filepath.Join(home, ".wrkr_drill.json")
}

// LoadStats reads the stats file, returning empty Stats if missing or corrupt.
func LoadStats() Stats {
	data, err := os.ReadFile(StatsPath())
	if err != nil {
		return Stats{MissedCounts: map[string]int{}}
	}
	var s Stats
	if err := json.Unmarshal(data, &s); err != nil {
		return Stats{MissedCounts: map[string]int{}}
	}
	if s.MissedCounts == nil {
		s.MissedCounts = map[string]int{}
	}
	return s
}

// SaveStats writes stats to disk.
func SaveStats(s Stats) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(StatsPath(), data, 0644)
}

// Record updates stats for one answered question.
func (s *Stats) Record(value int, toBase string, correct bool) {
	if correct {
		s.TotalCorrect++
	} else {
		s.TotalWrong++
		s.MissedCounts[statsKey(value, toBase)]++
	}
}

// TopMissed returns the n most-missed entries sorted by miss count.
func (s *Stats) TopMissed(n int) []MissEntry {
	type kv struct {
		k string
		v int
	}
	var entries []kv
	for k, v := range s.MissedCounts {
		entries = append(entries, kv{k, v})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].v > entries[j].v
	})
	if len(entries) > n {
		entries = entries[:n]
	}
	result := make([]MissEntry, len(entries))
	for i, e := range entries {
		var val int
		var base string
		fmt.Sscanf(e.k, "%d:%s", &val, &base)
		result[i] = MissEntry{
			Display: fmt.Sprintf("%d → %s", val, base),
			Count:   e.v,
		}
	}
	return result
}

// UpdateStreak increments the streak if today is a new day, resets if a day
// was missed. Safe to call multiple times in one day (idempotent after first call).
func (s *Stats) UpdateStreak() {
	today := time.Now().Format("2006-01-02")
	if s.LastDrillDate == today {
		return // already counted today
	}
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	if s.LastDrillDate == yesterday {
		s.Streak++
	} else {
		s.Streak = 1 // missed a day (or first ever session)
	}
	s.LastDrillDate = today
}

func statsKey(value int, toBase string) string {
	return fmt.Sprintf("%d:%s", value, toBase)
}
