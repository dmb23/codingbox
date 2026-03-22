package unit

import (
	"testing"

	"github.com/codingbox/codingbox/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestCanTransitionTo_ValidTransitions(t *testing.T) {
	tests := []struct {
		from string
		to   string
		ok   bool
	}{
		{models.StatusCreated, models.StatusRunning, true},
		{models.StatusCreated, models.StatusFailed, true},
		{models.StatusRunning, models.StatusStopped, true},
		{models.StatusRunning, models.StatusFailed, true},
		// Invalid transitions
		{models.StatusCreated, models.StatusStopped, false},
		{models.StatusRunning, models.StatusCreated, false},
		{models.StatusStopped, models.StatusRunning, false},
		{models.StatusStopped, models.StatusFailed, false},
		{models.StatusFailed, models.StatusRunning, false},
		{models.StatusFailed, models.StatusStopped, false},
	}

	for _, tt := range tests {
		t.Run(tt.from+"->"+tt.to, func(t *testing.T) {
			s := &models.SandboxSession{Status: tt.from}
			assert.Equal(t, tt.ok, s.CanTransitionTo(tt.to))
		})
	}
}
