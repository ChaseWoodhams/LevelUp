package main

// helpers_test.go — the plumbing the fake Halo server needs, kept out of the tests
// themselves so the assertions stay readable.

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

// chunkNamePrefix is the blob file name the Halo manifest uses for a film chunk.
const chunkNamePrefix = "filmChunk"

func fmtChunkName(idx int) string { return fmt.Sprintf("%s%d", chunkNamePrefix, idx) }

// parseChunkIndex reads the chunk index back out of a blob file name.
func parseChunkIndex(base string) (int, error) {
	var idx int
	_, err := fmt.Sscanf(base, chunkNamePrefix+"%d", &idx)
	return idx, err
}

// zlibBytes compresses a payload the way the Halo blob CDN serves film chunks — raw
// zlib, which haloclient.downloadBlob inflates on the way in.
func zlibBytes(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write(data); err != nil {
		t.Fatalf("zlib write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zlib close: %v", err)
	}
	return buf.Bytes()
}

// redirectTo sends every request to the test server, whatever host the Halo client
// resolved. Cheaper and more faithful than injecting an endpoint resolver: it also
// catches the blob URLs, which the client builds from the manifest rather than from any
// endpoint table.
type redirectTo string

func (r redirectTo) RoundTrip(req *http.Request) (*http.Response, error) {
	target, err := url.Parse(string(r))
	if err != nil {
		return nil, err
	}
	req = req.Clone(req.Context())
	req.URL.Scheme = target.Scheme
	req.URL.Host = target.Host
	req.Host = target.Host
	return http.DefaultTransport.RoundTrip(req)
}
