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
	Update() time.Time
	Join() *Client
}

// NewInc returns a new Inc.
func NewInc() Inc {
	inc := &internalInc{}
	inc.cond = sync.NewCond(inc.lock.RLocker())
	return inc
}

type internalInc struct {
	lock sync.RWMutex
	at   time.Time
	cond *sync.Cond // using lock.RLock
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

func (i *internalInc) Join() *Client {
	return &Client{inc: i}
}

// Client is a client to an Inc. It can be shared between goroutines or copied, but each change
// may only wake up a single goroutine.
type Client struct {
	inc *internalInc
	at  time.Time
}

// Wait blocks forever until the Inc is notified, returning the new version.
func (c *Client) Wait() time.Time {
	return c.internalWait(make(chan time.Time))
}

// WaitDelay is as per Wait, but returns the old version after the specified duration.
func (c *Client) WaitDelay(d time.Duration) time.Time {
	return c.internalWait(time.After(d))
}

func (c *Client) internalWait(after <-chan time.Time) time.Time {
	done := false

	c.inc.lock.RLock()
	defer c.inc.lock.RUnlock()

	for {
		if c.at.Before(c.inc.at) {
			c.at = c.inc.at
			break
		}
		if done {
			break
		}

		signalCh := make(chan bool)
		go func() {
			c.inc.cond.Wait()
			signalCh <- true
		}()

		select {
		case <-after:
			c.inc.cond.Broadcast() // wake up clients to prevent memory leaks
			<-signalCh             // the above goroutine will finish and call us
			done = true
		case <-signalCh:
			// do nothing, just check next iteration
		}
	}

	return c.at
}
