Provides `Inc`, a helper for concurrent Go applications.

Each instance encapsulates an always-increasing `time.Time` as a monotonically increasing version number.
Clients can wait for the version to change.

Good usages for this object are to guard some external state, e.g., the current state of physical sensors where history is not important.

## Usage

Basic, direct usage.

```go
i := inc.New()

// on one goroutine
var ver time.Time
ver = i.Wait(ver, nil)

// on another goroutine
i.Update()
```
