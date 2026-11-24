package s3

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/require"
)

var errReadFailed = errors.New("read failed")

// plainReader hides every method of the wrapped reader but Read, like a pipe or a decoder
type plainReader struct {
	io.Reader
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errReadFailed }

// cutReader gives its data and then io.ErrUnexpectedEOF, like a response body shorter than its
// Content-Length
type cutReader struct {
	data []byte
}

func (r *cutReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.ErrUnexpectedEOF
	}

	n := copy(p, r.data)
	r.data = r.data[n:]

	return n, nil
}

func readBody(t *testing.T, body io.Reader) []byte {
	t.Helper()

	data, err := io.ReadAll(body)
	require.NoError(t, err)

	return data
}

// rewound reads the body again from its start the way minio-go does before a retry
func rewound(t *testing.T, body io.Reader) []byte {
	t.Helper()

	seeker, ok := body.(io.Seeker)
	require.True(t, ok, "a body of a known length must be seekable for the retries of minio-go")

	_, err := seeker.Seek(0, io.SeekStart)
	require.NoError(t, err)

	return readBody(t, body)
}

func TestNewUploadSource(t *testing.T) {
	t.Parallel()

	data := []byte("0123456789abcdef")

	t.Run("bytes reader at its start is passed as is", func(t *testing.T) {
		t.Parallel()

		reader := bytes.NewReader(data)

		source, err := newUploadSource(reader)
		require.NoError(t, err)
		require.Same(t, reader, source.body)
		require.Equal(t, int64(len(data)), source.size)
		require.False(t, source.inOrder, "a ReaderAt goes in sections of itself")
	})

	t.Run("partly read bytes reader sends only the rest, also after a rewind", func(t *testing.T) {
		t.Parallel()

		reader := bytes.NewReader(data)
		_, err := reader.Read(make([]byte, 4))
		require.NoError(t, err)

		source, err := newUploadSource(reader)
		require.NoError(t, err)
		require.Equal(t, int64(len(data)-4), source.size)
		require.Equal(t, data[4:], readBody(t, source.body))
		require.Equal(t, data[4:], rewound(t, source.body))
	})

	t.Run("partly read strings reader", func(t *testing.T) {
		t.Parallel()

		reader := strings.NewReader(string(data))
		_, err := reader.Seek(10, io.SeekStart)
		require.NoError(t, err)

		source, err := newUploadSource(reader)
		require.NoError(t, err)
		require.Equal(t, int64(len(data)-10), source.size)
		require.Equal(t, data[10:], rewound(t, source.body))
	})

	t.Run("partly read regular file", func(t *testing.T) {
		t.Parallel()

		filename := filepath.Join(t.TempDir(), "object.bin")
		require.NoError(t, os.WriteFile(filename, data, 0o600))

		file, err := os.Open(filename)
		require.NoError(t, err)
		t.Cleanup(func() { _ = file.Close() })

		_, err = file.Seek(6, io.SeekStart)
		require.NoError(t, err)

		source, err := newUploadSource(file)
		require.NoError(t, err)
		require.Equal(t, int64(len(data)-6), source.size)
		require.Equal(t, data[6:], rewound(t, source.body))
	})

	t.Run("pipe is read ahead", func(t *testing.T) {
		t.Parallel()

		reader, writer, err := os.Pipe()
		require.NoError(t, err)
		t.Cleanup(func() { _ = reader.Close() })

		go func() {
			if _, err := writer.Write(data); err != nil {
				t.Errorf("write pipe: %v", err)
			}

			_ = writer.Close()
		}()

		source, err := newUploadSource(reader)
		require.NoError(t, err)
		require.Equal(t, int64(len(data)), source.size)
		require.Equal(t, data, rewound(t, source.body))
	})

	t.Run("short stream becomes one body of a known length", func(t *testing.T) {
		t.Parallel()

		source, err := newUploadSource(plainReader{bytes.NewReader(data)})
		require.NoError(t, err)
		require.Equal(t, int64(len(data)), source.size)
		require.Equal(t, data, rewound(t, source.body))
	})

	t.Run("empty stream", func(t *testing.T) {
		t.Parallel()

		source, err := newUploadSource(plainReader{bytes.NewReader(nil)})
		require.NoError(t, err)
		require.Equal(t, int64(0), source.size)
		require.Empty(t, readBody(t, source.body))
	})

	t.Run("stream of exactly the small read ahead size", func(t *testing.T) {
		t.Parallel()

		exact := bytes.Repeat([]byte{8}, readAheadSmallSize)

		source, err := newUploadSource(plainReader{bytes.NewReader(exact)})
		require.NoError(t, err)
		require.Equal(t, int64(len(exact)), source.size)
		require.Equal(t, exact, rewound(t, source.body))
	})

	t.Run("stream over the first read ahead buffer", func(t *testing.T) {
		t.Parallel()

		medium := bytes.Repeat([]byte{9}, 3*readAheadSmallSize+11)

		source, err := newUploadSource(plainReader{bytes.NewReader(medium)})
		require.NoError(t, err)
		require.Equal(t, int64(len(medium)), source.size)
		require.Equal(t, medium, rewound(t, source.body))
	})

	t.Run("stream of exactly the peek size is still one body of a known length", func(t *testing.T) {
		t.Parallel()

		exact := bytes.Repeat([]byte{5}, int(uploadPeekSize))

		source, err := newUploadSource(plainReader{bytes.NewReader(exact)})
		require.NoError(t, err)
		require.Equal(t, uploadPeekSize, source.size)
		require.Equal(t, exact, rewound(t, source.body))
	})

	t.Run("long stream keeps an unknown length and all its bytes", func(t *testing.T) {
		t.Parallel()

		long := bytes.Repeat([]byte{7}, int(uploadPeekSize)+1)

		source, err := newUploadSource(plainReader{bytes.NewReader(long)})
		require.NoError(t, err)
		require.Equal(t, int64(-1), source.size)
		require.True(t, source.inOrder)
		require.Equal(t, long, readBody(t, source.body))
	})

	t.Run("read error", func(t *testing.T) {
		t.Parallel()

		_, err := newUploadSource(failingReader{})
		require.ErrorIs(t, err, errReadFailed)
	})

	for name, size := range map[string]int{
		"cut stream within the small read ahead": readAheadSmallSize / 2,
		"cut stream past the small read ahead":   2 * readAheadSmallSize,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := newUploadSource(&cutReader{data: bytes.Repeat([]byte{1}, size)})
			require.ErrorIs(t, err, io.ErrUnexpectedEOF, "a cut body must not pass for the end of the stream")
		})
	}
}

