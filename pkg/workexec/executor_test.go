package workexec

import "testing"

func TestRunFromEnvironmentRequiresTaskIdentifiers(t *testing.T) {
	t.Setenv("WORKTASK_ID", "")
	t.Setenv("WORKTASK_TYPE", "")

	if err := RunFromEnvironment(); err == nil {
		t.Fatal("expected missing task identifiers to fail")
	}
}
