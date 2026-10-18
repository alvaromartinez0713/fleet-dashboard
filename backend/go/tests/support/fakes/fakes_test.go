package fakes

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFakeClock(t *testing.T) {
	clock := NewFakeClock()
	
	now1 := clock.Now()
	assert.NotNil(t, now1)
	
	clock.Advance(10 * time.Second)
	now2 := clock.Now()
	assert.Equal(t, 10*time.Second, now2.Sub(now1))
	
	targetTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	clock.SetTime(targetTime)
	assert.Equal(t, targetTime, clock.Now())
}

func TestFakeIDGeneratorWithDefaults(t *testing.T) {
	gen := NewFakeIDGeneratorWithDefaults()
	
	id1 := gen.Generate()
	id2 := gen.Generate()
	assert.NotEqual(t, id1, id2)
	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	
	gen.ResetSequence()
	id3 := gen.Generate()
	assert.Equal(t, id1, id3)
}

func TestFakeRateLimiter(t *testing.T) {
	limiter := NewFakeRateLimiter(2)
	ctx := context.Background()
	
	assert.True(t, limiter.Allow(ctx, "VH-001"))
	assert.True(t, limiter.Allow(ctx, "VH-001"))
	assert.False(t, limiter.Allow(ctx, "VH-001"))
	assert.Equal(t, 3, limiter.GetCallCount("VH-001"))
	
	limiter.Reset()
	assert.Equal(t, 0, limiter.GetCallCount("VH-001"))
	assert.True(t, limiter.Allow(ctx, "VH-001"))
}

func TestFakeKafkaPublisher(t *testing.T) {
	pub := &FakeKafkaPublisher{}
	ctx := context.Background()
	
	assert.Equal(t, 0, pub.PublishedCount())
	pub.Reset()
	assert.Equal(t, 0, pub.PublishedCount())
	assert.Nil(t, pub.LastPublishedEvent())
	assert.Empty(t, pub.PublishedEvents())
	
	err := pub.Publish(ctx, "topic", "key", nil)
	assert.NoError(t, err)
}