func TestUploadSourceCheck(t *testing.T) {
	t.Parallel()

	known := uploadSource{body: bytes.NewReader([]byte("abc")), size: 3}
	require.NoError(t, known.check(3))
	require.ErrorIs(t, known.check(2), ErrUploadSizeMismatch)

	drained := uploadSource{body: bytes.NewReader(nil), size: -1}
	require.NoError(t, drained.check(100))

	cut := uploadSource{body: bytes.NewReader([]byte("rest")), size: -1}
	require.ErrorIs(t, cut.check(100), ErrUploadStreamTooLong)
}

func TestUploadPartSize(t *testing.T) {
	t.Parallel()

	require.Equal(t, DefaultUploadPartSize, uploadPartSize(-1), "unknown length")
	require.Equal(t, DefaultUploadPartSize, uploadPartSize(10), "small object")
	require.Equal(t, DefaultUploadPartSize, uploadPartSize(DefaultUploadPartSize*maxUploadParts), "fits the parts")

	const huge int64 = DefaultUploadPartSize*maxUploadParts + 1

	partSize := uploadPartSize(huge)
	require.Greater(t, partSize, DefaultUploadPartSize)
	require.LessOrEqual(t, huge, partSize*maxUploadParts, "the huge object fits the parts")
}

func TestUploadThreads(t *testing.T) {
	t.Parallel()

	require.Equal(t, uint(DefaultUploadConcurrency), uploadThreads(-1, DefaultUploadPartSize), "unknown length")
	require.Equal(t, uint(1), uploadThreads(0, DefaultUploadPartSize), "empty object")
	require.Equal(t, uint(1), uploadThreads(10, DefaultUploadPartSize), "one part")
	require.Equal(t, uint(2), uploadThreads(DefaultUploadPartSize+1, DefaultUploadPartSize), "two parts")
	require.Equal(t, uint(DefaultUploadConcurrency), uploadThreads(1<<30, DefaultUploadPartSize), "many parts")
}

