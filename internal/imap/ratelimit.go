package imap

import (
	"context"
	"sync"
	"time"
)

// RateLimiter provides rate limiting for IMAP operations
type RateLimiter struct {
	mu         sync.Mutex
	tokens     int
	maxTokens  int
	refillRate time.Duration
	lastRefill time.Time
}

// NewRateLimiter creates a new rate limiter with the specified parameters
func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// DefaultRateLimiter returns a rate limiter with sensible defaults for IMAP operations
func DefaultRateLimiter() *RateLimiter {
	// Allow 10 operations per second with burst of 20
	return NewRateLimiter(20, 100*time.Millisecond)
}

// Wait blocks until a token is available or the context is cancelled
func (rl *RateLimiter) Wait(ctx context.Context) error {
	for {
		if rl.tryTake() {
			return nil
		}
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Millisecond):
			// Check again after a short delay
		}
	}
}

// TryTake attempts to take a token without blocking
func (rl *RateLimiter) TryTake() bool {
	return rl.tryTake()
}

// tryTake attempts to take a token, refilling if necessary
func (rl *RateLimiter) tryTake() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	rl.refill()
	
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}
	
	return false
}

// refill adds tokens based on elapsed time since last refill
func (rl *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	
	// Calculate how many tokens to add based on elapsed time
	tokensToAdd := int(elapsed / rl.refillRate)
	
	if tokensToAdd > 0 {
		rl.tokens += tokensToAdd
		if rl.tokens > rl.maxTokens {
			rl.tokens = rl.maxTokens
		}
		rl.lastRefill = now
	}
}

// TokensAvailable returns the current number of available tokens
func (rl *RateLimiter) TokensAvailable() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	rl.refill()
	return rl.tokens
}

// Reset resets the rate limiter to its initial state
func (rl *RateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	rl.tokens = rl.maxTokens
	rl.lastRefill = time.Now()
}

// GlobalRateLimiter provides a singleton rate limiter for IMAP operations
var GlobalRateLimiter = DefaultRateLimiter()

// WaitForRateLimit is a convenience function to wait for rate limit using the global limiter
func WaitForRateLimit(ctx context.Context) error {
	return GlobalRateLimiter.Wait(ctx)
}

// TryRateLimit is a convenience function to try rate limit using the global limiter
func TryRateLimit() bool {
	return GlobalRateLimiter.TryTake()
}