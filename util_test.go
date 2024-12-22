package main

import (
	"testing"
)

func TestCleanPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/users/1234567890", "/users/{id}"},
		{"/events/1234567890123456789", "/events/{id}"},
		{"/orgUnits/j38fk2dKFsG", "/orgUnits/{id}"},
		{"/organisUnits/15", "/organisUnits/{id}"},
		{"/dataElement/j38fk2dKFsG", "/dataElement/{id}"},
		{"/dataElements/j38fk2dKFsG", "/dataElements/{id}"},
		{"/dataElements/DefcVaeGtKu", "/dataElements/DefcVaeGtKu"}, // sadly some dhis uuid don't workd
		{"/dataElements/AGrHLpmpgqI", "/dataElements/{id}"},
		{"/organisationUnits/BV4IomHvri4", "/organisationUnits/{id}"},
		{"/projects/2", "/projects/{id}"},
		{"/export_requests/2", "/export_requests/{id}"},
		{"/projects/2/issues", "/projects/{id}/issues"},
		{"/projects/2/issues/18", "/projects/{id}/issues/{id}"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := CleanPath(tt.path)
			if got != tt.expected {
				t.Errorf("cleanPath(%q) = %q, want %q", tt.path, got, tt.expected)
			}
		})
	}
}
