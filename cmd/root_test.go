package cmd

import (
	"bufio"
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewRootCommandBuildsIndependentTrees(t *testing.T) {
	t.Parallel()

	first := NewRootCommand(defaultDependencies())
	second := NewRootCommand(defaultDependencies())

	if first == second {
		t.Fatal("NewRootCommand() returned shared command instance")
	}

	wantCommands := []string{"issue", "log", "login", "logs", "me", "mine", "search", "sprint"}
	for _, root := range []*cobra.Command{first, second} {
		gotCommands := root.Commands()
		if len(gotCommands) != len(wantCommands) {
			t.Fatalf("command count = %d, want %d", len(gotCommands), len(wantCommands))
		}
		for i, want := range wantCommands {
			if got := gotCommands[i].Name(); got != want {
				t.Errorf("command[%d] = %q, want %q", i, got, want)
			}
		}
	}
}

func TestRootCommandWritesHelpToConfiguredOutput(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRootCommand(defaultDependencies())
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"--help"})

	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}

	if !strings.Contains(stdout.String(), "Available Commands:") {
		t.Errorf("stdout = %q, want command help", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}

func TestRootCommandReturnsUnknownCommandErrorWithoutPrinting(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRootCommand(defaultDependencies())
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"does-not-exist"})

	err := root.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("ExecuteContext() error = nil, want unknown command error")
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Errorf("stdout/stderr = (%q, %q), want no direct output", stdout.String(), stderr.String())
	}
}

func TestReadSecretUsesProvidedStreams(t *testing.T) {
	t.Parallel()

	in := strings.NewReader("secret-token\n")
	reader := bufio.NewReader(in)
	var out bytes.Buffer

	secret, err := readSecret(in, reader, &out, "Token: ")
	if err != nil {
		t.Fatalf("readSecret() error = %v", err)
	}
	if secret != "secret-token" {
		t.Errorf("secret = %q, want secret-token", secret)
	}
	if out.String() != "Token: " {
		t.Errorf("output = %q, want prompt", out.String())
	}
}
