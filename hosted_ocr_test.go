package anydoc

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newParseServer runs a fake Firecrawl Parse endpoint and returns Options
// pointing at it.
func newParseServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Options) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv, &Options{Ocr: OcrHosted, APIURL: srv.URL}
}

func TestParseHostedSuccess(t *testing.T) {
	_, opts := newParseServer(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"success":true,"data":{"markdown":"# Scanned"}}`)
	})
	md, err := parseHosted([]byte("PDF bytes"), "scan.pdf", opts)
	if err != nil {
		t.Fatalf("parseHosted: %v", err)
	}
	if want := "# Scanned\n"; md != want {
		t.Errorf("markdown = %q, want %q (trailing newline added)", md, want)
	}
}

func TestParseHostedRequestShape(t *testing.T) {
	var got struct {
		options    map[string]any
		filename   string
		fileType   string
		file       string
		authHeader string
	}
	_, opts := newParseServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("ParseMultipartForm: %v", err)
			return
		}
		if err := json.Unmarshal([]byte(r.FormValue("options")), &got.options); err != nil {
			t.Errorf("options field is not JSON: %v", err)
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Errorf("FormFile: %v", err)
			return
		}
		data, err := io.ReadAll(file)
		if err != nil {
			t.Errorf("read file part: %v", err)
		}
		got.filename = header.Filename
		got.fileType = header.Header.Get("Content-Type")
		got.file = string(data)
		got.authHeader = r.Header.Get("Authorization")
		io.WriteString(w, `{"success":true,"data":{"markdown":"# Scanned\n"}}`)
	})
	opts.APIKey = "sk-test"
	md, err := parseHosted([]byte("PDF bytes"), "scan.pdf", opts)
	if err != nil {
		t.Fatalf("parseHosted: %v", err)
	}
	if md != "# Scanned\n" {
		t.Errorf("markdown = %q", md)
	}
	parsers, _ := got.options["parsers"].([]any)
	if len(parsers) != 1 {
		t.Fatalf("parsers = %v, want one entry", got.options["parsers"])
	}
	parser, _ := parsers[0].(map[string]any)
	if parser["type"] != "pdf" || parser["mode"] != "auto" {
		t.Errorf("parser = %v, want {type:pdf mode:auto}", parser)
	}
	if got.options["origin"] != "anydoc@"+Version {
		t.Errorf("origin = %v, want anydoc@%s", got.options["origin"], Version)
	}
	if got.filename != "scan.pdf" {
		t.Errorf("filename = %q, want scan.pdf", got.filename)
	}
	if got.fileType != "application/pdf" {
		t.Errorf("file content type = %q", got.fileType)
	}
	if got.file != "PDF bytes" {
		t.Errorf("file bytes = %q", got.file)
	}
	if got.authHeader != "Bearer sk-test" {
		t.Errorf("Authorization = %q, want Bearer sk-test", got.authHeader)
	}
}

func TestParseHostedFailures(t *testing.T) {
	cases := []struct {
		name    string
		status  int
		reply   string
		keyed   bool
		wantSub string
	}{
		{
			name:    "unauthorized",
			status:  http.StatusUnauthorized,
			reply:   `{"success":false,"error":"bad key"}`,
			keyed:   true,
			wantSub: "Firecrawl Parse rejected the API key: bad key",
		},
		{
			name:    "out of credits",
			status:  http.StatusPaymentRequired,
			reply:   `{"success":false,"error":"no credits"}`,
			wantSub: "Firecrawl Parse is out of credits: no credits",
		},
		{
			name:    "keyless rate limit",
			status:  http.StatusTooManyRequests,
			reply:   `{"success":false,"error":"slow down"}`,
			wantSub: "keyless limit reached, set FIRECRAWL_API_KEY: slow down",
		},
		{
			name:    "keyed rate limit",
			status:  http.StatusTooManyRequests,
			reply:   `{"success":false,"error":"slow down"}`,
			keyed:   true,
			wantSub: "rate limit reached: slow down",
		},
		{
			name:    "generic failure",
			status:  http.StatusInternalServerError,
			reply:   `{"success":false,"error":"boom"}`,
			wantSub: "Firecrawl Parse: boom",
		},
		{
			name:    "failed despite 200",
			status:  http.StatusOK,
			reply:   `{"success":false,"error":"no"}`,
			wantSub: "Firecrawl Parse: no",
		},
		{
			name:    "unparseable body falls back to status",
			status:  http.StatusBadGateway,
			reply:   `not json`,
			wantSub: "Firecrawl Parse: 502",
		},
		{
			name:    "no markdown",
			status:  http.StatusOK,
			reply:   `{"success":true,"data":{}}`,
			wantSub: "Firecrawl Parse returned no Markdown",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, opts := newParseServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				io.WriteString(w, tc.reply)
			})
			if tc.keyed {
				opts.APIKey = "sk-test"
			}
			md, err := parseHosted([]byte("PDF bytes"), "scan.pdf", opts)
			if md != "" {
				t.Errorf("markdown = %q, want empty on failure", md)
			}
			var ce *ConvertError
			if !errors.As(err, &ce) {
				t.Fatalf("err = %v, want *ConvertError", err)
			}
			if ce.Kind != "hosted" {
				t.Errorf("Kind = %q, want hosted", ce.Kind)
			}
			if !strings.Contains(ce.Detail, tc.wantSub) {
				t.Errorf("Detail = %q, want substring %q", ce.Detail, tc.wantSub)
			}
		})
	}
}

func TestParseHostedNetworkError(t *testing.T) {
	srv, opts := newParseServer(t, func(w http.ResponseWriter, r *http.Request) {})
	srv.Close()
	md, err := parseHosted([]byte("PDF bytes"), "scan.pdf", opts)
	if md != "" {
		t.Errorf("markdown = %q, want empty", md)
	}
	var ce *ConvertError
	if !errors.As(err, &ce) || ce.Kind != "hosted" {
		t.Fatalf("err = %v, want hosted ConvertError", err)
	}
	if !strings.HasPrefix(ce.Detail, "Firecrawl Parse: ") {
		t.Errorf("Detail = %q, want Firecrawl Parse: prefix", ce.Detail)
	}
}

func TestParseHostedEnvironment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    map[string]any{"markdown": "auth:" + auth},
		})
	}))
	t.Cleanup(srv.Close)

	t.Setenv("FIRECRAWL_API_KEY", "sk-env")
	t.Setenv("FIRECRAWL_API_URL", srv.URL)

	// Empty Options fields fall back to the environment variables.
	md, err := parseHosted([]byte("PDF bytes"), "scan.pdf", &Options{})
	if err != nil {
		t.Fatalf("parseHosted: %v", err)
	}
	if !strings.Contains(md, "Bearer sk-env") {
		t.Errorf("markdown = %q, want it to echo the env API key", md)
	}

	// Explicit Options fields win over the environment.
	md, err = parseHosted([]byte("PDF bytes"), "scan.pdf", &Options{APIKey: "sk-explicit", APIURL: srv.URL})
	if err != nil {
		t.Fatalf("parseHosted: %v", err)
	}
	if !strings.Contains(md, "Bearer sk-explicit") {
		t.Errorf("markdown = %q, want it to echo the explicit API key", md)
	}
}

func TestWantsOcrFallback(t *testing.T) {
	needsOcr := &ConvertError{Kind: "needs_ocr", Detail: "pages 1 of 2 need OCR"}
	handler := OcrHandlerFunc(func(pdf []byte) (string, error) { return "# OCR\n", nil })
	cases := []struct {
		name string
		err  error
		opts *Options
		want bool
	}{
		{"needs_ocr with hosted", needsOcr, &Options{Ocr: OcrHosted}, true},
		{"needs_ocr with custom handler", needsOcr, &Options{Ocr: OcrCustom, OcrHandler: handler}, true},
		{"needs_ocr with reject", needsOcr, &Options{Ocr: OcrReject}, false},
		{"needs_ocr with nil options", needsOcr, nil, false},
		{"needs_ocr with zero options", needsOcr, &Options{}, false},
		{"other error with hosted", &ConvertError{Kind: "malformed"}, &Options{Ocr: OcrHosted}, false},
		{"plain error with hosted", errors.New("boom"), &Options{Ocr: OcrHosted}, false},
		{"no error with hosted", nil, &Options{Ocr: OcrHosted}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := wantsOcrFallback(tc.err, tc.opts); got != tc.want {
				t.Errorf("wantsOcrFallback = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestOcrFallbackCustom(t *testing.T) {
	needsOcr := &ConvertError{Kind: "needs_ocr", Detail: "pages 1 of 2 need OCR"}
	var received []byte
	md, err := ocrFallback(needsOcr, []byte("PDF bytes"), "scan.pdf", &Options{
		Ocr: OcrCustom,
		OcrHandler: OcrHandlerFunc(func(pdf []byte) (string, error) {
			received = pdf
			return "# OCR", nil
		}),
	})
	if err != nil {
		t.Fatalf("ocrFallback: %v", err)
	}
	if md != "# OCR" {
		t.Errorf("markdown = %q, want the handler's output", md)
	}
	if string(received) != "PDF bytes" {
		t.Errorf("handler received %q, want the whole PDF", received)
	}
}

func TestOcrFallbackCustomErrorPassesThrough(t *testing.T) {
	sentinel := errors.New("onnx session failed")
	md, err := ocrFallback(&ConvertError{Kind: "needs_ocr"}, []byte("PDF bytes"), "scan.pdf", &Options{
		Ocr:        OcrCustom,
		OcrHandler: OcrHandlerFunc(func(pdf []byte) (string, error) { return "", sentinel }),
	})
	if md != "" {
		t.Errorf("markdown = %q, want empty on failure", md)
	}
	if err != sentinel {
		t.Errorf("err = %v, want the handler's own error unchanged", err)
	}
}

func TestOcrFallbackWithoutFallback(t *testing.T) {
	needsOcr := &ConvertError{Kind: "needs_ocr", Detail: "pages 1 of 2 need OCR"}
	cases := []struct {
		name string
		opts *Options
	}{
		{"nil options", nil},
		{"zero options", &Options{}},
		{"reject", &Options{Ocr: OcrReject}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			md, err := ocrFallback(needsOcr, []byte("PDF bytes"), "scan.pdf", tc.opts)
			if md != "" {
				t.Errorf("markdown = %q, want empty", md)
			}
			if err != needsOcr {
				t.Errorf("err = %v, want the original needs_ocr error", err)
			}
		})
	}
}

func TestOcrFallbackMisconfigured(t *testing.T) {
	cases := []struct {
		name    string
		opts    *Options
		wantSub string
	}{
		{"custom without handler", &Options{Ocr: OcrCustom}, "OcrHandler"},
		{"unknown mode", &Options{Ocr: "clown"}, "unknown Ocr mode"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			md, err := ocrFallback(&ConvertError{Kind: "needs_ocr"}, []byte("PDF bytes"), "scan.pdf", tc.opts)
			if md != "" {
				t.Errorf("markdown = %q, want empty", md)
			}
			var ce *ConvertError
			if !errors.As(err, &ce) {
				t.Fatalf("err = %v, want *ConvertError", err)
			}
			if ce.Kind != "unsupported" {
				t.Errorf("Kind = %q, want unsupported", ce.Kind)
			}
			if !strings.Contains(ce.Detail, tc.wantSub) {
				t.Errorf("Detail = %q, want substring %q", ce.Detail, tc.wantSub)
			}
		})
	}
}

func TestOcrFallbackHostedDispatch(t *testing.T) {
	_, opts := newParseServer(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"success":true,"data":{"markdown":"# Hosted\n"}}`)
	})
	md, err := ocrFallback(&ConvertError{Kind: "needs_ocr"}, []byte("PDF bytes"), "scan.pdf", opts)
	if err != nil {
		t.Fatalf("ocrFallback: %v", err)
	}
	if md != "# Hosted\n" {
		t.Errorf("markdown = %q, want the hosted response", md)
	}
}

func TestOcrHandlerFuncAdapter(t *testing.T) {
	handler := OcrHandlerFunc(func(pdf []byte) (string, error) { return string(pdf), nil })
	md, err := handler.OcrMarkdown([]byte("hello"))
	if err != nil {
		t.Fatalf("OcrMarkdown: %v", err)
	}
	if md != "hello" {
		t.Errorf("markdown = %q, want the adapted output", md)
	}
}
