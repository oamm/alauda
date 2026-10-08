package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestTokenNeverAppearsInHelpOrUsage(t *testing.T) {
	old := cliConfig
	defer func() { cliConfig = old }()
	const secret = "sentinel-do-not-disclose"
	t.Setenv("ALAUDA_TOKEN", secret)
	t.Setenv("REGISTRY_TOKEN", secret)
	for _, args := range [][]string{{"--help"}, {"services", "register", "--help"}, {"services", "register"}, {"services", "list", "--server", "http://127.0.0.1:1"}} {
		cliConfig = defaultCLIConfig()
		cmd := newRootCommand()
		var output bytes.Buffer
		cmd.SetOut(&output)
		cmd.SetErr(&output)
		cmd.SetArgs(args)
		err := cmd.Execute()
		if err != nil {
			output.WriteString(err.Error())
		}
		if strings.Contains(output.String(), secret) {
			t.Fatalf("credential leaked for %v", args)
		}
		if cmd.PersistentFlags().Lookup("token").DefValue != "" {
			t.Fatal("token has printable default")
		}
	}
}
