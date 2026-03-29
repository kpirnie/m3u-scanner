// Package cron implements a minimal five-field cron scheduler.
// Supported syntax: "minute hour dom month dow" with numeric values,
// "*" (any), "*/n" (step), "a-b" (range), and "a,b" (list).
// Named shortcuts (@daily, @hourly, etc.) are also supported.
//
// No external dependency — keeps the binary lean.
package cron

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

// Job is a scheduled function with a parsed cron schedule.
type Job struct {
	fields [5]field
	fn     func()
}

// New parses a five-field cron expression and returns a Job.
// Returns an error if the expression is invalid.
func New(expr string, fn func()) (*Job, error) {
	expr = expandShortcut(expr)
	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return nil, fmt.Errorf("expected 5 fields, got %d in %q", len(parts), expr)
	}

	// [min, max] bounds for: minute, hour, dom, month, dow
	limits := [5][2]int{{0, 59}, {0, 23}, {1, 31}, {1, 12}, {0, 6}}
	var fields [5]field
	for i, p := range parts {
		f, err := parseField(p, limits[i][0], limits[i][1])
		if err != nil {
			return nil, fmt.Errorf("field %d (%q): %w", i+1, p, err)
		}
		fields[i] = f
	}

	return &Job{fields: fields, fn: fn}, nil
}

// Run blocks forever, sleeping until the next scheduled time then calling fn
// in a new goroutine. Should be called in a dedicated goroutine.
func (j *Job) Run() {
	for {
		next := j.next(time.Now())
		log.Printf("[cron] next run at %s", next.Format("2006-01-02 15:04 MST"))
		time.Sleep(time.Until(next))
		log.Println("[cron] triggered scheduled rescan")
		go j.fn()
		// Sleep past the current minute before searching for the next slot
		time.Sleep(61 * time.Second)
	}
}

// next returns the earliest time after t that matches the schedule.
func (j *Job) next(t time.Time) time.Time {
	t = t.Truncate(time.Minute).Add(time.Minute)
	limit := t.Add(4 * 365 * 24 * time.Hour)

	for t.Before(limit) {
		switch {
		case !j.fields[3].matches(int(t.Month())):
			t = time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, t.Location())
		case !j.fields[2].matches(t.Day()) || !j.fields[4].matches(int(t.Weekday())):
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, t.Location())
		case !j.fields[1].matches(t.Hour()):
			t = t.Truncate(time.Hour).Add(time.Hour)
		case !j.fields[0].matches(t.Minute()):
			t = t.Add(time.Minute)
		default:
			return t
		}
	}
	return limit
}

// ── Field ─────────────────────────────────────────────────────────────────────

type field struct {
	values map[int]bool // nil = wildcard (matches any value)
}

func (f field) matches(v int) bool {
	if f.values == nil {
		return true
	}
	return f.values[v]
}

func parseField(s string, min, max int) (field, error) {
	if s == "*" {
		return field{}, nil
	}

	values := map[int]bool{}

	for _, part := range strings.Split(s, ",") {
		if err := parsePart(part, min, max, values); err != nil {
			return field{}, err
		}
	}

	return field{values: values}, nil
}

func parsePart(s string, min, max int, out map[int]bool) error {
	if idx := strings.Index(s, "/"); idx != -1 {
		// Step: * /n  or  start/n
		step, err := strconv.Atoi(s[idx+1:])
		if err != nil || step <= 0 {
			return fmt.Errorf("invalid step in %q", s)
		}
		start := min
		if s[:idx] != "*" {
			start, err = strconv.Atoi(s[:idx])
			if err != nil {
				return fmt.Errorf("invalid step start in %q", s)
			}
		}
		for v := start; v <= max; v += step {
			out[v] = true
		}
		return nil
	}

	if idx := strings.Index(s, "-"); idx != -1 {
		// Range: a-b
		a, err1 := strconv.Atoi(s[:idx])
		b, err2 := strconv.Atoi(s[idx+1:])
		if err1 != nil || err2 != nil || a > b {
			return fmt.Errorf("invalid range %q", s)
		}
		for v := a; v <= b; v++ {
			out[v] = true
		}
		return nil
	}

	// Single value
	v, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("invalid value %q", s)
	}
	if v < min || v > max {
		return fmt.Errorf("value %d out of range [%d,%d]", v, min, max)
	}
	out[v] = true
	return nil
}

func expandShortcut(expr string) string {
	switch strings.TrimSpace(expr) {
	case "@yearly", "@annually":
		return "0 0 1 1 *"
	case "@monthly":
		return "0 0 1 * *"
	case "@weekly":
		return "0 0 * * 0"
	case "@daily", "@midnight":
		return "0 0 * * *"
	case "@hourly":
		return "0 * * * *"
	default:
		return expr
	}
}
