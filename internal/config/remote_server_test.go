package config

import "testing"

func TestRemoteServer(t *testing.T) {
	for in, want := range map[string]string{"": "auto", "auto": "auto", "python": "python", "sh": "sh"} {
		c := &Config{MaxConcurrency: 1, RemoteServer: in}
		if err := c.Validate(); err != nil || c.GetRemoteServer() != want {
			t.Errorf("%q: %q %v", in, c.GetRemoteServer(), err)
		}
	}
	if err := (&Config{RemoteServer: "agent"}).Validate(); err == nil {
		t.Error("an unknown value is an error")
	}
}
