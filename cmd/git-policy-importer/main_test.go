package main

import (
	"flag"
	"testing"
)

func TestBespinDefault(t *testing.T) {
	flag.Parse()
	if *enableBespin {
		t.Errorf("Bespin should default to false, got true")
	}
}
