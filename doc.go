// Package anydoc converts documents (Word, PowerPoint, Excel, OpenDocument,
// RTF, EPUB, CSV, and PDF) to GitHub-Flavored Markdown, with full access to
// the parsed document model and embedded assets.
//
// This is the Go binding. It links a Rust static library through cgo; the
// module packages prebuilt archives for every supported platform, so users
// never need a Rust toolchain. See the README for the platform matrix and
// build requirements.
package anydoc
