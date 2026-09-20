package worker

import "sync/atomic"

type TargetGuard struct {
	running atomic.Bool
}

func (g *TargetGuard) TryAcquire() bool {
	return g.running.CompareAndSwap(false, true)
}

func (g *TargetGuard) Release() {
	g.running.Store(false)
}

//"If nobody is currently running this target, atomically mark it as running and give me the slot."
//If another worker gets there at exactly the same time, only one wins.
