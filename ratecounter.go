package ratecounter

import (
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// A RateCounter is a thread-safe counter which returns the number of times
// 'Incr' has been called in the last interval
type RateCounter struct {
	counter  Counter
	partials []Counter
	// The last time a partial was reset
	resetTime uint64
	current   int32
	resetting bool
	interval  uint32
	sync.Mutex
}

// NewRateCounter Constructs a new RateCounter
func NewRateCounter(intrvl time.Duration) *RateCounter {
	rc := &RateCounter{
		partials:  make([]Counter, 20),
		resetTime: UnixMilli(),
		interval:  uint32(intrvl.Nanoseconds() / 1000000),
	}

	return rc
}

func (r *RateCounter) updatePartials(interval uint32, val int64) {
	// The number of time slices we keep within the interval
	resolution := len(r.partials)
	// The last time a partial was reset
	resetTime := atomic.LoadUint64(&r.resetTime)
	now := UnixMilli()
	timeDiff := float32(now - resetTime)

	// The interval of time a partial is responsible for
	partialInterval := float32(interval) / float32(resolution)
	// The next partial to drop

	//fmt.Printf("now: %v rt: %v td: %v, pi: %v\n", now, resetTime, timeDiff, partialInterval)

	// We are beyond at least one partial interval
	if timeDiff > partialInterval {
		// Make sure only one of us does the updating
		r.Lock()
		if !r.resetting {
			r.resetting = true
			r.Unlock()
			defer func() {
				r.Lock()
				r.resetting = false
				r.Unlock()
			}()
		} else {
			r.Unlock()
			// Someone else is doing it
			return
		}

	} else {
		// No need to update the partials
		return
	}

	current := atomic.LoadInt32(&r.current)

	// We can only get here if we are updating the partials. The resetting flag should protect things
	// such that only one can get in at a time
	for ii := 0; timeDiff > partialInterval && ii < resolution; ii++ {
		// We need to do this potentially many times if there hasn't been an update for a while
		timeDiff = timeDiff - partialInterval

		next := (int(current) + 1) % resolution

		// Remove the last partial from the current count
		r.counter.Incr(-1 * r.partials[next].Value())
		//fmt.Printf("\tcurrent %v, next: %v, dec rate: %v dropped: %v\n", current, next, r.counter, r.partials[next].Value())
		// Reset the count in that partial to make ready for next
		r.partials[next].Reset()
		// Set the reset partial as the current partial

		current = int32(next)
	}
	atomic.StoreInt32(&r.current, int32(current))

	atomic.StoreUint64(&r.resetTime, now)
}

// WithResolution determines the minimum resolution of this counter, default is 20
func (r *RateCounter) WithResolution(resolution int) *RateCounter {
	if resolution < 1 {
		panic("RateCounter resolution cannot be less than 1")
	}

	r.partials = make([]Counter, resolution)
	r.current = 0

	return r
}

// Incr Add an event into the RateCounter
func (r *RateCounter) Incr(val int64) {

	r.counter.Incr(val)
	r.updatePartials(r.interval, val)
	current := atomic.LoadInt32(&r.current)
	r.partials[current].Incr(val)
}

// Rate Return the current number of events in the last interval
func (r *RateCounter) Rate() int64 {
	r.updatePartials(r.interval, 0)
	return r.counter.Value()
}

func (r *RateCounter) String() string {

	return strconv.FormatInt(r.Rate(), 10)
}

func UnixMilli() uint64 {
	return uint64(time.Now().UnixNano() / 1000000)
}
