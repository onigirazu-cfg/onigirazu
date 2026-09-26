package engine

import (
	"fmt"
	"strconv"
	"strings"
)

func isTemplate(v interface{}) bool {
	s, ok := v.(string)
	return ok && strings.Contains(s, "{{")
}

// serialBatches splits total hosts into the batch sizes of a play's serial:
// a number, a percentage ("30%") or a list of them whose last entry repeats
// ([1, 5, "20%"]); no serial is one batch
func serialBatches(serial interface{}, total int) ([]int, error) {
	if serial == nil || total == 0 {
		return []int{total}, nil
	}
	var steps []interface{}
	if list, ok := serial.([]interface{}); ok {
		steps = list
	} else {
		steps = []interface{}{serial}
	}
	if len(steps) == 0 {
		return []int{total}, nil
	}

	var batches []int
	for done, i := 0, 0; done < total; i++ {
		step := steps[len(steps)-1]
		if i < len(steps) {
			step = steps[i]
		}
		size, err := batchSize(step, total)
		if err != nil {
			return nil, err
		}
		if size > total-done {
			size = total - done
		}
		batches = append(batches, size)
		done += size
	}
	return batches, nil
}

// batchSize is one serial entry as a number of hosts, at least 1; 0 means
// all hosts
func batchSize(step interface{}, total int) (int, error) {
	var n int
	switch v := step.(type) {
	case int:
		n = v
	case float64:
		n = int(v)
	case string:
		s := strings.TrimSpace(v)
		if pct, ok := strings.CutSuffix(s, "%"); ok {
			p, err := strconv.ParseFloat(strings.TrimSpace(pct), 64)
			if err != nil {
				return 0, fmt.Errorf("serial: %q is not a percentage", v)
			}
			n = int(float64(total) * p / 100)
			if n < 1 {
				n = 1
			}
			return n, nil
		}
		parsed, err := strconv.Atoi(s)
		if err != nil {
			return 0, fmt.Errorf("serial: %q is not a number or a percentage", v)
		}
		n = parsed
	default:
		return 0, fmt.Errorf("serial: unexpected %T", step)
	}
	if n <= 0 {
		return total, nil
	}
	return n, nil
}
