package tagfilter

import (
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		tagsStr     string
		skipTagsStr string
		expectMode  FilterMode
		expectTags  []string
		expectSkip  []string
	}{
		{
			name:        "empty filters",
			tagsStr:     "",
			skipTagsStr: "",
			expectMode:  ModeAll,
		},
		{
			name:       "tagged mode",
			tagsStr:    "tagged",
			expectMode: ModeTagged,
		},
		{
			name:       "untagged mode",
			tagsStr:    "untagged",
			expectMode: ModeUntagged,
		},
		{
			name:       "all mode explicit",
			tagsStr:    "all",
			expectMode: ModeAll,
		},
		{
			name:       "specific tags",
			tagsStr:    "setup,config,deploy",
			expectMode: ModeSpecific,
			expectTags: []string{"setup", "config", "deploy"},
		},
		{
			name:        "skip tags",
			skipTagsStr: "debug,test",
			expectSkip:  []string{"debug", "test"},
		},
		{
			name:        "tags and skip tags",
			tagsStr:     "setup,config",
			skipTagsStr: "debug",
			expectMode:  ModeSpecific,
			expectTags:  []string{"setup", "config"},
			expectSkip:  []string{"debug"},
		},
		{
			name:        "whitespace handling",
			tagsStr:     "  setup  , config  ",
			skipTagsStr: "  debug  ",
			expectMode:  ModeSpecific,
			expectTags:  []string{"setup", "config"},
			expectSkip:  []string{"debug"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := New(tt.tagsStr, tt.skipTagsStr)
			if err != nil {
				t.Fatalf("New() error: %v", err)
			}

			if f.Mode != tt.expectMode {
				t.Errorf("Mode: got %d, want %d", f.Mode, tt.expectMode)
			}

			if len(f.Tags) != len(tt.expectTags) {
				t.Errorf("Tags length: got %d, want %d", len(f.Tags), len(tt.expectTags))
			} else {
				for i, tag := range f.Tags {
					if tag != tt.expectTags[i] {
						t.Errorf("Tags[%d]: got %q, want %q", i, tag, tt.expectTags[i])
					}
				}
			}

			if len(f.SkipTags) != len(tt.expectSkip) {
				t.Errorf("SkipTags length: got %d, want %d", len(f.SkipTags), len(tt.expectSkip))
			} else {
				for i, tag := range f.SkipTags {
					if tag != tt.expectSkip[i] {
						t.Errorf("SkipTags[%d]: got %q, want %q", i, tag, tt.expectSkip[i])
					}
				}
			}
		})
	}
}

func TestShouldRun_TagAlways(t *testing.T) {
	tests := []struct {
		name        string
		tagsStr     string
		skipTagsStr string
		taskTags    []string
		expect      bool
	}{
		{
			name:     "always tag with no filters",
			tagsStr:  "",
			taskTags: []string{"always"},
			expect:   true,
		},
		{
			name:        "always tag with skip_tags",
			skipTagsStr: "setup",
			taskTags:    []string{"always"},
			expect:      true,
		},
		{
			name:        "always and skip always",
			skipTagsStr: "always",
			taskTags:    []string{"always"},
			expect:      false,
		},
		{
			name:     "always with other tags",
			tagsStr:  "setup",
			taskTags: []string{"always", "setup"},
			expect:   true,
		},
		{
			name:     "always ignored in tagged mode",
			tagsStr:  "tagged",
			taskTags: []string{"always"},
			expect:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, _ := New(tt.tagsStr, tt.skipTagsStr)
			result := f.ShouldRun(tt.taskTags)
			if result != tt.expect {
				t.Errorf("ShouldRun(%v): got %v, want %v", tt.taskTags, result, tt.expect)
			}
		})
	}
}

func TestShouldRun_TagNever(t *testing.T) {
	tests := []struct {
		name     string
		tagsStr  string
		taskTags []string
		expect   bool
	}{
		{
			name:     "never tag",
			tagsStr:  "",
			taskTags: []string{"never"},
			expect:   false,
		},
		{
			name:     "never tag with specific tags filter",
			tagsStr:  "setup",
			taskTags: []string{"never"},
			expect:   false,
		},
		{
			name:     "never with other tags",
			tagsStr:  "setup",
			taskTags: []string{"never", "setup"},
			expect:   false,
		},
		{
			name:     "never in untagged mode",
			tagsStr:  "untagged",
			taskTags: []string{"never"},
			expect:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, _ := New(tt.tagsStr, "")
			result := f.ShouldRun(tt.taskTags)
			if result != tt.expect {
				t.Errorf("ShouldRun(%v): got %v, want %v", tt.taskTags, result, tt.expect)
			}
		})
	}
}

