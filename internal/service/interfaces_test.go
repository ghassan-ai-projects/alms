package service

import (
	"testing"
)

// Verify that the interfaces compile correctly.
// This is a compile-time check — no runtime tests needed for pure interfaces.

func TestInterfacesCompile(t *testing.T) {
	// This is a compile-time verification that the interfaces exist.
	// Actual interface satisfaction is checked by the compiler.
	_ = (*AgentStore)(nil)
	_ = (*LearningStore)(nil)
	_ = (*ProtocolStore)(nil)
}
