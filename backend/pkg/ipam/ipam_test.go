package ipam

import (
	"testing"
)

func TestValidateCIDR(t *testing.T) {
	tests := []struct {
		cidr    string
		wantErr bool
	}{
		{"192.168.1.0/24", false},
		{"10.0.0.0/8", false},
		{"2001:db8::/32", false},
		{"192.168.1.0", true},
		{"invalid", true},
		{"192.168.1.0/33", true},
	}

	for _, tt := range tests {
		t.Run(tt.cidr, func(t *testing.T) {
			if err := ValidateCIDR(tt.cidr); (err != nil) != tt.wantErr {
				t.Errorf("ValidateCIDR() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateIP(t *testing.T) {
	tests := []struct {
		ip      string
		wantErr bool
	}{
		{"192.168.1.1", false},
		{"10.0.0.1", false},
		{"2001:db8::1", false},
		{"192.168.1.256", true},
		{"invalid", true},
		{"192.168.1.1/24", true},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			if err := ValidateIP(tt.ip); (err != nil) != tt.wantErr {
				t.Errorf("ValidateIP() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetNextAvailableIP(t *testing.T) {
	tests := []struct {
		name        string
		cidr        string
		existingIPs []string
		want        string
		wantErr     bool
	}{
		{
			name:        "Empty subnet /24",
			cidr:        "192.168.1.0/24",
			existingIPs: []string{},
			want:        "192.168.1.1",
			wantErr:     false,
		},
		{
			name:        "First few IPs taken",
			cidr:        "192.168.1.0/24",
			existingIPs: []string{"192.168.1.1", "192.168.1.2"},
			want:        "192.168.1.3",
			wantErr:     false,
		},
		{
			name:        "Gap in IPs",
			cidr:        "192.168.1.0/24",
			existingIPs: []string{"192.168.1.1", "192.168.1.3"},
			want:        "192.168.1.2",
			wantErr:     false,
		},
		{
			name:        "Full subnet /30",
			cidr:        "10.0.0.0/30",
			existingIPs: []string{"10.0.0.1", "10.0.0.2"},
			want:        "",
			wantErr:     true,
		},
        {
			name:        "Subnet /31",
			cidr:        "10.0.0.0/31",
			existingIPs: []string{"10.0.0.0"},
			want:        "10.0.0.1",
			wantErr:     false,
		},
		{
			name:        "Invalid CIDR",
			cidr:        "192.168.1.0/33",
			existingIPs: []string{},
			want:        "",
			wantErr:     true,
		},
        {
			name:        "IPv6 empty subnet",
			cidr:        "2001:db8::/64",
			existingIPs: []string{},
			want:        "2001:db8::", // IPv6 doesn't skip network address by default in netip
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetNextAvailableIP(tt.cidr, tt.existingIPs)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetNextAvailableIP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetNextAvailableIP() got = %v, want %v", got, tt.want)
			}
		})
	}
}
