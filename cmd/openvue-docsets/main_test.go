package main

import (
	"bytes"
	"testing"
)

func TestCommands(t *testing.T) {
	for _, args := range [][]string{nil, {"help"}, {"--help"}, {"-h"}, {"version"}, {"--version"}} {
		t.Run(stringArgs(args), func(t *testing.T) {
			cmd := newCommand()
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			if out.Len() == 0 {
				t.Fatal("empty output")
			}
			if len(args) == 1 && (args[0] == "version" || args[0] == "--version") && out.String() != version+"\n" {
				t.Fatal(out.String())
			}
		})
	}
	for _, args := range [][]string{{"unknown"}, {"version", "extra"}, {"build", "extra"}, {"--unknown"}} {
		cmd := newCommand()
		cmd.SetArgs(args)
		if err := cmd.Execute(); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func stringArgs(args []string) string {
	if len(args) == 0 {
		return "default"
	}
	return args[0]
}
