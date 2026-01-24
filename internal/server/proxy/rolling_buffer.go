package proxy

// RollingBuffer maintains a bounded buffer that keeps only the most recent bytes
type RollingBuffer struct {
	buf []byte
	max int
}

// NewRollingBuffer creates a new rolling buffer with the specified maximum size
func NewRollingBuffer(max int) *RollingBuffer {
	return &RollingBuffer{
		buf: make([]byte, 0, max),
		max: max,
	}
}

// Write adds bytes to the buffer, keeping only the most recent up to max capacity
func (r *RollingBuffer) Write(p []byte) {
	if len(p) >= r.max {
		// If the incoming data is larger than or equal to max, keep only the tail
		r.buf = append(r.buf[:0], p[len(p)-r.max:]...)
		return
	}

	r.buf = append(r.buf, p...)
	if len(r.buf) > r.max {
		// Trim the buffer to keep only the most recent bytes
		r.buf = r.buf[len(r.buf)-r.max:]
	}
}

// Bytes returns the current contents of the buffer
func (r *RollingBuffer) Bytes() []byte {
	return r.buf
}

// String returns the buffer contents as a string
func (r *RollingBuffer) String() string {
	return string(r.buf)
}

// Reset clears the buffer
func (r *RollingBuffer) Reset() {
	r.buf = r.buf[:0]
}

// Len returns the current length of the buffer
func (r *RollingBuffer) Len() int {
	return len(r.buf)
}

// Cap returns the maximum capacity of the buffer
func (r *RollingBuffer) Cap() int {
	return r.max
}