type s3Request struct {
	method        string
	query         string
	contentLength int64
	body          []byte
}

type multipartServer struct {
	*httptest.Server

	mutex    sync.Mutex
	requests []s3Request

	// holdParts keeps every part request until a second one is in flight or holdPartsTimeout passes;
	// the parts are held before their bodies are read, as the requests of parallel parts all come in
	// at once while a sequential upload sends the next part only after the answer to this one
	holdParts   bool
	inFlight    atomic.Int64
	maxInFlight atomic.Int64

	deletes atomic.Int64
}

const holdPartsTimeout = 30 * time.Second

// newMultipartServer is a minimal s3 that takes single PUTs and multipart uploads and records them
func newMultipartServer(t *testing.T, holdParts bool) *multipartServer {
	t.Helper()

	server := &multipartServer{holdParts: holdParts}

	server.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		if r.Method == http.MethodPut && query.Has("partNumber") {
			server.holdPart()
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			// a failing upload cancels its parts in flight, their bodies end early
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		server.mutex.Lock()
		server.requests = append(server.requests, s3Request{method: r.Method, query: r.URL.RawQuery, contentLength: r.ContentLength, body: body})
		server.mutex.Unlock()

		w.Header().Set("ETag", `"`+strconv.Itoa(len(body))+`"`)

		switch {
		case r.Method == http.MethodPost && query.Has("uploads"):
			w.Header().Set("Content-Type", "application/xml")

			if _, err := io.WriteString(w, `<InitiateMultipartUploadResult><Bucket>bucket</Bucket><Key>key</Key><UploadId>upload-1</UploadId></InitiateMultipartUploadResult>`); err != nil {
				t.Errorf("write initiate response: %v", err)
			}
		case r.Method == http.MethodPost && query.Has("uploadId"):
			w.Header().Set("Content-Type", "application/xml")

			if _, err := io.WriteString(w, `<CompleteMultipartUploadResult><Bucket>bucket</Bucket><Key>key</Key><ETag>"done"</ETag></CompleteMultipartUploadResult>`); err != nil {
				t.Errorf("write complete response: %v", err)
			}
		case r.Method == http.MethodPut:
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodDelete:
			server.deletes.Add(1)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotImplemented)
		}
	}))
	t.Cleanup(server.Close)

	return server
}

func (s *multipartServer) holdPart() {
	inFlight := s.inFlight.Add(1)
	defer s.inFlight.Add(-1)

	for {
		current := s.maxInFlight.Load()
		if inFlight <= current || s.maxInFlight.CompareAndSwap(current, inFlight) {
			break
		}
	}

	if !s.holdParts {
		return
	}

	deadline := time.Now().Add(holdPartsTimeout)
	for s.maxInFlight.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
}

func (s *multipartServer) recorded() []s3Request {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return append([]s3Request(nil), s.requests...)
}

// uploaded puts a multipart upload together by the part numbers, parts may come in any order
func (s *multipartServer) uploaded(t *testing.T) (int, int, []byte) {
	t.Helper()

	var created, completed int

	parts := map[int][]byte{}

	for _, request := range s.recorded() {
		switch {
		case request.method == http.MethodPost && strings.Contains(request.query, "uploads"):
			created++
		case request.method == http.MethodPost:
			completed++
		case request.method == http.MethodPut && strings.Contains(request.query, "partNumber"):
			values, err := url.ParseQuery(request.query)
			require.NoError(t, err)

			number, err := strconv.Atoi(values.Get("partNumber"))
			require.NoError(t, err)

			parts[number] = request.body
		}
	}

	var joined []byte
	for number := 1; number <= len(parts); number++ {
		joined = append(joined, parts[number]...)
	}

	return created, completed, joined
}

