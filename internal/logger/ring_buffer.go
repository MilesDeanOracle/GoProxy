package logger

import "sync"

// RingBuffer stores the newest log entries in memory for UI hydration.
type RingBuffer struct {
	mu      sync.RWMutex
	entries []Entry
	next    int
	full    bool
}

// NewRingBuffer creates a fixed-size log ring buffer.
func NewRingBuffer(size int) *RingBuffer {
	if size <= 0 {
		size = 1
	}
	return &RingBuffer{
		entries: make([]Entry, size),
	}
}

// Add stores a log entry, evicting the oldest entry when the buffer is full.
func (b *RingBuffer) Add(entry Entry) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.entries[b.next] = entry
	b.next = (b.next + 1) % len(b.entries)
	if b.next == 0 {
		b.full = true
	}
}

// Recent returns at most n entries in chronological order.
func (b *RingBuffer) Recent(n int) []Entry {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if n <= 0 {
		return nil
	}

	count := b.next
	if b.full {
		count = len(b.entries)
	}
	if n > count {
		n = count
	}

	all := make([]Entry, 0, count)
	if b.full {
		all = append(all, b.entries[b.next:]...)
		all = append(all, b.entries[:b.next]...)
	} else {
		all = append(all, b.entries[:b.next]...)
	}

	return append([]Entry(nil), all[len(all)-n:]...)
}
