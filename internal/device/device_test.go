package device

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrCreateIsStableAndMachineLocal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "device.json")
	first, err := LoadOrCreate(path)
	if err != nil {
		t.Fatal(err)
	}
	second, err := LoadOrCreate(path)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == "" || first.ID != second.ID {
		t.Fatalf("device IDs: %#v %#v", first, second)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