func newTestClient(t *testing.T, url string) Client {
	t.Helper()

	client := NewClient("upload-test", url, "us-east-1", "", "", "bucket", "base", "", true)
	t.Cleanup(client.Close)

	return client
}

func TestUpload_RequestsByLength(t *testing.T) {
	t.Parallel()

	t.Run("short stream goes as one PUT", func(t *testing.T) {
		t.Parallel()

		server := newMultipartServer(t, false)

		data := bytes.Repeat([]byte("json"), 1000)
		require.NoError(t, newTestClient(t, server.URL).Upload(context.Background(), "id/meta.json", plainReader{bytes.NewReader(data)}))

		got := server.recorded()
		require.Len(t, got, 1)
		require.Equal(t, http.MethodPut, got[0].method)
		require.Empty(t, got[0].query, "a single PUT carries no multipart query")
		require.Equal(t, int64(len(data)), got[0].contentLength)
		require.Equal(t, data, got[0].body)
	})

	t.Run("long stream goes as a multipart upload", func(t *testing.T) {
		t.Parallel()

		server := newMultipartServer(t, false)

		data := bytes.Repeat([]byte{3}, int(uploadPeekSize)+1024)
		require.NoError(t, newTestClient(t, server.URL).Upload(context.Background(), "id/big.bin", plainReader{bytes.NewReader(data)}))

		created, completed, joined := server.uploaded(t)
		require.Equal(t, 1, created)
		require.Equal(t, 1, completed)
		require.Equal(t, data, joined)
	})
}

func TestUpload_PartsGoInParallel(t *testing.T) {
	t.Parallel()

	data := bytes.Repeat([]byte{4}, int(2*DefaultUploadPartSize)+100)

	for name, reader := range map[string]func() io.Reader{
		"stream read in order":   func() io.Reader { return plainReader{bytes.NewReader(data)} },
		"reader of a known size": func() io.Reader { return bytes.NewReader(data) },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server := newMultipartServer(t, true)

			require.NoError(t, newTestClient(t, server.URL).Upload(context.Background(), "id/big.bin", reader()))

			_, _, joined := server.uploaded(t)
			require.Equal(t, data, joined)
			require.GreaterOrEqual(t, server.maxInFlight.Load(), int64(2), "parts must be in flight at once")
		})
	}
}

// newObjectServer serves one object and counts the requests for it, and those with a Range
func newObjectServer(t *testing.T, data []byte) (*httptest.Server, *atomic.Int64, *atomic.Int64) {
	t.Helper()

	return newCutObjectServer(t, data, 0)
}

// newCutObjectServer announces the whole object but sends its GET body without the last cut bytes
func newCutObjectServer(t *testing.T, data []byte, cut int) (*httptest.Server, *atomic.Int64, *atomic.Int64) {
	t.Helper()

	var gets, ranged atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			gets.Add(1)
		}

		if r.Header.Get("Range") != "" {
			ranged.Add(1)
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		w.Header().Set("ETag", `"object"`)
		w.Header().Set("Last-Modified", "Wed, 14 Oct 2026 13:45:53 GMT")
		w.WriteHeader(http.StatusOK)

		if r.Method == http.MethodGet {
			// a cut body breaks the response off, that is the point of it
			_, _ = w.Write(data[:len(data)-cut]) //nolint:errcheck // the client side is checked
		}
	}))
	t.Cleanup(server.Close)

	return server, &gets, &ranged
}

func TestUpload_ObjectStreamedFromAnotherClient(t *testing.T) {
	t.Parallel()

	for name, size := range map[string]int{
		"one PUT":         1024*1024 + 3,
		"multipart parts": int(DefaultUploadPartSize) + 5,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			data := bytes.Repeat([]byte("object"), size/6+1)[:size]

			source, gets, ranged := newObjectServer(t, data)
			destination := newMultipartServer(t, false)

			reader, err := newTestClient(t, source.URL).Download(context.Background(), "id/object.bin")
			require.NoError(t, err)
			t.Cleanup(func() { _ = reader.Close() })

			require.NoError(t, newTestClient(t, destination.URL).Upload(context.Background(), "id/object.bin", reader))

			require.Equal(t, int64(1), gets.Load(), "the source is read by one GET")
			require.Zero(t, ranged.Load(), "the source is not read by ranges")

			created, _, joined := destination.uploaded(t)
			if created == 0 {
				got := destination.recorded()
				require.Len(t, got, 1)
				joined = got[0].body
			}

			require.Equal(t, data, joined)
		})
	}
}

