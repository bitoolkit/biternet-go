package config

import "testing"

func TestNixStartup(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value string
		want  *Startup
		fail  bool
	}{
		{name: "application managed"},
		{name: "disabled by Nix", value: `{"auto_start":false,"silent_start":false}`, want: &Startup{}},
		{name: "visible at login", value: `{"auto_start":true,"silent_start":false}`, want: &Startup{AutoStart: true}},
		{name: "hidden at login", value: `{"auto_start":true,"silent_start":true}`, want: &Startup{AutoStart: true, SilentStart: true}},
		{name: "invalid configuration", value: `broken`, fail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("BITERNET_NIX_STARTUP", tc.value)
			got, err := NixStartup()
			if (err != nil) != tc.fail {
				t.Fatalf("unexpected error: %v", err)
			}
			if (got == nil) != (tc.want == nil) || (got != nil && *got != *tc.want) {
				t.Fatalf("startup = %+v, want %+v", got, tc.want)
			}
		})
	}
}
