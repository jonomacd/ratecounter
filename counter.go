package ratecounter

import "sync/atomic"

// A Counter is a thread-safe counter implementation
type Counter uint32

// Incr method increments the counter by some value
func (c *Counter) Incr(val int64) {
	atomic.AddUint32((*uint32)(c), uint32(val))
}

// Reset method resets the counter's value to zero
func (c *Counter) Reset() {
	atomic.StoreUint32((*uint32)(c), 0)
}

// Value method returns the counter's current value
func (c *Counter) Value() int64 {
	return int64(atomic.LoadUint32((*uint32)(c)))
}
