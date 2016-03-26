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

func (i *internalInc) Wait(t time.Time, after <-chan bool) time.Time {
	var done bool
	i.lock.RLock()
	defer i.lock.RUnlock()

	for {
		if t.Before(i.at) {
			t = i.at
			break
		}
		if done {
			break
		}

		signalCh := make(chan bool)
		go func() {
			i.cond.Wait()
			signalCh <- true
		}()

		select {
		case <-after:
			i.cond.Broadcast() // wake up clients to prevent memory leaks
			<-signalCh         // the above goroutine will finish and call us
			after = nil
			done = true
		case <-signalCh:
			// do nothing, just check next iteration
		}
	}

	if after != nil {
		go func() {
			<-after // prevent leak
		}()
	}
	return t
}
