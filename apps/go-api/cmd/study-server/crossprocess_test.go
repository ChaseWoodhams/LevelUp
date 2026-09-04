package main

// crossprocess_test.go — THE READ-ONLY CRITERION, PROVED WHERE IT ACTUALLY HAPPENS.
//
// The ticket asks that the archive be "opened read-only, so serving while an archive run is
// writing is safe". A same-process test cannot prove that: in one process
// `duckdb.OpenReadForQuery` BORROWS the cached read-write handle, which is the one arrangement
// that never occurs in production — `study-archiver watch` is a separate invocation from the
// OS scheduler, and this server is a separate process again.
//
// So the writer here is a REAL SECOND PROCESS: the test binary re-executed against a hidden
// test that opens the archive read-write and holds it. Both directions matter, and the second
// one matters more:
//
//   - a server started while a capture is running must be able to read;
//   - a server left running must NOT stop the next hourly capture from opening read-write.
//
// The second is the one that could break the archive's whole reason for existing. A study
// server that quietly blocked the hourly pass would cost films, and films expire.

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ddb "levelup/go-api/internal/platform/duckdb"
)

// holdEnv names the archive the child process must hold, and how. Its presence is what turns
// a re-executed test binary into the writer (or reader) standing in for the other tool.
const (
	holdEnvPath = "STUDY_TEST_HOLD_ARCHIVE"
	holdEnvMode = "STUDY_TEST_HOLD_MODE"
	holdReady   = "HELD"
	holdTimeout = 30 * time.Second
)

// TestHoldsArchive is not a test. It is the child process: it opens the archive in the
// requested mode, says so on stdout, and waits to be killed.
//
// It lives as a Test function because that is how a Go package re-executes itself — there is
// no other entry point into a test binary — and it skips instantly when the environment does
// not ask for it, so a normal run never sees it.
func TestHoldsArchive(t *testing.T) {
	path := os.Getenv(holdEnvPath)
	if path == "" {
		t.Skip("child-process helper; runs only when " + holdEnvPath + " is set")
	}
	var err error
	if os.Getenv(holdEnvMode) == "rw" {
		var db *ddb.DB
		if db, err = ddb.OpenReadWrite(path); err == nil {
			defer func() { _ = db.Close() }()
		}
	} else {
		var release func()
		if _, release, err = ddb.OpenReadForQuery(path); err == nil {
			defer release()
		}
	}
	if err != nil {
		fmt.Println("FAILED " + err.Error())
		return
	}
	fmt.Println(holdReady)
	time.Sleep(holdTimeout)
}

// holdArchiveInAnotherProcess starts the child and returns once it reports the file held.
func holdArchiveInAnotherProcess(t *testing.T, path, mode string) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestHoldsArchive$", "-test.v")
	cmd.Env = append(os.Environ(), holdEnvPath+"="+path, holdEnvMode+"="+mode)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("child stdout: %v", err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting the %s holder: %v", mode, err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	scan := bufio.NewScanner(stdout)
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if line == holdReady {
			return
		}
		if strings.HasPrefix(line, "FAILED ") {
			t.Fatalf("the %s holder could not open the archive: %s", mode, strings.TrimPrefix(line, "FAILED "))
		}
	}
	t.Fatalf("the %s holder never reported holding the archive", mode)
}

// TestServerStartsWhileACaptureRuns — a capture in progress must not stop the server booting.
//
// This is what `newArchiveSource` buys: it locates the archive without opening it, so startup
// does not need a lock it cannot have. The earlier design opened at boot and exited 1 here.
func TestServerStartsWhileACaptureRuns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive.duckdb")
	seedArchive(t, path)
	holdArchiveInAnotherProcess(t, path, "rw")

	if _, err := newArchiveSource(path); err != nil {
		t.Fatalf("the server must start while another process writes the archive: %v", err)
	}
}

// TestRequestDuringACaptureIsBusyNotBroken — the honest degradation. DuckDB is
// single-instance-per-file across processes, so while a capture holds the archive this server
// genuinely cannot read it. What it must NOT do is hang, or call that a fault: the answer is
// errArchiveBusy, which the handler publishes as a retryable 503.
func TestRequestDuringACaptureIsBusyNotBroken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive.duckdb")
	seedArchive(t, path)
	src, err := newArchiveSource(path)
	if err != nil {
		t.Fatalf("newArchiveSource: %v", err)
	}
	holdArchiveInAnotherProcess(t, path, "rw")

	start := time.Now()
	_, err = src.open(context.Background())
	if !errors.Is(err, errArchiveBusy) {
		t.Fatalf("err = %v, want errArchiveBusy", err)
	}
	// The retry budget is bounded on purpose: a capture runs for minutes, and a request that
	// waited that long would be worse than one that says "busy".
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("gave up after %s; the retry budget must stay short", elapsed)
	}
}

// TestArchiverCanWriteBetweenRequests is the direction that matters MOST, and it is about the
// archiver rather than about this server: a study server left running overnight must not be
// what stops the next hourly capture. Films expire; a blocked pass costs them.
//
// THIS TEST FAILED against the first design, which held the handle for the server's lifetime.
// That is the bug it exists to keep out.
//
// AND ITS NAME IS THE EXACT CLAIM. It proves the archiver can take the file BETWEEN requests,
// not during one — while a request is in flight the archiver waits, exactly as this server
// waits during a capture, and no arrangement of two DuckDB processes avoids that. What makes
// the trade work is duration, not priority: a request lasts milliseconds and a capture lasts
// minutes, so the hourly pass has all day to find a gap.
func TestArchiverCanWriteBetweenRequests(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive.duckdb")
	seedArchive(t, path)
	src, err := newArchiveSource(path)
	if err != nil {
		t.Fatalf("newArchiveSource: %v", err)
	}

	// A request happens, and finishes, exactly as it does in production.
	a, err := src.open(context.Background())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := a.listMatches(context.Background(), mustFilter(t, rawFilter{})); err != nil {
		t.Fatalf("listMatches: %v", err)
	}
	a.Close()

	// The child stands in for `study-archiver watch`: it opens read-write exactly as the
	// archiver does, and reports whether the running server let it.
	holdArchiveInAnotherProcess(t, path, "rw")
}
