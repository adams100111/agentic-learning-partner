package store

import (
	"errors"
	"testing"
)

func TestRequireReportsCapabilityAlternative(t *testing.T) {
	err := Require("local", Capabilities{
		CapabilityPersistence: true,
	}, CapabilitySync)
	if err == nil {
		t.Fatal("expected missing capability error")
	}
	var capabilityError CapabilityError
	if !errors.As(err, &capabilityError) {
		t.Fatalf("error type = %T", err)
	}
	if capabilityError.Provider != "local" || capabilityError.Capability != CapabilitySync {
		t.Fatalf("error = %#v", capabilityError)
	}
	if capabilityError.Alternative == "" {
		t.Fatal("expected actionable alternative")
	}
}
