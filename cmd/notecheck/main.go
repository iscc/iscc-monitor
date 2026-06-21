// Command notecheck is the fully-independent signature-parity oracle for the
// monitor's checkpoint verifier. It reads the verifier key from --vkey and the
// full checkpoint text from stdin, then runs the Go reference signed-note tooling
// — transparency-dev/formats/note.NewVerifier + golang.org/x/mod/sumdb/note.Open
// (the same code paths Tessera fsck uses). On success it prints "OK <name>" and
// exits 0; otherwise it exits non-zero.
//
// This binary is the EXTERNAL parity check for internal/logclient/verify.go: the
// strict len(n.Sigs)==0 || len(n.UnverifiedSigs)!=0 reject mirrors VerifyCheckpoint
// exactly, so a divergence between the monitor's own crypto path and the reference
// tooling is caught by shelling this oracle out in CI.
//
// Exit codes (CI relies on them): missing/bad --vkey or stdin read error → 2;
// note.Open failure or the strict-signature reject → 1; success → 0.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	f_note "github.com/transparency-dev/formats/note"
	"golang.org/x/mod/sumdb/note"
)

func main() {
	vkey := flag.String("vkey", "", "verifier key string (name+hash+base64)")
	flag.Parse()
	if *vkey == "" {
		fmt.Fprintln(os.Stderr, "missing --vkey")
		os.Exit(2)
	}
	name, err := run(*vkey, os.Stdin, os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCode(err))
	}
	fmt.Fprintf(os.Stdout, "OK %s\n", name)
}

// errKind distinguishes the two non-zero exit classes so main can map run's
// error to the documented exit code. setupError (bad vkey / stdin read) exits 2;
// any other error (note.Open failure or the strict-signature reject) exits 1.
type errKind struct {
	err   error
	setup bool
}

func (e *errKind) Error() string { return e.err.Error() }

// exitCode maps a run error to its documented process exit code: a setup error
// (bad vkey or stdin read) is 2, any verification error is 1.
func exitCode(err error) int {
	if k, ok := err.(*errKind); ok && k.setup {
		return 2
	}
	return 1
}

// run is the testable core: it builds the verifier from vkey, reads the full
// checkpoint from in, opens it as a signed note, and applies the strict reject
// (no verified sig OR any unverified sig). It returns the signer name on success.
//
// A bad vkey or a stdin read failure is wrapped as a setup error (exit 2); a
// note.Open failure or the strict-signature reject is a verification error
// (exit 1). The strict reject is the load-bearing parity invariant shared with
// internal/logclient/verify.go — keep it identical.
func run(vkey string, in io.Reader, out io.Writer) (string, error) {
	v, err := f_note.NewVerifier(vkey)
	if err != nil {
		return "", &errKind{err: fmt.Errorf("bad vkey: %w", err), setup: true}
	}
	body, err := io.ReadAll(in)
	if err != nil {
		return "", &errKind{err: fmt.Errorf("read stdin: %w", err), setup: true}
	}
	n, err := note.Open(body, note.VerifierList(v))
	if err != nil {
		return "", fmt.Errorf("note.Open failed: %w", err)
	}
	if len(n.Sigs) == 0 || len(n.UnverifiedSigs) != 0 {
		return "", fmt.Errorf("unexpected signatures: verified=%d unverified=%d", len(n.Sigs), len(n.UnverifiedSigs))
	}
	return n.Sigs[0].Name, nil
}
