//go:build linux

package aggregate

import (
	"bufio"
	"strings"
	"testing"
)

func TestParseCPUInfoScanner_x86(t *testing.T) {
	content := `processor	: 0
vendor_id	: GenuineIntel
cpu family	: 6
model		: 142
model name	: Intel(R) Core(TM) i7-8550U CPU @ 1.80GHz
stepping	: 10
`
	model := parseCPUInfoScanner(bufio.NewScanner(strings.NewReader(content)))
	expected := "Intel(R) Core(TM) i7-8550U CPU @ 1.80GHz"
	if model != expected {
		t.Fatalf("expected %q, got %q", expected, model)
	}
}

func TestParseCPUInfoScanner_ARMHardware(t *testing.T) {
	content := `processor	: 0
BogoMIPS	: 38.40
Features	: fp asimd evtstrm crc32 cpuid
CPU implementer	: 0x41
CPU architecture: 8
CPU variant	: 0x0
CPU part	: 0xd03
CPU revision	: 4

Hardware	: BCM2835
Revision	: a02082
`
	model := parseCPUInfoScanner(bufio.NewScanner(strings.NewReader(content)))
	expected := "BCM2835"
	if model != expected {
		t.Fatalf("expected %q, got %q", expected, model)
	}
}

func TestGetCPUModel_NotEmpty(t *testing.T) {
	model := getCPUModel()
	if strings.TrimSpace(model) == "" {
		t.Fatalf("expected non-empty CPU model, got empty string")
	}
}
