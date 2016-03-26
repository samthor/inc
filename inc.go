// Package inc is a way for multiple listeners to listen to some ever-changing data, with a version
// number based on the current time. Listeners will be informed if they are out of date and if
// there is new data.
package inc

import (
	"sync"
	"time"
)

// Inc is an object that changes over time.
type Inc interface {
	// At returns the current version of this Inc.
	At() time.Time

	// Update increments the version of this Inc, to now or later.
	Update() time.Time

	// Wait delays until the version passes the specified time, or a message is received on the
	// channel. This returns the new version. The specified channel, if non-nil, will always be
	// read from.
	Wait(time.Time, <-chan bool) time.Time

	// Pend returns a channel that returns the next version after the specified time. This request
	// can be canceled by sending a bool on returned write-only channel. Callers must either read
	// or write to the returned channels, respectively.
	Pend(time.Time) (<-chan time.Time, chan<- bool)
}

// New returns a new Inc.
func New() Inc {
	inc := &internalInc{}
	inc.cond = sync.NewCond(inc.lock.RLocker())
	return inc
}

type internalInc struct {
	lock sync.RWMutex
	at   time.Time
	cond *sync.Cond // using lock.RLock
}

func (i *internalInc) At() time.Time {
	i.lock.RLock()
	defer i.lock.RUnlock()
	return i.at
}

func (i *internalInc) Update() time.Time {
	i.lock.Lock()
	defer i.lock.Unlock()

	// update i.at, ensure it always moves forward
	n := time.Now()
	if n.Before(i.at) {
		n = i.at.Add(1)
	}
	i.at = n

	i.cond.Broadcast()
	return n
}

func (i *internalInc) Pend(t time.Time) (<-chan time.Time, chan<- bool) {
	out := make(chan time.Time)
	cancel := make(chan bool)

	go func() {
		i.lock.RLock()

		for !t.Before(i.at) {
			signalCh := make(chan bool)
			go func() {
				i.cond.Wait()
				signalCh <- true
			}()

			select {
			case <-cancel:
				i.cond.Broadcast() // wake up clients to prevent memory leaks
				<-signalCh         // the above goroutine will finish and call us
				i.lock.RUnlock()   // explicitly unlock
				return             // canceled, so give up
			case <-signalCh:
				// do nothing, just check next iteration
			}
		}

		i.lock.RUnlock()
		select {
		case out <- i.at:
			// ok
		case <-cancel:
			// canceled after all
		}
	}()

	return out, cancel
}

func (i *internalInc) Wait(t time.Time, after <-chan bool) time.Time {
	update, cancel := i.Pend(t)
	select {
	case <-after:
		cancel <- true
	case t = <-update:
		// great, t is anew

		if after != nil {
			go func() {
				<-after // prevent leak
			}()
		}
	}
	return t
}
