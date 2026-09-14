package hwconvert

import (
	"testing"

	pb "loadmon/protocol"

	oneview "github.com/spa-nsk/oneview2"
)

func TestResolveProcessorSpeedMHz(t *testing.T) {
	cases := []struct {
		name                     string
		operating, max, fallback int
		wantOperating, wantMax   int
	}{
		{"all present, unchanged", 2400, 2600, 3000, 2400, 2600},
		{"operating missing, use max", 0, 2600, 3000, 2600, 2600},
		{"operating and max missing, use fallback", 0, 0, 3000, 3000, 3000},
		{"max missing, use fallback", 2400, 0, 3000, 2400, 3000},
		{"nothing available", 0, 0, 0, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotOp, gotMax := resolveProcessorSpeedMHz(c.operating, c.max, c.fallback)
			if gotOp != c.wantOperating || gotMax != c.wantMax {
				t.Errorf("resolveProcessorSpeedMHz(%d, %d, %d) = (%d, %d), want (%d, %d)",
					c.operating, c.max, c.fallback, gotOp, gotMax, c.wantOperating, c.wantMax)
			}
		})
	}
}

func TestProcessorSpeedFallbackMHz(t *testing.T) {
	if got := processorSpeedFallbackMHz(nil); got != 0 {
		t.Errorf("processorSpeedFallbackMHz(nil) = %d, want 0", got)
	}

	procs := []oneview.ProcessorInfo{
		{OperatingSpeedMHz: 0, MaxSpeedMHz: 0},
		{OperatingSpeedMHz: 0, MaxSpeedMHz: 2100},
		{OperatingSpeedMHz: 2400, MaxSpeedMHz: 2600},
	}
	if got := processorSpeedFallbackMHz(procs); got != 2100 {
		t.Errorf("processorSpeedFallbackMHz = %d, want 2100 (first non-zero Max after skipping the zero entry)", got)
	}

	procsOperatingFirst := []oneview.ProcessorInfo{
		{OperatingSpeedMHz: 1900, MaxSpeedMHz: 2600},
	}
	if got := processorSpeedFallbackMHz(procsOperatingFirst); got != 1900 {
		t.Errorf("processorSpeedFallbackMHz = %d, want 1900 (OperatingSpeedMHz preferred over MaxSpeedMHz)", got)
	}
}

func TestServerHardwareToMessageNilSafety(t *testing.T) {
	// Should not panic on nil input/output.
	ServerHardwareToMessage(nil, nil)
	ServerHardwareToMessage(&oneview.Server{}, nil)
	ServerHardwareToMessage(nil, &pb.ServerHardware{})
}

func TestMessageToServerHardwareNilSafety(t *testing.T) {
	MessageToServerHardware(nil, nil)
	MessageToServerHardware(&pb.ServerHardware{}, nil)
	MessageToServerHardware(nil, &oneview.Server{})
}

func TestServerHardwareRoundTrip(t *testing.T) {
	in := &oneview.Server{
		Identity: oneview.ServerIdentity{
			Name:         "srv01",
			SerialNumber: "SN12345",
			Model:        "DL380 Gen10",
			PowerState:   "On",
		},
		Processors: []oneview.ProcessorInfo{
			{Model: "Xeon Gold 6248", TotalCores: 20, OperatingSpeedMHz: 2500, MaxSpeedMHz: 3900},
		},
	}

	msg := &pb.ServerHardware{}
	ServerHardwareToMessage(in, msg)

	if msg.Base == nil {
		t.Fatal("Base must not be nil after conversion")
	}
	if msg.Base.Name != "srv01" {
		t.Errorf("Base.Name = %q, want %q", msg.Base.Name, "srv01")
	}
	if msg.Base.SerialNumber != "SN12345" {
		t.Errorf("Base.SerialNumber = %q, want %q", msg.Base.SerialNumber, "SN12345")
	}
	if msg.Base.ProcessorCount != 1 {
		t.Errorf("Base.ProcessorCount = %d, want 1", msg.Base.ProcessorCount)
	}
	if len(msg.Processors) != 1 || msg.Processors[0].OperatingSpeedMHz != 2500 {
		t.Fatalf("Processors = %+v, want one entry with OperatingSpeedMHz=2500", msg.Processors)
	}

	out := &oneview.Server{}
	MessageToServerHardware(msg, out)

	if out.Identity.Name != in.Identity.Name {
		t.Errorf("round-trip Identity.Name = %q, want %q", out.Identity.Name, in.Identity.Name)
	}
	if out.Identity.SerialNumber != in.Identity.SerialNumber {
		t.Errorf("round-trip Identity.SerialNumber = %q, want %q", out.Identity.SerialNumber, in.Identity.SerialNumber)
	}
	if len(out.Processors) != 1 || out.Processors[0].OperatingSpeedMHz != 2500 || out.Processors[0].TotalCores != 20 {
		t.Errorf("round-trip Processors = %+v, want one entry matching input", out.Processors)
	}
}
