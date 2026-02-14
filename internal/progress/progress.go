package progress

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

var frames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner displays an animated spinner with a status message on a terminal.
// For non-terminal writers, it prints each status update as a new line.
type Spinner struct {
	w        io.Writer
	mu       sync.Mutex
	message  string
	done     chan struct{}
	exited   chan struct{}
	started  bool
	stopOnce sync.Once
	isTTY    bool
}

// New creates a new Spinner that writes to w.
func New(w io.Writer) *Spinner {
	return &Spinner{
		w:      w,
		done:   make(chan struct{}),
		exited: make(chan struct{}),
		isTTY:  isTerminal(w),
	}
}

// Start begins the spinner animation with the given message.
func (s *Spinner) Start(msg string) {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return
	}
	s.started = true
	s.message = msg
	s.mu.Unlock()

	if s.isTTY {
		go s.run()
	} else {
		_, _ = fmt.Fprintf(s.w, "%s\n", msg)
		close(s.exited)
	}
}

func (s *Spinner) run() {
	defer close(s.exited)
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()
	i := 0
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.mu.Lock()
			msg := s.message
			s.mu.Unlock()
			_, _ = fmt.Fprintf(s.w, "\r\033[K%s %s", frames[i%len(frames)], msg)
			i++
		}
	}
}

// Update changes the spinner's status message.
func (s *Spinner) Update(msg string) {
	s.mu.Lock()
	prev := s.message
	s.message = msg
	started := s.started
	s.mu.Unlock()

	if !s.isTTY && started && msg != prev {
		_, _ = fmt.Fprintf(s.w, "%s\n", msg)
	}
}

// Stop halts the spinner and clears the line.
// Safe to call multiple times and even if Start was never called.
func (s *Spinner) Stop() {
	if !s.started {
		return
	}
	s.stopOnce.Do(func() {
		close(s.done)
		<-s.exited
		if s.isTTY {
			_, _ = fmt.Fprintf(s.w, "\r\033[K")
		}
	})
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}
