package threading

import (
	"fmt"
	"strings"
	"sync"
)

type errorsSafe struct {
	errs []error
	mu   sync.Mutex
}

func newError() *errorsSafe {
	return &errorsSafe{}
}

func (e *errorsSafe) append(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.errs = append(e.errs, err)
}

func (e *errorsSafe) Error() string {
	var b strings.Builder
	var color Color
	add := func(s string) string {
		if color == Blue {
			color = Cyan
		} else {
			color = Blue
		}
		return color.Add(s)
	}
	for i, err := range e.errs {
		b.WriteString(add(fmt.Sprintf("(errMsg%d:%s)", i+1, err.Error())))
	}
	return b.String()
}
