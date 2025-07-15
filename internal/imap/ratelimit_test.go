package imap

import (
	"context"
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	maxTokens := 10
	refillRate := 100 * time.Millisecond
	
	rl := NewRateLimiter(maxTokens, refillRate)
	
	if rl.maxTokens != maxTokens {
		t.Errorf("Expected maxTokens %d, got %d", maxTokens, rl.maxTokens)
	}
	
	if rl.refillRate != refillRate {
		t.Errorf("Expected refillRate %v, got %v", refillRate, rl.refillRate)
	}
	
	if rl.tokens != maxTokens {
		t.Errorf("Expected initial tokens %d, got %d", maxTokens, rl.tokens)
	}
}

func TestDefaultRateLimiter(t *testing.T) {
	rl := DefaultRateLimiter()
	
	if rl.maxTokens != 20 {
		t.Errorf("Expected maxTokens 20, got %d", rl.maxTokens)
	}
	
	if rl.refillRate != 100*time.Millisecond {
		t.Errorf("Expected refillRate 100ms, got %v", rl.refillRate)
	}
	
	if rl.tokens != 20 {
		t.Errorf("Expected initial tokens 20, got %d", rl.tokens)
	}
}

func TestRateLimiter_TryTake(t *testing.T) {
	rl := NewRateLimiter(2, 100*time.Millisecond)
	
	// Should be able to take 2 tokens
	if !rl.TryTake() {
		t.Error("Expected to be able to take first token")
	}
	
	if !rl.TryTake() {
		t.Error("Expected to be able to take second token")
	}
	
	// Should not be able to take a third token immediately
	if rl.TryTake() {
		t.Error("Expected not to be able to take third token immediately")
	}
	
	// Check tokens available
	if rl.TokensAvailable() != 0 {
		t.Errorf("Expected 0 tokens available, got %d", rl.TokensAvailable())
	}
}

func TestRateLimiter_Refill(t *testing.T) {
	rl := NewRateLimiter(2, 50*time.Millisecond)
	
	// Take all tokens
	rl.TryTake()
	rl.TryTake()
	
	if rl.TokensAvailable() != 0 {
		t.Errorf("Expected 0 tokens after taking all, got %d", rl.TokensAvailable())
	}
	
	// Wait for refill
	time.Sleep(60 * time.Millisecond)
	
	// Should have at least 1 token after refill
	if rl.TokensAvailable() < 1 {
		t.Errorf("Expected at least 1 token after refill, got %d", rl.TokensAvailable())
	}
}

func TestRateLimiter_Wait(t *testing.T) {
	rl := NewRateLimiter(1, 50*time.Millisecond)
	
	// Take the only token
	if !rl.TryTake() {
		t.Error("Expected to be able to take first token")
	}
	
	// Wait should succeed after refill
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	start := time.Now()
	err := rl.Wait(ctx)
	elapsed := time.Since(start)
	
	if err != nil {
		t.Errorf("Expected Wait to succeed, got error: %v", err)
	}
	
	if elapsed < 40*time.Millisecond {
		t.Errorf("Expected Wait to take at least 40ms, took %v", elapsed)
	}
}

func TestRateLimiter_WaitTimeout(t *testing.T) {
	rl := NewRateLimiter(1, 200*time.Millisecond) // Slow refill
	
	// Take the only token
	if !rl.TryTake() {
		t.Error("Expected to be able to take first token")
	}
	
	// Wait should timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	
	err := rl.Wait(ctx)
	
	if err == nil {
		t.Error("Expected Wait to timeout")
	}
	
	if err != context.DeadlineExceeded {
		t.Errorf("Expected DeadlineExceeded, got %v", err)
	}
}

func TestRateLimiter_WaitCancellation(t *testing.T) {
	rl := NewRateLimiter(1, 200*time.Millisecond) // Slow refill
	
	// Take the only token
	if !rl.TryTake() {
		t.Error("Expected to be able to take first token")
	}
	
	// Cancel the context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	
	err := rl.Wait(ctx)
	
	if err == nil {
		t.Error("Expected Wait to be cancelled")
	}
	
	if err != context.Canceled {
		t.Errorf("Expected Canceled, got %v", err)
	}
}

func TestRateLimiter_Reset(t *testing.T) {
	rl := NewRateLimiter(5, 100*time.Millisecond)
	
	// Take some tokens
	rl.TryTake()
	rl.TryTake()
	rl.TryTake()
	
	if rl.TokensAvailable() != 2 {
		t.Errorf("Expected 2 tokens before reset, got %d", rl.TokensAvailable())
	}
	
	// Reset should restore all tokens
	rl.Reset()
	
	if rl.TokensAvailable() != 5 {
		t.Errorf("Expected 5 tokens after reset, got %d", rl.TokensAvailable())
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	rl := NewRateLimiter(10, 10*time.Millisecond)
	
	// Test concurrent access
	done := make(chan bool, 20)
	
	for i := 0; i < 20; i++ {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			
			err := rl.Wait(ctx)
			if err != nil {
				t.Errorf("Unexpected error in concurrent access: %v", err)
			}
			done <- true
		}()
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 20; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for concurrent access test")
		}
	}
}

func TestRateLimiter_RefillDoesNotExceedMax(t *testing.T) {
	rl := NewRateLimiter(3, 10*time.Millisecond)
	
	// Wait for potential refill
	time.Sleep(100 * time.Millisecond)
	
	// Should not exceed max tokens
	if rl.TokensAvailable() > 3 {
		t.Errorf("Expected tokens not to exceed 3, got %d", rl.TokensAvailable())
	}
}

func TestGlobalRateLimiter(t *testing.T) {
	// Test that global rate limiter is available
	if GlobalRateLimiter == nil {
		t.Error("GlobalRateLimiter should not be nil")
	}
	
	// Test convenience functions
	if !TryRateLimit() {
		t.Error("Expected TryRateLimit to succeed initially")
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	err := WaitForRateLimit(ctx)
	if err != nil {
		t.Errorf("Expected WaitForRateLimit to succeed, got error: %v", err)
	}
}

func TestRateLimiter_TokensRefillOverTime(t *testing.T) {
	rl := NewRateLimiter(5, 20*time.Millisecond)
	
	// Take all tokens
	for i := 0; i < 5; i++ {
		if !rl.TryTake() {
			t.Errorf("Expected to take token %d", i)
		}
	}
	
	// Should have 0 tokens
	if rl.TokensAvailable() != 0 {
		t.Errorf("Expected 0 tokens after taking all, got %d", rl.TokensAvailable())
	}
	
	// Wait for 2.5 refill periods
	time.Sleep(50 * time.Millisecond)
	
	// Should have at least 2 tokens (50ms / 20ms = 2.5)
	available := rl.TokensAvailable()
	if available < 2 {
		t.Errorf("Expected at least 2 tokens after 50ms, got %d", available)
	}
	
	// Should not exceed max tokens
	if available > 5 {
		t.Errorf("Expected tokens not to exceed 5, got %d", available)
	}
}

func BenchmarkRateLimiter_TryTake(b *testing.B) {
	rl := NewRateLimiter(1000, time.Microsecond)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rl.TryTake()
		}
	})
}

func BenchmarkRateLimiter_Wait(b *testing.B) {
	rl := NewRateLimiter(1000, time.Microsecond)
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.Wait(ctx)
	}
}