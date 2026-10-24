package main

import (
	"fmt"
	"os"
	"syscall"
)

// suiteLockPath is the advisory lock guarding integration-suite runs.
//
// Two concurrent instances sharing the dev DB (payroll-mysql) and its asynq
// queue pollute each other's fixtures: void idempotency keys consumed by the
// other run, BCC import jobs processed by the wrong checkout's working
// directory (ENOENT on the uploaded file), snapshots taken mid-write. The dev
// DB is localhost-bound, so a machine-local flock matches the pollution
// boundary exactly. The kernel releases the lock on ANY process exit —
// including os.Exit paths and crashes — so a dead lane can never wedge the
// next one.
const suiteLockPath = "/tmp/payroll-integration-suite.lock"

// acquireSuiteLock blocks until this process owns the suite lock. A run that
// has to wait says so once, so a queued lane reads as waiting, not hung.
func acquireSuiteLock() *os.File {
	f, err := os.OpenFile(suiteLockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n  FATAL: cannot open suite lock %s: %v\n\n", suiteLockPath, err)
		os.Exit(2)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err == nil {
		return f
	}
	fmt.Printf("🔒 Another integration-suite run holds the lock (%s) — this run waits for it to finish (fixture-pollution guard).\n", suiteLockPath)
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		fmt.Fprintf(os.Stderr, "\n  FATAL: waiting on suite lock: %v\n\n", err)
		os.Exit(2)
	}
	fmt.Println("🔓 Lock acquired — starting suite.")
	return f
}

// releaseSuiteLock lets a same-process rerun start immediately. The kernel
// drops the lock on process exit regardless, so callers may also just exit.
func releaseSuiteLock(f *os.File) {
	if f == nil {
		return
	}
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	_ = f.Close()
}
