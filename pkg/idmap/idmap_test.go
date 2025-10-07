package idmap_test

import (
	"encoding/hex"
	"errors"
	"testing"

	"github.com/ngharo/sss_idmap_ad2unix/pkg/idmap"
)

func TestNewIDMapContext(t *testing.T) {
	ctx, err := idmap.NewIDMapContext()
	if err != nil {
		t.Fatalf("NewIDMapContext() failed: %v", err)
	}
	defer func() {
		if err := ctx.Close(); err != nil {
			t.Errorf("Close() failed: %v", err)
		}
	}()

	if ctx == nil {
		t.Fatal("NewIDMapContext() returned nil context")
	}
}

func TestNewIDMapContextWithDomain(t *testing.T) {
	config := idmap.DomainConfig{
		DomainName: "EXAMPLE",
		DomainSID:  "S-1-5-21-3623811015-3361044348-30300820",
		IDRange: idmap.IDRange{
			Min: 10000,
			Max: 20000,
		},
	}

	ctx, err := idmap.NewIDMapContextWithDomain(config)
	if err != nil {
		t.Fatalf("NewIDMapContextWithDomain() failed: %v", err)
	}
	defer ctx.Close()

	if ctx == nil {
		t.Fatal("NewIDMapContextWithDomain() returned nil context")
	}
}

func TestAddDomain(t *testing.T) {
	ctx, err := idmap.NewIDMapContext()
	if err != nil {
		t.Fatalf("NewIDMapContext() failed: %v", err)
	}
	defer ctx.Close()

	config := idmap.DomainConfig{
		DomainName: "TESTDOMAIN",
		DomainSID:  "S-1-5-21-1234567890-1234567890-1234567890",
		IDRange: idmap.IDRange{
			Min: 10000,
			Max: 20000,
		},
	}

	err = ctx.AddDomain(config)
	if err != nil {
		t.Errorf("AddDomain() failed: %v", err)
	}
}

func TestAddDomain_InvalidRange(t *testing.T) {
	ctx, err := idmap.NewIDMapContext()
	if err != nil {
		t.Fatalf("NewIDMapContext() failed: %v", err)
	}
	defer ctx.Close()

	tests := []struct {
		name   string
		config idmap.DomainConfig
	}{
		{
			name: "min equals max",
			config: idmap.DomainConfig{
				DomainName: "INVALID1",
				DomainSID:  "S-1-5-21-1111111111-2222222222-3333333333",
				IDRange:    idmap.IDRange{Min: 10000, Max: 10000},
			},
		},
		{
			name: "min greater than max",
			config: idmap.DomainConfig{
				DomainName: "INVALID2",
				DomainSID:  "S-1-5-21-1111111111-2222222222-3333333333",
				IDRange:    idmap.IDRange{Min: 20000, Max: 10000},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ctx.AddDomain(tt.config)
			if err == nil {
				t.Error("AddDomain() expected error for invalid range, got nil")
			}
			if !errors.Is(err, idmap.ErrInvalidRange) {
				t.Errorf("AddDomain() expected ErrInvalidRange, got: %v", err)
			}
		})
	}
}

func TestSIDToUnixID_WithDomain(t *testing.T) {
	config := idmap.DomainConfig{
		DomainName: "EXAMPLE",
		DomainSID:  "S-1-5-21-3623811015-3361044348-30300820",
		IDRange: idmap.IDRange{
			Min: 10000,
			Max: 20000,
		},
	}

	ctx, err := idmap.NewIDMapContextWithDomain(config)
	if err != nil {
		t.Fatalf("NewIDMapContextWithDomain() failed: %v", err)
	}
	defer ctx.Close()

	// Test converting a SID from the configured domain
	sid := "S-1-5-21-3623811015-3361044348-30300820-1013"
	unixID, err := ctx.SIDToUnixID(sid)
	if err != nil {
		t.Fatalf("SIDToUnixID(%q) failed: %v", sid, err)
	}

	// Verify the ID is within the configured range
	if unixID < config.IDRange.Min || unixID > config.IDRange.Max {
		t.Errorf("SIDToUnixID(%q) = %d, want ID in range [%d, %d]",
			sid, unixID, config.IDRange.Min, config.IDRange.Max)
	}

	t.Logf("SID %s mapped to Unix ID %d", sid, unixID)
}

