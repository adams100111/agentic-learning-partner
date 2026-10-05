package domain

import "testing"

func TestRegistryLoadsGoPack(t *testing.T) {
	registry := NewRegistry()
	pack, err := registry.Load("go")
	if err != nil {
		t.Fatal(err)
	}
	if pack.Domain != "go" {
		t.Fatalf("domain = %q", pack.Domain)
	}
	if pack.Version != "0.1.0" {
		t.Fatalf("version = %q", pack.Version)
	}
	if !pack.HasCompetency("go.runtime.context") {
		t.Fatal("go.runtime.context not found")
	}
	if len(pack.Competencies) < 30 {
		t.Fatalf("competency count = %d", len(pack.Competencies))
	}
}

func TestRegistryRejectsUnknownPack(t *testing.T) {
	_, err := NewRegistry().Load("rust")
	if err == nil {
		t.Fatal("expected unknown-domain error")
	}
}

func TestPackCompatibility(t *testing.T) {
	pack, err := NewRegistry().Load("go")
	if err != nil {
		t.Fatal(err)
	}
	ok, err := pack.Supports(">=0.1 <0.2")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected Go v0 pack to satisfy >=0.1 <0.2")
	}
}
