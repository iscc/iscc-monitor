// Test for the SIGTERM shutdown trap: notifyShutdown registers SIGTERM (alongside
// SIGINT) on the process's shutdown context, so a container/orchestrator stop
// (which sends SIGTERM, not SIGINT) cancels the same context run() drives the
// follower loop on and drains rather than being SIGKILLed mid-commit. The test
// drives the notifyShutdown seam directly — not the whole of run() (which opens a
// real store and blocks on Loop.Run) — sends itself a real SIGTERM, and asserts the
// returned context cancels. It is non-vacuous: dropping syscall.SIGTERM from the
// registration leaves SIGTERM at its default disposition (terminate), so the
// context never cancels and this test hits its deadline.
//
// syscall.Kill / syscall.Getpid self-signalling is Unix-only, so the file is
// //go:build unix. The production syscall.SIGTERM registration in notifyShutdown is
// unconditional and compiles on every OS (incl. Windows); this is a platform-
// capability guard on the test's signalling mechanism, not a gate dodge.
//go:build unix

package main

import (
	"syscall"
	"testing"
	"time"
)

// TestSIGTERMCancelsShutdownContext proves notifyShutdown's context cancels when the
// process receives SIGTERM, so docker stop drains the in-flight poll and runs the
// deferred store.Close instead of cutting them off. It delivers a real in-process
// SIGTERM and asserts the context's Done channel fires within a short deadline. The
// returned stop() is deferred so the test un-registers the process-wide SIGTERM
// trap and never swallows a later signal in a sibling test.
func TestSIGTERMCancelsShutdownContext(t *testing.T) {
	ctx, stop := notifyShutdown()
	defer stop()

	if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("send SIGTERM to self: %v", err)
	}

	select {
	case <-ctx.Done():
		// Pass: SIGTERM cancelled the shutdown context.
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown context did not cancel within 2s of SIGTERM; notifyShutdown is not trapping SIGTERM")
	}
}
