package app

import (
	"strings"
	"testing"
)

func TestWantsVersion(t *testing.T) {
	for _, a := range []string{"--version", "-version", "-V"} {
		if !wantsVersion([]string{a}) {
			t.Fatalf("%s is not recognised", a)
		}
	}
	for _, args := range [][]string{nil, {"--lang", "cn"}, {"--lang=cn"}} {
		if wantsVersion(args) {
			t.Fatalf("%q asks for the version", args)
		}
	}
	// The version is never empty, and names the command either way.
	if v := version(); !strings.HasPrefix(v, "otty-snippets-view ") || len(v) <= len("otty-snippets-view ") {
		t.Fatalf("version = %q", v)
	}
}
