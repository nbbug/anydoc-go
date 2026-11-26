// Package anydoc converts documents (Word, PowerPoint, Excel, OpenDocument,
// RTF, EPUB, CSV, and PDF) to GitHub-Flavored Markdown, with full access to
// the parsed document model and embedded assets.
//
// This is the Go binding. It links the prebuilt Rust shared library through
// cgo; the library is downloaded per platform from the GitHub release, so users
// never need a Rust toolchain. The library is loaded at run time, so it has to
// ship with the program. See the README for the platform matrix, download
// links, and build requirements.
package anydoc
