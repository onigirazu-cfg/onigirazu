package expression

import (
	"fmt"
	"hash/fnv"
	"math/rand"
	"reflect"
	"time"
)

// randomOptions reads the arguments after the value: start, step and seed,
// positional or as "name", value pairs (from start=, step=, seed=)
func randomOptions(params []interface{}) (start, step int, seed interface{}, err error) {
	step = 1
	positional := 0
	for i := 0; i < len(params); i++ {
		if name, ok := params[i].(string); ok && i+1 < len(params) && (name == "seed" || name == "start" || name == "step") {
			i++
			switch name {
			case "seed":
				seed = params[i]
			case "start":
				start, err = intArg(params[i])
			case "step":
				step, err = intArg(params[i])
			}
		} else {
			switch positional {
			case 0:
				start, err = intArg(params[i])
			case 1:
				step, err = intArg(params[i])
			case 2:
				seed = params[i]
			}
			positional++
		}
		if err != nil {
			return 0, 0, nil, err
		}
	}
	if step <= 0 {
		return 0, 0, nil, fmt.Errorf("random: step must be positive")
	}
	return start, step, seed, nil
}

func intArg(v interface{}) (int, error) {
	switch n := v.(type) {
	case int:
		return n, nil
	case int64:
		return int(n), nil
	case float64:
		return int(n), nil
	}
	var i int
	if _, err := fmt.Sscan(fmt.Sprint(v), &i); err != nil {
		return 0, fmt.Errorf("random: %v is not a number", v)
	}
	return i, nil
}

// source is seeded by seed (the same seed gives the same numbers), or by the
// clock
func source(seed interface{}) *rand.Rand {
	if seed == nil {
		return rand.New(rand.NewSource(time.Now().UnixNano())) // #nosec G404 -- not for secrets
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(fmt.Sprint(seed)))
	return rand.New(rand.NewSource(int64(h.Sum64() >> 1))) // #nosec G404 G115 -- not for secrets; 63 bits fit
}

// jinjaRandom is Ansible's random: a random item of a list or character of a
// string, or for a number N one of start, start+step, ... below N
func jinjaRandom(p ...interface{}) (interface{}, error) {
	start, step, seed, err := randomOptions(p[1:])
	if err != nil {
		return nil, err
	}
	r := source(seed)
	switch v := p[0].(type) {
	case string:
		if v == "" {
			return nil, fmt.Errorf("random: empty string")
		}
		runes := []rune(v)
		return string(runes[r.Intn(len(runes))]), nil
	case int, int64, float64:
		end, _ := intArg(v)
		if end <= start {
			return nil, fmt.Errorf("random: empty range from %d to %d", start, end)
		}
		n := (end - start + step - 1) / step
		return start + r.Intn(n)*step, nil
	}
	rv := reflect.ValueOf(p[0])
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		if rv.Len() == 0 {
			return nil, fmt.Errorf("random: empty list")
		}
		return rv.Index(r.Intn(rv.Len())).Interface(), nil
	}
	return nil, fmt.Errorf("random: expected a list, string or number, got %T", p[0])
}

// jinjaShuffle returns the items of a list in random order (seed as random)
func jinjaShuffle(p ...interface{}) (interface{}, error) {
	_, _, seed, err := randomOptions(p[1:])
	if err != nil {
		return nil, err
	}
	rv := reflect.ValueOf(p[0])
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, fmt.Errorf("shuffle: expected a list, got %T", p[0])
	}
	out := make([]interface{}, rv.Len())
	for i := range out {
		out[i] = rv.Index(i).Interface()
	}
	source(seed).Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out, nil
}
