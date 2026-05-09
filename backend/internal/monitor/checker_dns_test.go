package monitor

import (
	"context"
	"testing"

	"blackgrid/internal/db"
)

func TestDNSCheckerMatchValues(t *testing.T) {
	tests := []struct {
		name     string
		returned []string
		expected []string
		mode     string
		want     bool
	}{
		{"any match", []string{"10.0.0.1", "10.0.0.2"}, []string{"10.0.0.1"}, "any", true},
		{"any no match", []string{"10.0.0.1", "10.0.0.2"}, []string{"10.0.0.3"}, "any", false},
		{"all match", []string{"10.0.0.1", "10.0.0.2"}, []string{"10.0.0.1", "10.0.0.2"}, "all", true},
		{"all missing", []string{"10.0.0.1"}, []string{"10.0.0.1", "10.0.0.2"}, "all", false},
		{"exact match", []string{"10.0.0.1", "10.0.0.2"}, []string{"10.0.0.2", "10.0.0.1"}, "exact", true},
		{"exact extra", []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"}, []string{"10.0.0.1", "10.0.0.2"}, "exact", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := matchValues(tt.returned, tt.expected, tt.mode)
			if got != tt.want {
				t.Errorf("matchValues() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDNSChecker_Check(t *testing.T) {
	c := &DNSChecker{}

	t.Run("invalid config", func(t *testing.T) {
		m := db.Monitor{Config: []byte("{invalid}")}
		res := c.Check(context.Background(), m)
		if res.Status != "down" {
			t.Errorf("expected status down for invalid config, got %s", res.Status)
		}
	})

	t.Run("no hostname", func(t *testing.T) {
		m := db.Monitor{Target: "", Config: []byte("{}")}
		res := c.Check(context.Background(), m)
		if res.Status != "down" || res.ErrorMessage != "no hostname configured" {
			t.Errorf("expected no hostname error, got %s: %s", res.Status, res.ErrorMessage)
		}
	})
}
