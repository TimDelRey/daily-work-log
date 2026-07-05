package cli

import (
	"bytes"
	"errors"
	"testing"
)

func TestRunPrintsHelp(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	app := NewApp(&out, &errOut)

	err := app.Run([]string{"--help"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if out.String() != usage {
		t.Fatalf("expected usage in stdout, got %q", out.String())
	}
	if errOut.String() != "" {
		t.Fatalf("expected empty stderr, got %q", errOut.String())
	}
}

func TestRunRequiresCommand(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	app := NewApp(&out, &errOut)

	err := app.Run(nil)
	if !errors.Is(err, ErrUnknownCommand) {
		t.Fatalf("expected ErrUnknownCommand, got %v", err)
	}

	if out.String() != "" {
		t.Fatalf("expected empty stdout, got %q", out.String())
	}
	if errOut.String() != usage {
		t.Fatalf("expected usage in stderr, got %q", errOut.String())
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	app := NewApp(&out, &errOut)

	err := app.Run([]string{"week"})
	if !errors.Is(err, ErrUnknownCommand) {
		t.Fatalf("expected ErrUnknownCommand, got %v", err)
	}

	if out.String() != "" {
		t.Fatalf("expected empty stdout, got %q", out.String())
	}
	if errOut.String() != usage {
		t.Fatalf("expected usage in stderr, got %q", errOut.String())
	}
}

func TestRunCallsReportBuilder(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "today",
			args: []string{commandToday},
			want: "today report\n",
		},
		{
			name: "yesterday",
			args: []string{commandYesterday},
			want: "yesterday report\n",
		},
		{
			name: "status",
			args: []string{commandStatus},
			want: "status report\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			var errOut bytes.Buffer

			app := NewAppWithBuilder(&out, &errOut, stubBuilder{})

			err := app.Run(tt.args)
			if err != nil {
				t.Fatalf("Run returned error: %v", err)
			}

			if out.String() != tt.want {
				t.Fatalf("expected stdout %q, got %q", tt.want, out.String())
			}
			if errOut.String() != "" {
				t.Fatalf("expected empty stderr, got %q", errOut.String())
			}
		})
	}
}

type stubBuilder struct{}

func (stubBuilder) BuildToday() (string, error) {
	return "today report", nil
}

func (stubBuilder) BuildYesterday() (string, error) {
	return "yesterday report", nil
}

func (stubBuilder) BuildStatus() (string, error) {
	return "status report", nil
}
