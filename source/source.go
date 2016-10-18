package inc

import (
	"sync"
)

// Source is something that has a current state and which can be subscribed to.
type Source interface {
	Sub() (interface{}, <-chan interface{})
	Update(interface{}) (bool, error)
}

func New() Source {
	return &internalSource{
		bufferSize: 5,
	}
}

type internalSource struct {
	lock       sync.Mutex
	bufferSize int
	state      interface{}
	subs       []chan interface{}
}

func (is *internalSource) Sub() (state interface{}, feed <-chan interface{}) {
	is.lock.Lock()
	defer is.lock.Unlock()

	ch := make(chan interface{}, is.bufferSize)
	is.subs = append(is.subs, ch)
	return nil, ch
}

func (is *internalSource) Update(update interface{}) (bool, error) {
	is.lock.Lock()
	defer is.lock.Unlock()

	var dead int
	for i, sub := range is.subs {
		select {
		case sub <- update:
			// ok
		default:
			close(sub) // sender closes chan
			is.subs[i] = nil
			dead++
		}
	}

	// if any entries were dead and nil'ed, then recreate the subs without them.
	if dead > 0 {
		tmp := is.subs
		is.subs = make([]chan interface{}, 0, len(tmp)-dead)
		for _, sub := range tmp {
			if sub != nil {
				is.subs = append(is.subs, sub)
			}
		}
	}

	return true, nil
}
