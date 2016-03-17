package inc

import (
	"time"
)

// Client is a client to an Inc. It can be copied to fork state, but each instance should only
// be used by a single instance at a time.
type Client struct {
	Inc Inc
	at  time.Time
}

// Wait blocks forever until the Inc is notified, returning the new version.
func (c *Client) Wait() time.Time {
	c.at = c.Inc.Wait(c.at, nil)
	return c.at
}

// WaitDelay is as per Wait, but returns the old version after the specified duration.
func (c *Client) WaitDelay(d time.Duration) time.Time {
	ch := make(chan bool)
	go func() {
		<-time.After(d)
		ch <- true
	}()
	c.at = c.Inc.Wait(c.at, ch)
	return c.at
}
