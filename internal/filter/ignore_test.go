package filter

import "testing"

func TestDefaultMatcher(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "app/models/person.rb", want: true},
		{path: "sig/person.rbs", want: true},
		{path: "person.rbi", want: false},
		{path: "app/models/person.rbi", want: false},
		{path: "sorbet/rbi/generated.rbi", want: false},
		{path: "sorbet/rbi/gems/dependency.rbi", want: false},
		{path: "nested/sorbet/rbi/file.rb", want: true},
	}

	matcher := Default()
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := matcher.Include(tt.path); got != tt.want {
				t.Fatalf("Include(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
