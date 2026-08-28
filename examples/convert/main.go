// Command convert converts a document file to GitHub-Flavored Markdown and
// prints it to stdout. The format is detected from the file content, with the
// extension as the fallback for signature-less formats (CSV).
//
// Usage:
//
//	go run ./examples/convert -file report.docx
//	go run ./examples/convert -file data.csv > data.md
//	go run ./examples/convert -file scan.pdf -ocr hosted
package main

import (
	"flag"
	"fmt"
	"os"

	anydoc "github.com/nbbug/anydoc-go"
)

func main() {
	file := flag.String("file", "", "path of the document to convert")
	ocr := flag.String("ocr", "", `what happens to a PDF whose pages need OCR: "" (reject) or "hosted"`)
	flag.Parse()
	if *file == "" {
		fmt.Fprintln(os.Stderr, "usage: convert -file <document> [-ocr hosted]")
		os.Exit(2)
	}

	var opts *anydoc.Options
	switch *ocr {
	case "":
	case "hosted":
		opts = &anydoc.Options{Ocr: anydoc.OcrHosted}
	default:
		fmt.Fprintln(os.Stderr, `convert: -ocr must be "" or "hosted"`)
		os.Exit(2)
	}

	markdown, err := anydoc.ToMarkdownWithOptions(*file, opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "convert:", err)
		os.Exit(1)
	}
	fmt.Print(markdown)
}