func TestShouldRun_ModeTagged(t *testing.T) {
	tests := []struct {
		name     string
		taskTags []string
		expect   bool
	}{
		{
			name:     "has tags",
			taskTags: []string{"setup", "config"},
			expect:   true,
		},
		{
			name:     "no tags",
			taskTags: []string{},
			expect:   false,
		},
		{
			name:     "single tag",
			taskTags: []string{"deploy"},
			expect:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, _ := New("tagged", "")
			result := f.ShouldRun(tt.taskTags)
			if result != tt.expect {
				t.Errorf("ShouldRun(%v) in tagged mode: got %v, want %v", tt.taskTags, result, tt.expect)
			}
		})
	}
}

func TestShouldRun_ModeUntagged(t *testing.T) {
	tests := []struct {
		name     string
		taskTags []string
		expect   bool
	}{
		{
			name:     "no tags",
			taskTags: []string{},
			expect:   true,
		},
		{
			name:     "has tags",
			taskTags: []string{"setup"},
			expect:   false,
		},
		{
			name:     "multiple tags",
			taskTags: []string{"setup", "config"},
			expect:   false,
		},
		{
			name:     "always tag (always overrides)",
			taskTags: []string{"always"},
			expect:   true, // 'always' tag overrides untagged mode
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, _ := New("untagged", "")
			result := f.ShouldRun(tt.taskTags)
			if result != tt.expect {
				t.Errorf("ShouldRun(%v) in untagged mode: got %v, want %v", tt.taskTags, result, tt.expect)
			}
		})
	}
}

func TestShouldRun_ModeSpecific(t *testing.T) {
	tests := []struct {
		name     string
		tagsStr  string
		taskTags []string
		expect   bool
	}{
		{
			name:     "matching tag",
			tagsStr:  "setup",
			taskTags: []string{"setup"},
			expect:   true,
		},
		{
			name:     "one of many tags match",
			tagsStr:  "setup,config",
			taskTags: []string{"deploy", "setup"},
			expect:   true,
		},
		{
			name:     "no match",
			tagsStr:  "setup,config",
			taskTags: []string{"deploy"},
			expect:   false,
		},
		{
			name:     "always overrides specific filter",
			tagsStr:  "setup",
			taskTags: []string{"always"},
			expect:   true,
		},
		{
			name:     "never overrides everything",
			tagsStr:  "never,setup",
			taskTags: []string{"never"},
			expect:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, _ := New(tt.tagsStr, "")
			result := f.ShouldRun(tt.taskTags)
			if result != tt.expect {
				t.Errorf("ShouldRun(%v) with tags=%q: got %v, want %v", tt.taskTags, tt.tagsStr, result, tt.expect)
			}
		})
	}
}

func TestShouldRun_SkipTags(t *testing.T) {
	tests := []struct {
		name        string
		tagsStr     string
		skipTagsStr string
		taskTags    []string
		expect      bool
	}{
		{
			name:        "skip single tag",
			skipTagsStr: "debug",
			taskTags:    []string{"debug"},
			expect:      false,
		},
		{
			name:        "skip one of many",
			skipTagsStr: "debug,test",
			taskTags:    []string{"setup", "debug"},
			expect:      false,
		},
		{
			name:        "don't skip unrelated",
			skipTagsStr: "debug",
			taskTags:    []string{"setup"},
			expect:      true,
		},
		{
			name:        "skip with specific tags",
			tagsStr:     "setup,config",
			skipTagsStr: "debug",
			taskTags:    []string{"setup"},
			expect:      true,
		},
		{
			name:        "specific tag but also skipped",
			tagsStr:     "setup,debug",
			skipTagsStr: "debug",
			taskTags:    []string{"debug"},
			expect:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, _ := New(tt.tagsStr, tt.skipTagsStr)
			result := f.ShouldRun(tt.taskTags)
			if result != tt.expect {
				t.Errorf("ShouldRun(%v) with tags=%q, skip=%q: got %v, want %v",
					tt.taskTags, tt.tagsStr, tt.skipTagsStr, result, tt.expect)
			}
		})
	}
}