func TestIDMapContext_Close(t *testing.T) {
	ctx, err := idmap.NewIDMapContext()
	if err != nil {
		t.Fatalf("NewIDMapContext() failed: %v", err)
	}

	err = ctx.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// Closing twice should not cause issues
	err = ctx.Close()
	if err != nil {
		t.Errorf("Close() called twice failed: %v", err)
	}
}

func TestIDMapContext_SIDToUnixID_InvalidSID(t *testing.T) {
	tests := []struct {
		name string
		sid  string
	}{
		{
			name: "empty SID",
			sid:  "",
		},
		{
			name: "invalid format",
			sid:  "not-a-sid",
		},
		{
			name: "partial SID",
			sid:  "S-1-5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := idmap.NewIDMapContext()
			if err != nil {
				t.Fatalf("NewIDMapContext() failed: %v", err)
			}
			defer ctx.Close()

			_, err = ctx.SIDToUnixID(tt.sid)
			if err == nil {
				t.Errorf("SIDToUnixID(%q) expected error, got nil", tt.sid)
			}

			if !errors.Is(err, idmap.ErrInvalidSID) && !errors.Is(err, idmap.ErrNotFound) && !errors.Is(err, idmap.ErrInternal) {
				t.Errorf("SIDToUnixID(%q) expected known error type, got: %v", tt.sid, err)
			}
		})
	}
}

func TestSIDToUnixID(t *testing.T) {
	// Deterministic offline tests with known SID to UID/GID mappings
	// These test cases verify that the same SID always maps to the same Unix ID

	tests := []struct {
		name       string
		config     idmap.DomainConfig
		sid        string
		wantUnixID uint32
	}{
		{
			name: "EXAMPLE domain user 1013",
			config: idmap.DomainConfig{
				DomainName: "EXAMPLE",
				DomainSID:  "S-1-5-21-3623811015-3361044348-30300820",
				IDRange:    idmap.IDRange{Min: 10000, Max: 20000},
			},
			sid:        "S-1-5-21-3623811015-3361044348-30300820-1013",
			wantUnixID: 11013,
		},
		{
			name: "EXAMPLE domain user 500",
			config: idmap.DomainConfig{
				DomainName: "EXAMPLE",
				DomainSID:  "S-1-5-21-3623811015-3361044348-30300820",
				IDRange:    idmap.IDRange{Min: 10000, Max: 20000},
			},
			sid:        "S-1-5-21-3623811015-3361044348-30300820-500",
			wantUnixID: 10500,
		},
		{
			name: "EXAMPLE domain group 513 (Domain Users)",
			config: idmap.DomainConfig{
				DomainName: "EXAMPLE",
				DomainSID:  "S-1-5-21-3623811015-3361044348-30300820",
				IDRange:    idmap.IDRange{Min: 10000, Max: 20000},
			},
			sid:        "S-1-5-21-3623811015-3361044348-30300820-513",
			wantUnixID: 10513,
		},
		{
			name: "TESTDOMAIN with different range",
			config: idmap.DomainConfig{
				DomainName: "TESTDOMAIN",
				DomainSID:  "S-1-5-21-1234567890-1234567890-1234567890",
				IDRange:    idmap.IDRange{Min: 20000, Max: 30000},
			},
			sid:        "S-1-5-21-1234567890-1234567890-1234567890-1001",
			wantUnixID: 21001,
		},
		{
			name: "TESTDOMAIN with high RID",
			config: idmap.DomainConfig{
				DomainName: "TESTDOMAIN",
				DomainSID:  "S-1-5-21-1234567890-1234567890-1234567890",
				IDRange:    idmap.IDRange{Min: 20000, Max: 30000},
			},
			sid:        "S-1-5-21-1234567890-1234567890-1234567890-5000",
			wantUnixID: 25000,
		},
		{
			name: "CONTOSO domain administrator",
			config: idmap.DomainConfig{
				DomainName: "CONTOSO",
				DomainSID:  "S-1-5-21-1111111111-2222222222-3333333333",
				IDRange:    idmap.IDRange{Min: 100000, Max: 200000},
			},
			sid:        "S-1-5-21-1111111111-2222222222-3333333333-500",
			wantUnixID: 100500,
		},
		{
			name: "CONTOSO domain guest",
			config: idmap.DomainConfig{
				DomainName: "CONTOSO",
				DomainSID:  "S-1-5-21-1111111111-2222222222-3333333333",
				IDRange:    idmap.IDRange{Min: 100000, Max: 200000},
			},
			sid:        "S-1-5-21-1111111111-2222222222-3333333333-501",
			wantUnixID: 100501,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := idmap.NewIDMapContextWithDomain(tt.config)
			if err != nil {
				t.Fatalf("NewIDMapContextWithDomain() failed: %v", err)
			}
			defer ctx.Close()

			gotUnixID, err := ctx.SIDToUnixID(tt.sid)
			if err != nil {
				t.Fatalf("SIDToUnixID(%q) failed: %v", tt.sid, err)
			}

			if gotUnixID != tt.wantUnixID {
				t.Errorf("SIDToUnixID(%q) = %d, want %d", tt.sid, gotUnixID, tt.wantUnixID)
			}

			// Verify the mapping is deterministic by converting the same SID again
			gotUnixID2, err := ctx.SIDToUnixID(tt.sid)
			if err != nil {
				t.Fatalf("SIDToUnixID(%q) second call failed: %v", tt.sid, err)
			}

			if gotUnixID != gotUnixID2 {
				t.Errorf("SIDToUnixID(%q) not deterministic: first=%d, second=%d",
					tt.sid, gotUnixID, gotUnixID2)
			}

			// Verify the ID is within the configured range
			if gotUnixID < tt.config.IDRange.Min || gotUnixID > tt.config.IDRange.Max {
				t.Errorf("SIDToUnixID(%q) = %d, outside range [%d, %d]",
					tt.sid, gotUnixID, tt.config.IDRange.Min, tt.config.IDRange.Max)
			}
		})
	}
}

