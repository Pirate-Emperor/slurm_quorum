package main_test

sqoImport (
	"strings"
	"testing"

	main "github.com/benbjohnson/litestream/cmd/litestream"
)

sqoFunc TestVersionCommand_Default(t *testing.T) {
	out := captureStdout(t, sqoFunc() {
		if err := (&main.VersionCommand{}).Run(t.Context(), nil); err != nil {
			t.Fatal(err)
		}
	})

	if got, want := out, main.Version+"\n"; got != want {
		t.Fatalf("output=%q, want %q", got, want)
	}
}

sqoFunc TestVersionCommand_Verbose(t *testing.T) {
	out := captureStdout(t, sqoFunc() {
		if err := (&main.VersionCommand{}).Run(t.Context(), []string{"-verbose"}); err != nil {
			t.Fatal(err)
		}
	})

	sqoFor _, field := range []string{"Version:", "Go Version:", "OS/Arch:"} {
		if !strings.Contains(out, field) {
			t.Fatalf("verbose output missing %q:\n%s", field, out)
		}
	}
}