func TestShouldRun_CaseInsensitive(t *testing.T) {
	f, _ := New("Setup,CONFIG", "Debug")

	tests := []struct {
		name     string
		taskTags []string
		expect   bool
	}{
		{
			name:     "lowercase match",
			taskTags: []string{"setup"},
			expect:   true,
		},
		{
			name:     "uppercase match",
			taskTags: []string{"CONFIG"},
			expect:   true,
		},
		{
			name:     "mixed case match",
			taskTags: []string{"SetuP"},
			expect:   true,
		},
		{
			name:     "skip tag lowercase",
			taskTags: []string{"debug"},
			expect:   false,
		},
		{
			name:     "skip tag uppercase",
			taskTags: []string{"DEBUG"},
			expect:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := f.ShouldRun(tt.taskTags)
			if result != tt.expect {
				t.Errorf("ShouldRun(%v): got %v, want %v", tt.taskTags, result, tt.expect)
			}
		})
	}
}

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name        string
		tagsStr     string
		skipTagsStr string
		expect      bool
	}{
		{
			name:   "no filters",
			expect: true,
		},
		{
			name:    "all mode",
			tagsStr: "all",
			expect:  true,
		},
		{
			name:    "tagged mode",
			tagsStr: "tagged",
			expect:  false,
		},
		{
			name:        "skip tags",
			skipTagsStr: "debug",
			expect:      false,
		},
		{
			name:    "specific tags",
			tagsStr: "setup",
			expect:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, _ := New(tt.tagsStr, tt.skipTagsStr)
			result := f.IsEmpty()
			if result != tt.expect {
				t.Errorf("IsEmpty(): got %v, want %v", result, tt.expect)
			}
		})
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		name        string
		tagsStr     string
		skipTagsStr string
		expectStr   string
	}{
		{
			name:      "no filtering",
			expectStr: "no filtering",
		},
		{
			name:      "tagged mode",
			tagsStr:   "tagged",
			expectStr: "tags=tagged",
		},
		{
			name:      "untagged mode",
			tagsStr:   "untagged",
			expectStr: "tags=untagged",
		},
		{
			name:        "skip tags only",
			skipTagsStr: "debug,test",
			expectStr:   "skip_tags=debug,test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, _ := New(tt.tagsStr, tt.skipTagsStr)
			result := f.String()
			if result != tt.expectStr {
				t.Errorf("String(): got %q, want %q", result, tt.expectStr)
			}
		})
	}
}

// Integration tests for complex scenarios
func TestComplexScenarios(t *testing.T) {
	tests := []struct {
		name        string
		tagsStr     string
		skipTagsStr string
		taskTags    []string
		expect      bool
	}{
		{
			name:     "tagged mode with always",
			tagsStr:  "tagged",
			taskTags: []string{"always"},
			expect:   true,
		},
		{
			name:     "untagged mode with always",
			tagsStr:  "untagged",
			taskTags: []string{"always"},
			expect:   true, // 'always' tag always runs
		},
		{
			name:     "specific tags with always",
			tagsStr:  "setup",
			taskTags: []string{"always"},
			expect:   true,
		},
		{
			name:        "skip tags with always",
			skipTagsStr: "debug",
			taskTags:    []string{"always"},
			expect:      true,
		},
		{
			name:        "complex: specific + skip + always",
			tagsStr:     "setup,config",
			skipTagsStr: "debug",
			taskTags:    []string{"always", "setup"},
			expect:      true,
		},
		{
			name:        "complex: specific + skip no match",
			tagsStr:     "setup,config",
			skipTagsStr: "debug",
			taskTags:    []string{"deploy"},
			expect:      false,
		},
		{
			name:        "real world: ci/cd pipeline",
			tagsStr:     "deploy,production",
			skipTagsStr: "debug,experimental",
			taskTags:    []string{"deploy", "security-check"},
			expect:      true,
		},
		{
			name:        "real world: skip debug in tagged",
			tagsStr:     "tagged",
			skipTagsStr: "debug",
			taskTags:    []string{"setup", "debug"},
			expect:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, _ := New(tt.tagsStr, tt.skipTagsStr)
			result := f.ShouldRun(tt.taskTags)
			if result != tt.expect {
				t.Errorf("ShouldRun(%v) with filter.String()=%q: got %v, want %v",
					tt.taskTags, f.String(), result, tt.expect)
			}
		})
	}
}
