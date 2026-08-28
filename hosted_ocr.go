package anydoc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"strings"
	"time"
)

// The Firecrawl Parse fallback for OcrHosted. It mirrors the Node and Python
// bindings: on a needs_ocr failure the whole document is sent to Parse (Parse
// has no page selection), keyless unless a key is given, and any failure
// reports the `hosted` error kind.

const (
	firecrawlDefaultAPIURL = "https://api.firecrawl.dev"
	firecrawlTimeout       = 300 * time.Second
)

// parseRequestOptions is the JSON `options` field of the Parse request.
type parseRequestOptions struct {
	Parsers []parseParser `json:"parsers"`
	Origin  string        `json:"origin"`
}

type parseParser struct {
	Type string `json:"type"`
	Mode string `json:"mode"`
}

// sendsToHosted reports whether err is the needs_ocr failure the hosted OCR
// fallback exists for.
func sendsToHosted(err error, opts *Options) bool {
	if err == nil || opts == nil || opts.Ocr != OcrHosted {
		return false
	}
	var ce *ConvertError
	return errors.As(err, &ce) && ce.Kind == "needs_ocr"
}

// parseHosted sends pdf to Firecrawl Parse and returns the extracted
// Markdown. Failures report Kind "hosted", matching the Node binding's
// `hosted` error code and the Python HostedError.
func parseHosted(pdf []byte, filename string, opts *Options) (string, error) {
	apiKey := opts.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("FIRECRAWL_API_KEY")
	}
	apiURL := opts.APIURL
	if apiURL == "" {
		apiURL = os.Getenv("FIRECRAWL_API_URL")
	}
	if apiURL == "" {
		apiURL = firecrawlDefaultAPIURL
	}

	parseOptions, err := json.Marshal(parseRequestOptions{
		Parsers: []parseParser{{Type: "pdf", Mode: "auto"}},
		Origin:  fmt.Sprintf("anydoc@%s", Version),
	})
	if err != nil {
		return "", hostedError(fmt.Sprintf("Firecrawl Parse: %v", err))
	}

	body, contentType, err := multipartBody(parseOptions, filename, pdf)
	if err != nil {
		return "", hostedError(fmt.Sprintf("Firecrawl Parse: %v", err))
	}

	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(apiURL, "/")+"/v2/parse", body)
	if err != nil {
		return "", hostedError(fmt.Sprintf("Firecrawl Parse: %v", err))
	}
	req.Header.Set("Content-Type", contentType)
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := (&http.Client{Timeout: firecrawlTimeout}).Do(req)
	if err != nil {
		return "", hostedError(fmt.Sprintf("Firecrawl Parse: %v", err))
	}
	defer resp.Body.Close()
	replyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", hostedError(fmt.Sprintf("Firecrawl Parse: %v", err))
	}

	var reply struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
		Data    *struct {
			Markdown string `json:"markdown"`
		} `json:"data"`
	}
	// An unparseable body reads as {success:false} and the detail falls back
	// to the HTTP status, like the Node binding's `?? response.statusText`.
	_ = json.Unmarshal(replyBytes, &reply)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || !reply.Success {
		detail := reply.Error
		if detail == "" {
			detail = resp.Status
		}
		return "", hostedError(describeHostedFailure(resp.StatusCode, detail, apiKey != ""))
	}

	markdown := ""
	if reply.Data != nil {
		markdown = reply.Data.Markdown
	}
	if markdown == "" {
		return "", hostedError("Firecrawl Parse returned no Markdown")
	}
	if !strings.HasSuffix(markdown, "\n") {
		markdown += "\n"
	}
	return markdown, nil
}

// multipartBody builds the Parse /v2/parse request body: the `options` JSON
// field followed by the PDF file part.
func multipartBody(parseOptions []byte, filename string, pdf []byte) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if err := mw.WriteField("options", string(parseOptions)); err != nil {
		return nil, "", err
	}
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, sanitizeFilename(filename)))
	header.Set("Content-Type", "application/pdf")
	part, err := mw.CreatePart(header)
	if err != nil {
		return nil, "", err
	}
	if _, err := part.Write(pdf); err != nil {
		return nil, "", err
	}
	if err := mw.Close(); err != nil {
		return nil, "", err
	}
	return &buf, mw.FormDataContentType(), nil
}

// sanitizeFilename keeps the multipart header well-formed, like the Python
// binding.
func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, `"`, "_")
	name = strings.ReplaceAll(name, "\r", "_")
	name = strings.ReplaceAll(name, "\n", "_")
	return name
}

// describeHostedFailure maps a Parse failure to a message, matching the Node
// and Python bindings.
func describeHostedFailure(status int, detail string, keyed bool) string {
	switch status {
	case http.StatusUnauthorized:
		return fmt.Sprintf("Firecrawl Parse rejected the API key: %s", detail)
	case http.StatusPaymentRequired:
		return fmt.Sprintf("Firecrawl Parse is out of credits: %s", detail)
	case http.StatusTooManyRequests:
		if keyed {
			return fmt.Sprintf("Firecrawl Parse rate limit reached: %s", detail)
		}
		return fmt.Sprintf("Firecrawl Parse keyless limit reached, set FIRECRAWL_API_KEY: %s", detail)
	default:
		return fmt.Sprintf("Firecrawl Parse: %s", detail)
	}
}

func hostedError(detail string) error {
	return &ConvertError{Kind: "hosted", Detail: detail}
}
