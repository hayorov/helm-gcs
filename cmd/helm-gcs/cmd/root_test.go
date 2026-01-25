// Copyright © 2024 helm-gcs contributors
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"testing"
)

func TestRootCommandStructure(t *testing.T) {
	if rootCmd == nil {
		t.Fatal("rootCmd should not be nil")
	}

	if rootCmd.Use != "helm-gcs" {
		t.Errorf("rootCmd.Use = %q, want %q", rootCmd.Use, "helm-gcs")
	}

	if rootCmd.Short != "Manage Helm repositories on Google Cloud Storage" {
		t.Errorf("rootCmd.Short = %q, want %q", rootCmd.Short, "Manage Helm repositories on Google Cloud Storage")
	}
}

func TestSubcommandsExist(t *testing.T) {
	expectedCommands := []string{"init", "push", "pull", "rm", "version"}

	commands := rootCmd.Commands()
	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		commandNames[cmd.Name()] = true
	}

	for _, expected := range expectedCommands {
		if !commandNames[expected] {
			t.Errorf("expected subcommand %q not found", expected)
		}
	}
}

func TestRootFlagRegistration(t *testing.T) {
	tests := []struct {
		flagName    string
		description string
	}{
		{"service-account", "service account flag should be registered"},
		{"debug", "debug flag should be registered"},
	}

	for _, tt := range tests {
		t.Run(tt.flagName, func(t *testing.T) {
			flag := rootCmd.PersistentFlags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("%s: flag %q not found", tt.description, tt.flagName)
			}
		})
	}
}

func TestServiceAccountFlagDefaults(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("service-account")
	if flag == nil {
		t.Fatal("service-account flag not found")
	}

	if flag.DefValue != "" {
		t.Errorf("service-account default value = %q, want empty string", flag.DefValue)
	}
}

func TestDebugFlagDefaults(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("debug")
	if flag == nil {
		t.Fatal("debug flag not found")
	}

	if flag.DefValue != "false" {
		t.Errorf("debug default value = %q, want %q", flag.DefValue, "false")
	}
}

func TestInitCommandStructure(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"init"})
	if err != nil {
		t.Fatalf("failed to find init command: %v", err)
	}

	if cmd.Use != "init gs://bucket/path" {
		t.Errorf("init command Use = %q, want %q", cmd.Use, "init gs://bucket/path")
	}

	if cmd.Short != "init a repository" {
		t.Errorf("init command Short = %q, want %q", cmd.Short, "init a repository")
	}
}

func TestPushCommandStructure(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"push"})
	if err != nil {
		t.Fatalf("failed to find push command: %v", err)
	}

	if cmd.Use != "push [chart.tar.gz] [repository]" {
		t.Errorf("push command Use = %q, want %q", cmd.Use, "push [chart.tar.gz] [repository]")
	}
}

func TestPushCommandFlags(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"push"})
	if err != nil {
		t.Fatalf("failed to find push command: %v", err)
	}

	expectedFlags := []struct {
		name     string
		defValue string
	}{
		{"force", "false"},
		{"retry", "false"},
		{"public", "false"},
		{"publicUrl", ""},
		{"bucketPath", ""},
	}

	for _, expected := range expectedFlags {
		flag := cmd.Flags().Lookup(expected.name)
		if flag == nil {
			t.Errorf("push command: flag %q not found", expected.name)
			continue
		}
		if flag.DefValue != expected.defValue {
			t.Errorf("push command: flag %q default = %q, want %q", expected.name, flag.DefValue, expected.defValue)
		}
	}
}

func TestPullCommandStructure(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"pull"})
	if err != nil {
		t.Fatalf("failed to find pull command: %v", err)
	}

	if cmd.Use != "pull gs://bucket/path" {
		t.Errorf("pull command Use = %q, want %q", cmd.Use, "pull gs://bucket/path")
	}

	if cmd.Short != "prints a file on stdout" {
		t.Errorf("pull command Short = %q, want %q", cmd.Short, "prints a file on stdout")
	}
}

func TestRmCommandStructure(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"rm"})
	if err != nil {
		t.Fatalf("failed to find rm command: %v", err)
	}

	if cmd.Use != "rm [chart] [repository]" {
		t.Errorf("rm command Use = %q, want %q", cmd.Use, "rm [chart] [repository]")
	}
}

func TestRmCommandAliases(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"rm"})
	if err != nil {
		t.Fatalf("failed to find rm command: %v", err)
	}

	hasRemoveAlias := false
	for _, alias := range cmd.Aliases {
		if alias == "remove" {
			hasRemoveAlias = true
			break
		}
	}

	if !hasRemoveAlias {
		t.Error("rm command should have 'remove' alias")
	}
}

