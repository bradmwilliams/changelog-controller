package changelog_controller

import (
	"testing"
)

func TestComputeRpmDiff(t *testing.T) {
	tests := []struct {
		name        string
		from        map[string]string
		to          map[string]string
		wantNil     bool
		wantChanged int
		wantAdded   int
		wantRemoved int
	}{
		{
			name:    "both nil",
			from:    nil,
			to:      nil,
			wantNil: true,
		},
		{
			name:    "both empty",
			from:    map[string]string{},
			to:      map[string]string{},
			wantNil: true,
		},
		{
			name:    "identical packages",
			from:    map[string]string{"kernel": "5.14.0-284.el9", "systemd": "252-14.el9"},
			to:      map[string]string{"kernel": "5.14.0-284.el9", "systemd": "252-14.el9"},
			wantNil: true,
		},
		{
			name:        "package added",
			from:        map[string]string{"kernel": "5.14.0-284.el9"},
			to:          map[string]string{"kernel": "5.14.0-284.el9", "cri-o": "1.30.0-1.el9"},
			wantAdded:   1,
			wantChanged: 0,
			wantRemoved: 0,
		},
		{
			name:        "package removed",
			from:        map[string]string{"kernel": "5.14.0-284.el9", "old-pkg": "1.0"},
			to:          map[string]string{"kernel": "5.14.0-284.el9"},
			wantAdded:   0,
			wantChanged: 0,
			wantRemoved: 1,
		},
		{
			name:        "package version changed",
			from:        map[string]string{"kernel": "5.14.0-284.el9"},
			to:          map[string]string{"kernel": "5.14.0-427.el9"},
			wantAdded:   0,
			wantChanged: 1,
			wantRemoved: 0,
		},
		{
			name:        "mixed changes",
			from:        map[string]string{"kernel": "5.14.0-284.el9", "old-pkg": "1.0", "systemd": "252-14.el9"},
			to:          map[string]string{"kernel": "5.14.0-427.el9", "new-pkg": "2.0", "systemd": "252-14.el9"},
			wantAdded:   1,
			wantChanged: 1,
			wantRemoved: 1,
		},
		{
			name:        "from empty, all added",
			from:        map[string]string{},
			to:          map[string]string{"kernel": "5.14.0-284.el9", "systemd": "252-14.el9"},
			wantAdded:   2,
			wantChanged: 0,
			wantRemoved: 0,
		},
		{
			name:        "to empty, all removed",
			from:        map[string]string{"kernel": "5.14.0-284.el9", "systemd": "252-14.el9"},
			to:          map[string]string{},
			wantAdded:   0,
			wantChanged: 0,
			wantRemoved: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeRpmDiff(tt.from, tt.to)

			if tt.wantNil {
				if got != nil {
					t.Errorf("ComputeRpmDiff() = %+v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatal("ComputeRpmDiff() = nil, want non-nil")
			}
			if len(got.Changed) != tt.wantChanged {
				t.Errorf("Changed has %d entries, want %d", len(got.Changed), tt.wantChanged)
			}
			if len(got.Added) != tt.wantAdded {
				t.Errorf("Added has %d entries, want %d", len(got.Added), tt.wantAdded)
			}
			if len(got.Removed) != tt.wantRemoved {
				t.Errorf("Removed has %d entries, want %d", len(got.Removed), tt.wantRemoved)
			}
		})
	}
}

func TestComputeRpmDiff_Values(t *testing.T) {
	from := map[string]string{
		"kernel":  "5.14.0-284.el9",
		"old-pkg": "1.0.0-1.el9",
	}
	to := map[string]string{
		"kernel":  "5.14.0-427.el9",
		"new-pkg": "2.0.0-1.el9",
	}

	got := ComputeRpmDiff(from, to)
	if got == nil {
		t.Fatal("ComputeRpmDiff() = nil")
	}

	if c, ok := got.Changed["kernel"]; !ok {
		t.Error("kernel not in Changed")
	} else {
		if c.Old != "5.14.0-284.el9" {
			t.Errorf("kernel Old = %q, want %q", c.Old, "5.14.0-284.el9")
		}
		if c.New != "5.14.0-427.el9" {
			t.Errorf("kernel New = %q, want %q", c.New, "5.14.0-427.el9")
		}
	}

	if ver, ok := got.Added["new-pkg"]; !ok {
		t.Error("new-pkg not in Added")
	} else if ver != "2.0.0-1.el9" {
		t.Errorf("new-pkg version = %q, want %q", ver, "2.0.0-1.el9")
	}

	if ver, ok := got.Removed["old-pkg"]; !ok {
		t.Error("old-pkg not in Removed")
	} else if ver != "1.0.0-1.el9" {
		t.Errorf("old-pkg version = %q, want %q", ver, "1.0.0-1.el9")
	}
}
