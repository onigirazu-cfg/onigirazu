package facts

import (
	"reflect"
	"testing"
)

func TestParseLocalFacts(t *testing.T) {
	out := "\n" + localFactMarker + "json\n{\"a\": 1, \"b\": [\"x\"]}\n" +
		"\n" + localFactMarker + "ini\n[s]\nk = v\nn: 2\n" +
		"\n" + localFactMarker + "text\nplain words\n"
	got := parseLocalFacts(out)
	want := map[string]interface{}{
		"json": map[string]interface{}{"a": float64(1), "b": []interface{}{"x"}},
		"ini":  map[string]interface{}{"s": map[string]interface{}{"k": "v", "n": "2"}},
		"text": "plain words",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v\nwant %#v", got, want)
	}
	if len(parseLocalFacts("")) != 0 {
		t.Error("no files, no facts")
	}
}
