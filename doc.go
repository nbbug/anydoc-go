// Package anydoc converts documents (Word, PowerPoint, Excel, OpenDocument,
// RTF, EPUB, CSV, and PDF) to GitHub-Flavored Markdown, with full access to
// the parsed document model and embedded assets.
//
// This is the Go binding. It links a prebuilt Rust static (or shared) library
// through cgo; the libraries are downloaded per platform from the GitHub
// release, so users never need a Rust toolchain. See the README for the
// platform matrix, download links, and build requirements.
package anydoc