func TestRmCommandFlags(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"rm"})
	if err != nil {
		t.Fatalf("failed to find rm command: %v", err)
	}

	versionFlag := cmd.Flags().Lookup("version")
	if versionFlag == nil {
		t.Error("rm command: version flag not found")
	} else {
		if versionFlag.Shorthand != "v" {
			t.Errorf("rm command: version flag shorthand = %q, want %q", versionFlag.Shorthand, "v")
		}
	}

	retryFlag := cmd.Flags().Lookup("retry")
	if retryFlag == nil {
		t.Error("rm command: retry flag not found")
	}
}

func TestVersionCommandStructure(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"version"})
	if err != nil {
		t.Fatalf("failed to find version command: %v", err)
	}

	if cmd.Use != "version" {
		t.Errorf("version command Use = %q, want %q", cmd.Use, "version")
	}

	if cmd.Short != "print current helm-gcs version" {
		t.Errorf("version command Short = %q, want %q", cmd.Short, "print current helm-gcs version")
	}
}

func TestServiceAccountFunction(t *testing.T) {
	originalValue := flagServiceAccount
	defer func() { flagServiceAccount = originalValue }()

	flagServiceAccount = ""
	if ServiceAccount() != "" {
		t.Errorf("ServiceAccount() = %q, want empty string", ServiceAccount())
	}

	flagServiceAccount = "/path/to/sa.json"
	if ServiceAccount() != "/path/to/sa.json" {
		t.Errorf("ServiceAccount() = %q, want %q", ServiceAccount(), "/path/to/sa.json")
	}
}

func TestCommandCount(t *testing.T) {
	commands := rootCmd.Commands()
	expectedMinimum := 5

	if len(commands) < expectedMinimum {
		t.Errorf("expected at least %d subcommands, got %d", expectedMinimum, len(commands))
	}
}

func TestRootCommandHasPersistentPreRunE(t *testing.T) {
	if rootCmd.PersistentPreRunE == nil {
		t.Error("rootCmd should have PersistentPreRunE set")
	}
}

func TestMetadataFlagOnPushCommand(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"push"})
	if err != nil {
		t.Fatalf("failed to find push command: %v", err)
	}

	flag := cmd.Flags().Lookup("metadata")
	if flag == nil {
		t.Error("push command: metadata flag not found")
	}
}

func TestInitCommandHasArgs(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"init"})
	if err != nil {
		t.Fatalf("failed to find init command: %v", err)
	}

	if cmd.Args == nil {
		t.Error("init command should have Args validator set")
	}
}

func TestPushCommandHasArgs(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"push"})
	if err != nil {
		t.Fatalf("failed to find push command: %v", err)
	}

	if cmd.Args == nil {
		t.Error("push command should have Args validator set")
	}
}

func TestRmCommandHasArgs(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"rm"})
	if err != nil {
		t.Fatalf("failed to find rm command: %v", err)
	}

	if cmd.Args == nil {
		t.Error("rm command should have Args validator set")
	}
}

func TestFindCommandByAlias(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"remove"})
	if err != nil {
		t.Fatalf("failed to find command by alias 'remove': %v", err)
	}

	if cmd.Name() != "rm" {
		t.Errorf("expected to find 'rm' command via 'remove' alias, got %q", cmd.Name())
	}
}

func TestAllCommandsHaveRunE(t *testing.T) {
	commandsToCheck := []string{"init", "push", "pull", "rm"}

	for _, cmdName := range commandsToCheck {
		cmd, _, err := rootCmd.Find([]string{cmdName})
		if err != nil {
			t.Errorf("failed to find %s command: %v", cmdName, err)
			continue
		}

		if cmd.RunE == nil {
			t.Errorf("%s command should have RunE set", cmdName)
		}
	}
}

func TestVersionCommandHasRun(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"version"})
	if err != nil {
		t.Fatalf("failed to find version command: %v", err)
	}

	if cmd.Run == nil {
		t.Error("version command should have Run set")
	}
}

func TestDebugFlagType(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("debug")
	if flag == nil {
		t.Fatal("debug flag not found")
	}

	if flag.Value.Type() != "bool" {
		t.Errorf("debug flag type = %q, want %q", flag.Value.Type(), "bool")
	}
}

func TestServiceAccountFlagType(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("service-account")
	if flag == nil {
		t.Fatal("service-account flag not found")
	}

	if flag.Value.Type() != "string" {
		t.Errorf("service-account flag type = %q, want %q", flag.Value.Type(), "string")
	}
}

func TestVersionCommandExecution(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"version"})
	if err != nil {
		t.Fatalf("failed to find version command: %v", err)
	}

	originalPreRunE := rootCmd.PersistentPreRunE

	err = originalPreRunE(cmd, []string{})
	if err != nil {
		t.Errorf("PersistentPreRunE failed for version command: %v", err)
	}
}
