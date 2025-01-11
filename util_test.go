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
		{"/dataElements/DefcVaeGtKu", "/dataElements/{id}"}, // sadly some dhis uuid don't workd
		{"/dataElements/AGrHLpmpgqI", "/dataElements/{id}"},
		{"/submission/max-size/A0ST0qwp", "/submission/max-size/{id}"},
		{"/single/9ahTsRs2", "/single/{id}"},
		{"/single/BahTsRsD", "/single/{id}"},
		{"/organisationUnits/BV4IomHvri4", "/organisationUnits/{id}"},
		{"/projects/2", "/projects/{id}"},
		{"/export_requests/2", "/export_requests/{id}"},
		{"/projects/2/issues", "/projects/{id}/issues"},
		{"/projects/2/issues/18", "/projects/{id}/issues/{id}"},
		{"/transform/xform/OyanEbca", "/transform/xform/OyanEbca"},
		{"/transform/xform/xXeYccbp", "/transform/xform/xXeYccbp"},
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
