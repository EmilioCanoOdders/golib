package golib

import (
	"os"
	"sync/atomic"
	"time"
)

// Now returns the current time, as time.Now does, for a game whose world
// follows the wall clock instead of its updates, such as one that goes on
// while the player is in another program:
//
//	elapsed := golib.Now().Sub(g.last)
//
// Under golib shot, which runs updates as fast as it can, Now returns instead
// the time the program started plus 1/60 of a second for every update run so
// far, so such a world moves in screenshots as it does when played, and the
// screenshots repeat. Timing that can follow the updates should add up dt
// instead (see "Time" in the package documentation).
func Now() time.Time {
	if shotClock.on.Load() {
		// Multiplying before dividing keeps 60 updates at exactly a second.
		return shotClock.start.Add(time.Duration(shotClock.updates.Load()) * time.Second / updatesPerSecond)
	}
	return time.Now()
}

// shotClock is Now's clock under golib shot: when the program started, and
// how many updates runShots has run.
var shotClock struct {
	on      atomic.Bool
	start   time.Time
	updates atomic.Int64
}

// init starts Now's clock for golib shot before main runs, so the time a game
// reads while it sets up, before Run, is the run's start too.
func init() { startShotClock(os.Getenv) }

// startShotClock starts Now's clock at the time it is called, with no updates
// run, when getenv says golib shot runs the program.
func startShotClock(getenv func(string) string) {
	if getenv(shotDirEnv) == "" {
		return
	}
	shotClock.start = time.Now()
	shotClock.updates.Store(0)
	shotClock.on.Store(true)
}