func TestUploadSource_DownloadedObject(t *testing.T) {
	t.Parallel()

	data := []byte(r2FakeBody)
	server, _, _ := newObjectServer(t, data)

	reader, err := newTestClient(t, server.URL).Download(context.Background(), "id/id.json")
	require.NoError(t, err)
	t.Cleanup(func() { _ = reader.Close() })

	source, err := newUploadSource(reader)
	require.NoError(t, err)
	require.IsType(t, &minio.Object{}, source.body, "minio-go reads its own object in order, any other ReaderAt by ranges")
	require.Equal(t, int64(len(data)), source.size)
	require.True(t, source.inOrder)
}

// cutOnceReader gives its data, then io.ErrUnexpectedEOF once and io.EOF after it
type cutOnceReader struct {
	data []byte
	cut  bool
}

func (r *cutOnceReader) Read(p []byte) (int, error) {
	if len(r.data) > 0 {
		n := copy(p, r.data)
		r.data = r.data[n:]

		return n, nil
	}

	if !r.cut {
		r.cut = true

		return 0, io.ErrUnexpectedEOF
	}

	return 0, io.EOF
}

func TestUpload_CutStreamIsNotCompleted(t *testing.T) {
	t.Parallel()

	data := bytes.Repeat([]byte{2}, int(uploadPeekSize)+2*1024*1024)

	for name, reader := range map[string]func() io.Reader{
		"cut for good": func() io.Reader { return &cutReader{data: data} },
		"cut once":     func() io.Reader { return &cutOnceReader{data: data} },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server := newMultipartServer(t, false)

			err := newTestClient(t, server.URL).Upload(context.Background(), "id/big.bin", reader())
			require.ErrorIs(t, err, io.ErrUnexpectedEOF)

			_, completed, _ := server.uploaded(t)
			require.Zero(t, completed, "a cut stream must not complete the object")
		})
	}
}

func TestUpload_ObjectShorterThanAnnouncedIsRemoved(t *testing.T) {
	t.Parallel()

	data := bytes.Repeat([]byte{6}, int(2*DefaultUploadPartSize)+10)

	source, _, _ := newCutObjectServer(t, data, 1024*1024)
	destination := newMultipartServer(t, false)

	reader, err := newTestClient(t, source.URL).Download(context.Background(), "id/object.bin")
	require.NoError(t, err)
	t.Cleanup(func() { _ = reader.Close() })

	err = newTestClient(t, destination.URL).Upload(context.Background(), "id/object.bin", reader)
	require.ErrorIs(t, err, ErrUploadSizeMismatch)
	require.Equal(t, int64(1), destination.deletes.Load(), "the short object is removed")
}

func TestUpload_PartlyReadFileInParts(t *testing.T) {
	t.Parallel()

	data := bytes.Repeat([]byte("file"), int(2*DefaultUploadPartSize)/4+1000)
	filename := filepath.Join(t.TempDir(), "big.bin")
	require.NoError(t, os.WriteFile(filename, data, 0o600))

	file, err := os.Open(filename)
	require.NoError(t, err)
	t.Cleanup(func() { _ = file.Close() })

	_, err = file.Seek(1000, io.SeekStart)
	require.NoError(t, err)

	server := newMultipartServer(t, false)
	require.NoError(t, newTestClient(t, server.URL).Upload(context.Background(), "id/big.bin", file))

	created, completed, joined := server.uploaded(t)
	require.Equal(t, 1, created)
	require.Equal(t, 1, completed)
	require.Equal(t, data[1000:], joined)
}