func TestDecodeSID(t *testing.T) {
	tests := []struct {
		name    string
		hexSID  string
		wantSID string
		wantErr bool
	}{
		{
			name:    "example",
			hexSID:  "01050000000000051500000025ec493a619500b06dc9700a2fe80500",
			wantSID: "S-1-5-21-977923109-2952828257-175163757-387119",
			wantErr: false,
		},
		{
			name:    "EXAMPLE domain administrator",
			hexSID:  "010500000000000515000000c7f7fed77c7755c8945ace01f4010000",
			wantSID: "S-1-5-21-3623811015-3361044348-30300820-500",
			wantErr: false,
		},
		{
			name:    "EXAMPLE domain user 1013",
			hexSID:  "010500000000000515000000c7f7fed77c7755c8945ace01f5030000",
			wantSID: "S-1-5-21-3623811015-3361044348-30300820-1013",
			wantErr: false,
		},
		{
			name:    "well-known SID - Everyone",
			hexSID:  "010100000000000100000000",
			wantSID: "S-1-1-0",
			wantErr: false,
		},
		{
			name:    "well-known SID - Local System",
			hexSID:  "010100000000000512000000",
			wantSID: "S-1-5-18",
			wantErr: false,
		},
		{
			name:    "well-known SID - Authenticated Users",
			hexSID:  "010100000000000512000000",
			wantSID: "S-1-5-18",
			wantErr: false,
		},
		{
			name:    "SID too short - only 7 bytes",
			hexSID:  "01050000000000",
			wantSID: "",
			wantErr: true,
		},
		{
			name:    "SID too short - empty",
			hexSID:  "",
			wantSID: "",
			wantErr: true,
		},
		{
			name:    "invalid length - header says 5 sub-authorities but data missing",
			hexSID:  "010500000000000515000000",
			wantSID: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sidBytes, _ := hex.DecodeString(tt.hexSID)
			gotSID, err := idmap.DecodeSID(sidBytes)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DecodeSID() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("DecodeSID() unexpected error: %v", err)
			}

			if gotSID != tt.wantSID {
				t.Errorf("DecodeSID() = %q, want %q", gotSID, tt.wantSID)
			}
		})
	}
}
