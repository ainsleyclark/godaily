package news

import "fmt"

var registry = map[Source]func() Fetcher{}

// Register associates a Source with a factory function.
// Called from each source package's init().
func Register(s Source, f func() Fetcher) {
	registry[s] = f
}

// Get returns a new Fetcher for the given Source.
func Get(s Source) (Fetcher, error) {
	f, ok := registry[s]
	if !ok {
		return nil, fmt.Errorf("no fetcher registered for source %q", s)
	}
	return f(), nil
}

// Validate checks that every entry in Sources has a registered fetcher.
// Call at startup or in tests to catch missing registrations early.
func Validate() error {
	for _, s := range Sources {
		if _, ok := registry[s]; !ok {
			return fmt.Errorf("no fetcher registered for source %q", s)
		}
	}
	return nil
}
