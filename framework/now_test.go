package golib

import (
	"testing"
	"time"
)

// Outside golib shot, Now is the wall clock.
func TestNowIsTheWallClock(t *testing.T) {
	if shotClock.on.Load() {
		t.Skip("the tests run under golib shot")
	}
	before := time.Now()
	now := Now()
	if now.Before(before) || now.Sub(before) > time.Second {
		t.Errorf("Now() = %v, want the wall clock, about %v", now, before)
	}
}

// Under golib shot, Now starts when the program does and moves 1/60 s for
// every update, whatever time passes on the wall clock.
func TestNowFollowsTheUpdatesUnderShot(t *testing.T) {
	t.Cleanup(func() {
		shotClock.on.Store(false)
		shotClock.updates.Store(0)
	})
	startShotClock(func(name string) string { return "" })
	if shotClock.on.Load() {
		t.Fatal("the shot clock started with no golib shot variables set")
	}

	startShotClock(func(name string) string {
		if name == shotDirEnv {
			return "shots"
		}
		return ""
	})
	start := Now()
	time.Sleep(20 * time.Millisecond)
	if got := Now(); !got.Equal(start) {
		t.Errorf("before any update, Now moved %v with the wall clock, want it still", got.Sub(start))
	}
	for _, tt := range []struct {
		updates int64
		want    time.Duration
	}{
		{1, 16666666 * time.Nanosecond},
		{60, time.Second},
		{90, 1500 * time.Millisecond},
		{3600, time.Minute},
	} {
		shotClock.updates.Store(tt.updates)
		if got := Now().Sub(start); got != tt.want {
			t.Errorf("after %d updates, Now is %v after the start, want %v", tt.updates, got, tt.want)
		}
	}
}
