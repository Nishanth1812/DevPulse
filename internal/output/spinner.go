package output

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

type Spinner struct {
	message  string
	disabled bool
	done     chan struct{}
	once     sync.Once
	wg       sync.WaitGroup
}

// NewSpinner returns a spinner. The spinner renders to stderr, so TTY probing
// must check stderr (not stdout) — otherwise the \r control sequences corrupt
// redirected stderr when stdout is piped.
func NewSpinner(noColor bool) *Spinner {
	return &Spinner{
		disabled: noColor || !term.IsTerminal(int(os.Stderr.Fd())),
	}
}

func (s *Spinner) Start(message string) {
	if s.disabled {
		return
	}

	s.message = message
	s.done = make(chan struct{})
	s.wg.Add(1)

	go func() {
		defer s.wg.Done()

		frames := []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		index := 0
		for {
			select {
			case <-s.done:
				return
			case <-ticker.C:
				frame := frames[index%len(frames)]
				_, _ = fmt.Fprintf(os.Stderr, "\r%c %s", frame, s.message)
				index++
			}
		}
	}()
}

func (s *Spinner) Stop() {
	if s.disabled || s.done == nil {
		return
	}

	s.once.Do(func() {
		close(s.done)
		s.wg.Wait()
		clearLine := strings.Repeat(" ", len(s.message)+4)
		_, _ = fmt.Fprintf(os.Stderr, "\r%s\r", clearLine)
	})
}
