package unit

import (
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
)

func TestConcurrentSessionIDs_AreUnique(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := ulid.Make().String()
		assert.False(t, ids[id], "duplicate ULID generated: %s", id)
		ids[id] = true
	}
}

func TestConcurrentSessionIDs_AreSortable(t *testing.T) {
	var prev string
	for i := 0; i < 10; i++ {
		id := ulid.Make().String()
		if prev != "" {
			assert.True(t, id > prev, "ULIDs should be monotonically increasing: %s <= %s", id, prev)
		}
		prev = id
	}
}
