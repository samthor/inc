package inc

import (
	"testing"
	"time"
)

func TestInc(t *testing.T) {
	i := New()
	i.Update()

	// wait immediately, has update
	c := Client{Inc: i}
	c.Wait()

	// many updates coalesced
	for c := 0; c < 5; c++ {
		i.Update()
	}
	c.Wait() // flush these updates

	actual := i.Update()
	expected := c.Wait()
	if !actual.Equal(expected) {
		t.Errorf("expected last update to match times")
	}

	done := make(chan time.Time)
	go func() {
		// acts as chan size 1
		done <- c.Wait()
	}()

	select {
	case <-done:
		t.Errorf("expected nothing more to be pending")
	case <-time.After(time.Millisecond):
		// fine
	}

	actual = c.WaitDelay(0)
	if !actual.Equal(expected) {
		t.Errorf("expected delay update to match previous")
	}

	at := i.At()
	if !at.Equal(actual) {
		t.Errorf("expected at version to equal previous")
	}
}

func TestPend(t *testing.T) {
	i := New()
	iv := i.Update()

	var v time.Time
	out, _ := i.Pend(v)

	v = <-out
	if !v.Equal(iv) {
		t.Errorf("expected pend version to equal updated")
	}

	_, cancel := i.Pend(v)
	cancel <- true // should never block

	_, cancel = i.Pend(v)
	i.Update()
	cancel <- true // should still never block, but is after loop
}
