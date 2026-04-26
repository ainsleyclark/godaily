package news

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSource_String(t *testing.T) {
	input := SourceDevTo
	got := input.String()
	assert.IsType(t, got, "devto")
}
