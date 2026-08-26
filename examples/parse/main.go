// Command parse parses a document into the anydoc document model and prints a
// short summary of its blocks, notes, and embedded assets.
//
// Usage:
//
//	go run ./examples/parse -file report.docx
package main

import (
	"flag"
	"fmt"
	"os"

	anydoc "github.com/nbbug/anydoc-go"
)

func main() {
	file := flag.String("file", "", "path of the document to parse")
	flag.Parse()
	if *file == "" {
		fmt.Fprintln(os.Stderr, "usage: parse -file <document>")
		os.Exit(2)
	}

	data, err := os.ReadFile(*file)
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse:", err)
		os.Exit(1)
	}

	doc, err := anydoc.ToDocument(data, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse:", err)
		os.Exit(1)
	}

	fmt.Printf("blocks: %d, notes: %d, assets: %d\n", len(doc.Blocks), len(doc.Notes), len(doc.Assets))
	for _, block := range doc.Blocks {
		fmt.Printf("  %-12s %s\n", block.Kind, blockSummary(block))
	}
}

// blockSummary returns a one-line description of a block's payload.
func blockSummary(b anydoc.Block) string {
	switch b.Kind {
	case "heading", "paragraph":
		return inlineText(b.Content)
	case "list":
		return fmt.Sprintf("%s, %d items", b.List.Marker, len(b.List.Items))
	case "table":
		rows := len(b.Table.Grid)
		cols := 0
		if rows > 0 {
			cols = len(b.Table.Grid[0])
		}
		return fmt.Sprintf("%dx%d, %s", rows, cols, b.Table.Kind)
	case "code_block":
		return fmt.Sprintf("%d bytes", len(*b.Text))
	case "math":
		return *b.Math
	default:
		return ""
	}
}

// inlineText concatenates the text spans of a slice of inlines.
func inlineText(inlines []anydoc.Inline) string {
	out := ""
	for _, in := range inlines {
		if in.Text != nil {
			out += *in.Text
		}
	}
	return out
}
