package factory

import (
	"testing"

	"github.com/free5gc/openapi/models"
)

func TestNrfRegistrationEnabledOrDefault(t *testing.T) {
	disabled := false
	enabled := true

	tests := []struct {
		name   string
		config *Configuration
		want   bool
	}{
		{name: "nil configuration defaults enabled", want: true},
		{name: "omitted defaults enabled", config: &Configuration{}, want: true},
		{name: "explicit enabled", config: &Configuration{NrfRegistrationEnabled: &enabled}, want: true},
		{name: "explicit disabled", config: &Configuration{NrfRegistrationEnabled: &disabled}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.NrfRegistrationEnabledOrDefault(); got != tt.want {
				t.Fatalf("enabled mismatch: got %v want %v", got, tt.want)
			}
		})
	}
}

func TestUPNodeNupfEeApiRootValidation(t *testing.T) {
	tests := []struct {
		name      string
		node      UPNode
		wantRoot  string
		wantError bool
	}{
		{
			name: "omitted",
			node: UPNode{Type: "UPF"},
		},
		{
			name: "normalizes path and trailing slash",
			node: UPNode{
				Type:          "UPF",
				NupfEeApiRoot: stringPtr("https://upf.example.com/api//"),
			},
			wantRoot: "https://upf.example.com/api",
		},
		{
			name: "empty rejected",
			node: UPNode{
				Type:          "UPF",
				NupfEeApiRoot: stringPtr(""),
			},
			wantError: true,
		},
		{
			name: "AN rejected",
			node: UPNode{
				Type:          "AN",
				NupfEeApiRoot: stringPtr("https://upf.example.com"),
			},
			wantError: true,
		},
		{
			name: "query rejected",
			node: UPNode{
				Type:          "UPF",
				NupfEeApiRoot: stringPtr("https://upf.example.com?x=1"),
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.node.validate()
			if tt.wantError {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantRoot != "" {
				if tt.node.NupfEeApiRoot == nil || *tt.node.NupfEeApiRoot != tt.wantRoot {
					t.Fatalf("root mismatch: got %v want %q", tt.node.NupfEeApiRoot, tt.wantRoot)
				}
			}
		})
	}
}

func TestUPNodeTAIValidation(t *testing.T) {
	tai := models.Tai{PlmnId: &models.PlmnId{Mcc: "466", Mnc: "92"}, Tac: "000001"}
	tests := []struct {
		name      string
		node      UPNode
		wantError bool
	}{
		{name: "UPF service area", node: UPNode{Type: "UPF", TAIs: []models.Tai{tai}}},
		{name: "AN rejected", node: UPNode{Type: "AN", TAIs: []models.Tai{tai}}, wantError: true},
		{name: "duplicate rejected", node: UPNode{Type: "UPF", TAIs: []models.Tai{tai, tai}}, wantError: true},
		{name: "incomplete rejected", node: UPNode{Type: "UPF", TAIs: []models.Tai{{Tac: "000001"}}}, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.node.validate()
			if (err != nil) != tt.wantError {
				t.Fatalf("error mismatch: got %v wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestStaticSessionResolutionValidation(t *testing.T) {
	pduSessionID := int32(7)
	valid := StaticSessionMapping{
		Supi:          "imsi-001010000000001",
		UEIPv4:        "192.0.2.1",
		NupfEeApiRoot: "http://127.0.0.8:8088/root/",
		PDUSessionID:  &pduSessionID,
	}
	tests := []struct {
		name      string
		config    StaticSessionResolutionConfig
		wantError bool
	}{
		{name: "disabled and omitted"},
		{
			name: "enabled valid",
			config: StaticSessionResolutionConfig{
				Enabled:  true,
				Sessions: []StaticSessionMapping{valid},
			},
		},
		{
			name:      "enabled missing sessions",
			config:    StaticSessionResolutionConfig{Enabled: true},
			wantError: true,
		},
		{
			name: "duplicate SUPI",
			config: StaticSessionResolutionConfig{
				Enabled:  true,
				Sessions: []StaticSessionMapping{valid, valid},
			},
			wantError: true,
		},
		{
			name: "invalid UE IPv4",
			config: StaticSessionResolutionConfig{
				Enabled: true,
				Sessions: []StaticSessionMapping{{
					Supi: "imsi-1", UEIPv4: "2001:db8::1", NupfEeApiRoot: "http://upf.example",
				}},
			},
			wantError: true,
		},
		{
			name: "invalid API root",
			config: StaticSessionResolutionConfig{
				Enabled: true,
				Sessions: []StaticSessionMapping{{
					Supi: "imsi-1", UEIPv4: "192.0.2.1", NupfEeApiRoot: "upf.example",
				}},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.config.validate()
			if (err != nil) != tt.wantError {
				t.Fatalf("error mismatch: got %v wantError %v", err, tt.wantError)
			}
			if err == nil && len(tt.config.Sessions) == 1 &&
				tt.config.Sessions[0].NupfEeApiRoot != "http://127.0.0.8:8088/root" {
				t.Fatalf("API root was not normalized: %q", tt.config.Sessions[0].NupfEeApiRoot)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
