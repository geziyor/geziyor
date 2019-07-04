package middleware

import (
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

func TestRandomDelay(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	delay := time.Millisecond * 1000
	min := float64(delay) * 0.5
	max := float64(delay) * 1.5
	randomDelay := rand.Intn(int(max-min)) + int(min)

	assert.True(t, time.Duration(randomDelay).Seconds() < 1.5)
	assert.True(t, time.Duration(randomDelay).Seconds() > 0.5)
}
