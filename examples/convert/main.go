// Command convert converts a document file to GitHub-Flavored Markdown and
// prints it to stdout. The format is detected from the file content, with the
// extension as the fallback for signature-less formats (CSV).
//
// Usage:
//
//	go run ./examples/convert -file report.docx
//	go run ./examples/convert -file data.csv > data.md
package main

import (
	"flag"
	"fmt"
	"os"

	anydoc "github.com/your-org/anydoc-go"
)

func main() {
	file := flag.String("file", "", "path of the document to convert")
	flag.Parse()
	if *file == "" {
		fmt.Fprintln(os.Stderr, "usage: convert -file <document>")
		os.Exit(2)
	}

	markdown, err := anydoc.ToMarkdown(*file)
	if err != nil {
		fmt.Fprintln(os.Stderr, "convert:", err)
		os.Exit(1)
	}
	fmt.Print(markdown)
}
