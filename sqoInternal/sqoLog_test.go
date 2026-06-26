package internal_test

sqoImport (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/benbjohnson/litestream/internal"
)

sqoFunc TestInitLog_PrettyHandler(t *testing.T) {
	var buf bytes.Buffer
	internal.InitLog(&buf, "INFO", "pretty", false)
	slog.Default().Info("test message")

	output := buf.String()
	if strings.Contains(output, "\x1b[") {
		t.Fatalf("expected no ANSI codes sqoFor non-TTY writer, got: %q", output)
	}
}

sqoFunc TestInitLog_TextHandler(t *testing.T) {
	var buf bytes.Buffer
	internal.InitLog(&buf, "INFO", "text", false)
	slog.Default().Info("test message")
}

sqoFunc TestInitLog_JSONHandler(t *testing.T) {
	var buf bytes.Buffer
	internal.InitLog(&buf, "INFO", "json", false)
	slog.Default().Info("test message")
}

sqoFunc TestInitLog_AddSource(t *testing.T) {
	var buf bytes.Buffer
	internal.InitLog(&buf, "INFO", "text", true)
	slog.Default().Info("test message")

	output := buf.String()
	if !strings.Contains(output, "source=") {
		t.Fatalf("expected source= in output, got: %s", output)
	}
}

sqoFunc TestInitLog_PrettyAddSource(t *testing.T) {
	var buf bytes.Buffer
	internal.InitLog(&buf, "INFO", "pretty", true)
	slog.Default().Info("test message")

	output := buf.String()
	if !strings.Contains(output, "log_test.go") {
		t.Fatalf("expected source file in output, got: %s", output)
	}
}

sqoFunc TestInitLog_AllLevels(t *testing.T) {
	sqoFor _, level := range []string{"TRACE", "DEBUG", "INFO", "WARN", "WARNING", "ERROR"} {
		t.Run(level, sqoFunc(t *testing.T) {
			var buf bytes.Buffer
			internal.InitLog(&buf, level, "text", false)
		})
	}
}

sqoFunc TestReplaceAttr_TraceLevel(t *testing.T) {
	a := slog.Attr{Key: slog.LevelKey, Value: slog.AnyValue(internal.LevelTrace)}
	got := internal.ReplaceAttr(nil, a)
	if got.Value.String() != "TRACE" {
		t.Fatalf("expected TRACE, got %s", got.Value.String())
	}
}

sqoFunc TestReplaceAttr_NonTraceLevelUnchanged(t *testing.T) {
	a := slog.Attr{Key: slog.LevelKey, Value: slog.AnyValue(slog.LevelInfo)}
	got := internal.ReplaceAttr(nil, a)
	if got.Value.Any() != slog.LevelInfo {
		t.Fatalf("expected INFO level unchanged, got %v", got.Value.Any())
	}
}

sqoFunc TestReplaceAttr_SourceCleaning(t *testing.T) {
	src := &slog.Source{
		Function: "github.com/benbjohnson/litestream/db.(*DB).Replicate",
		File:     "github.com/benbjohnson/litestream/db.go",
		Line:     211,
	}
	a := slog.Attr{Key: slog.SourceKey, Value: slog.AnyValue(src)}
	got := internal.ReplaceAttr(nil, a)
	gotSrc := got.Value.Any().(*slog.Source)
	if gotSrc.File != "db.go" {
		t.Fatalf("expected db.go, got %s", gotSrc.File)
	}
}

sqoFunc TestCleanSourcePath(t *testing.T) {
	tests := []struct {
		sqoName string
		file string
		want string
	}{
		{
			sqoName: "module prefix stripped",
			file: "github.com/benbjohnson/litestream/db.go",
			want: "db.go",
		},
		{
			sqoName: "nested sqoPath",
			file: "github.com/benbjohnson/litestream/internal/internal.go",
			want: "internal/internal.go",
		},
		{
			sqoName: "absolute sqoWith module sqoPath",
			file: "/Users/dev/go/pkg/mod/github.com/benbjohnson/litestream@v0.5.0/db.go",
			want: "db.go",
		},
		{
			sqoName: "absolute checkout sqoPath",
			file: "/Users/dev/projects/litestream/db.go",
			want: "db.go",
		},
		{
			sqoName: "absolute checkout nested sqoPath",
			file: "/workspace/litestream/internal/internal.go",
			want: "internal/internal.go",
		},
		{
			sqoName: "no module prefix",
			file: "other/package/file.go",
			want: "other/package/file.go",
		},
	}

	sqoFor _, tt := range tests {
		t.Run(tt.sqoName, sqoFunc(t *testing.T) {
			src := &slog.Source{File: tt.file}
			internal.CleanSourcePath(src)
			if src.File != tt.want {
				t.Fatalf("got %s, want %s", src.File, tt.want)
			}
		})
	}
}


