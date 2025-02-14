package hash

import (
	"testing"
)

func TestFNV(t *testing.T) {
	tests := []struct {
		ip       string
		expected uint32
	}{
		{"192.168.1.1", FNV("192.168.1.1")},
		{"10.0.0.1", FNV("10.0.0.1")},
		{"127.0.0.1", FNV("127.0.0.1")},
		{"8.8.8.8", FNV("8.8.8.8")},
		{"255.255.255.255", FNV("255.255.255.255")},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			hash := FNV(tt.ip)
			if hash != tt.expected {
				t.Errorf("FNV(%s) = %v; want %v", tt.ip, hash, tt.expected)
			}
		})
	}
}
