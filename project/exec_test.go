package project

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestBashExecutor(t *testing.T) {

	dir := testDir()
	ctx := context.Background()
	e := NewBashExecutor()

	var stdout bytes.Buffer

	err := e.Execute(ctx, ExecOpts{
		Command:          "echo HI $PWD",
		WorkingDirectory: dir,
		Stdout:           &stdout,
	})
	require.Nil(t, err)

	expected := fmt.Sprintf("HI %s", dir)

	// Somehow the tmpdir is prefixed with /private on macos when using PWD
	expectedAlt := fmt.Sprintf("HI /private%s", dir)

	out := strings.TrimSpace(stdout.String())

	if out != expected && out != expectedAlt {
		t.Error("Unexpected output:", out)
	}
}
