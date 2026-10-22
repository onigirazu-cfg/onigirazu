package cli

import (
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ServeAt accepts one anchored schedule or a list of schedules.
type ServeAt []string

func (a *ServeAt) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		if node.Tag != "!!str" {
			return fmt.Errorf("at: expected a string or list of strings")
		}
		*a = ServeAt{node.Value}
	case yaml.SequenceNode:
		values := make(ServeAt, 0, len(node.Content))
		for _, entry := range node.Content {
			if entry.Kind != yaml.ScalarNode || entry.Tag != "!!str" {
				return fmt.Errorf("at: expected a string or list of strings")
			}
			values = append(values, entry.Value)
		}
		*a = values
	default:
		return fmt.Errorf("at: expected a string or list of strings")
	}
	return nil
}

// UnmarshalYAML keeps malformed schedule values associated with their job.
func (j *ServeJob) UnmarshalYAML(node *yaml.Node) error {
	type plain ServeJob
	var decoded plain
	if err := node.Decode(&decoded); err != nil {
		name := ""
		for i := 0; i+1 < len(node.Content); i += 2 {
			if node.Content[i].Value == "name" {
				name = node.Content[i+1].Value
				break
			}
		}
		return fmt.Errorf("job %s: %w", name, err)
	}
	*j = ServeJob(decoded)
	return nil
}

type serveAt struct {
	days   [7]bool
	minute int
}

func parseServeDays(spec string) ([7]bool, error) {
	var selected [7]bool
	days := map[string]int{"sun": 0, "mon": 1, "tue": 2, "wed": 3, "thu": 4, "fri": 5, "sat": 6}
	for _, part := range strings.Split(spec, ",") {
		if part == "*" {
			for i := range selected {
				selected[i] = true
			}
			continue
		}
		bounds := strings.Split(part, "-")
		first, ok := days[bounds[0]]
		if !ok || len(bounds) > 2 {
			return selected, fmt.Errorf("invalid days %q", spec)
		}
		last := first
		if len(bounds) == 2 {
			last, ok = days[bounds[1]]
			if !ok {
				return selected, fmt.Errorf("invalid days %q", spec)
			}
		}
		for d := first; ; d = (d + 1) % 7 {
			selected[d] = true
			if d == last {
				break
			}
		}
	}
	return selected, nil
}

func parseServeClock(s string) (int, error) {
	t, err := time.Parse("15:04", s)
	if err != nil || len(s) != 5 {
		return 0, fmt.Errorf("invalid time %q (expected HH:MM)", s)
	}
	return t.Hour()*60 + t.Minute(), nil
}

func validateServeMissed(policy string) error {
	if policy != "" && policy != "run_once" && policy != "skip" {
		return fmt.Errorf("invalid missed policy %q", policy)
	}
	return nil
}

func (j *ServeJob) prepareSchedule(globalMissed string) error {
	fail := func(err error) error { return fmt.Errorf("job %s: %w", j.Name, err) }
	if err := validateServeMissed(globalMissed); err != nil {
		return fail(fmt.Errorf("global %w", err))
	}
	if err := validateServeMissed(j.Missed); err != nil {
		return fail(err)
	}
	if j.Every < 0 {
		return fail(fmt.Errorf("every must not be negative"))
	}
	if j.Every != 0 && len(j.At) > 0 {
		return fail(fmt.Errorf("every and at are exclusive"))
	}
	if j.AtTZ != "" && len(j.At) == 0 {
		return fail(fmt.Errorf("at_tz requires at"))
	}
	j.at, j.atLoc = nil, nil
	if len(j.At) > 0 {
		zone := j.AtTZ
		if zone == "" {
			zone = "UTC"
		}
		loc, err := time.LoadLocation(zone)
		if err != nil {
			return fail(fmt.Errorf("at_tz: %w", err))
		}
		entries := make([]serveAt, 0, len(j.At))
		for _, spec := range j.At {
			fields := strings.Fields(spec)
			if len(fields) != 2 {
				return fail(fmt.Errorf("at: invalid %q (expected <days> HH:MM)", spec))
			}
			days, err := parseServeDays(fields[0])
			if err != nil {
				return fail(fmt.Errorf("at: %w", err))
			}
			minute, err := parseServeClock(fields[1])
			if err != nil {
				return fail(fmt.Errorf("at: %w", err))
			}
			entries = append(entries, serveAt{days: days, minute: minute})
		}
		j.AtTZ, j.atLoc, j.at = zone, loc, entries
	}
	j.missed = j.Missed
	if j.missed == "" {
		j.missed = globalMissed
	}
	if j.missed == "" {
		j.missed = "run_once"
		if j.Kind == "apply" {
			j.missed = "skip"
		}
	}
	return nil
}

func (j *ServeJob) hasSchedule() bool { return j.Every > 0 || len(j.At) > 0 }

func (j *ServeJob) scheduleSignature() string {
	if j.Every > 0 {
		return "every=" + j.Every.String()
	}
	return "at=" + strings.Join(j.At, ";") + ";tz=" + j.AtTZ
}

func (j *ServeJob) nextAt(after time.Time) time.Time {
	local := after.In(j.atLoc)
	var next time.Time
	for day := 0; day < 8; day++ {
		date := time.Date(local.Year(), local.Month(), local.Day()+day, 12, 0, 0, 0, j.atLoc)
		for _, entry := range j.at {
			if !entry.days[date.Weekday()] {
				continue
			}
			// time.Date normalizes missing local times; prefer the first repeated time.
			candidate := time.Date(date.Year(), date.Month(), date.Day(), entry.minute/60, entry.minute%60, 0, 0, j.atLoc)
			_, offset := candidate.Zone()
			for _, nearby := range []time.Time{candidate.Add(-24 * time.Hour), candidate.Add(24 * time.Hour)} {
				_, otherOffset := nearby.Zone()
				first := candidate.Add(time.Duration(offset-otherOffset) * time.Second)
				if first.Before(candidate) && first.Year() == candidate.Year() && first.YearDay() == candidate.YearDay() && first.Hour() == candidate.Hour() && first.Minute() == candidate.Minute() {
					candidate = first
					_, offset = candidate.Zone()
				}
			}
			if candidate.After(after) && (next.IsZero() || candidate.Before(next)) {
				next = candidate
			}
		}
	}
	return next.UTC()
}
