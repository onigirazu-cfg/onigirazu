package managed

import "testing"

func TestFindIndex(t *testing.T) {
	s := &State{Resources: []*Record{{Host: "a", Type: "file", ID: "/x"}, {Host: "b", Type: "file", ID: "/x"}}}
	if r := s.Find("b", "file", "/x"); r != s.Resources[1] {
		t.Fatalf("find b: %v", r)
	}
	// a clone finds its own records, not the original's
	c := s.Clone()
	if r := c.Find("a", "file", "/x"); r == s.Resources[0] || r != c.Resources[0] {
		t.Fatalf("clone found %p, want %p", r, c.Resources[0])
	}
	// a list changed under the index
	s.Resources = append(s.Resources, &Record{Host: "c", Type: "user", ID: "u"})
	if s.Find("c", "user", "u") == nil {
		t.Fatal("appended record not found")
	}
	if !s.Remove("a", "file", "/x") || s.Find("a", "file", "/x") != nil {
		t.Fatal("removed record still found")
	}
}
