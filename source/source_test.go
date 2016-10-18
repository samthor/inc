package inc

import (
	"testing"
)

func TestSource(t *testing.T) {
	src := New()

	_, sub := src.Sub()

	src.Update(1)

	x := <-sub
	if expected := 1; x != expected {
		t.Errorf("expected data %v, got %v", expected, x)
	}

}
