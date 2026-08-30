package main

import "testing"

func TestBuildRuntimeWiresServices(t *testing.T) {
	runtime := buildRuntime(nil)

	if runtime.registry == nil {
		t.Fatal("runtime registry is nil")
	}
	if runtime.syncer == nil {
		t.Fatal("runtime syncer is nil")
	}
	if runtime.learning == nil {
		t.Fatal("runtime learning service is nil")
	}
	if runtime.gc == nil {
		t.Fatal("runtime GC service is nil")
	}
}
