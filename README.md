**English · [中文文档](README.zh-CN.md)**

# anydoc-go

Go bindings for [anydoc](https://github.com/firecrawl/anydoc) — the Rust
library that converts Word, PowerPoint, Excel, OpenDocument, RTF, EPUB, CSV,
and PDF documents to GitHub-Flavored Markdown.

anydoc-go links anydoc as a **static library through cgo**, with prebuilt
archives packaged in the module for every supported platform. Users of this
module never install Rust or any other toolchain: `go get` downloads the
archive and `go build` links it.

## Requirements

|             |                                                                                       |
| ----------- | ------------------------------------------------------------------------------------- |
| Go          | ≥ 1.22                                                                                |
| CGO         | `CGO_ENABLED=1` (the Go default on all supported platforms)                           |
| C toolchain | the standard one for your OS — Xcode CLT on macOS, gcc on Linux, mingw-w64 on Windows |
| Rust        | **not required** to use the module; only to rebuild the archives                      |

## Install

```
go get github.com/nbbug/anydoc-go
```

## Quickstart

```go
package main

import (
	"fmt"
	"os"

	anydoc "github.com/nbbug/anydoc-go"
)

func main() {
	// Convert a file: the format is detected from the content.
	markdown, err := anydoc.ToMarkdown("report.docx")
	if err != nil {
		panic(err)
	}
	fmt.Print(markdown)

	// Convert bytes that are already in memory. Signature-less formats
	// like CSV must be named explicitly.
	data, err := os.ReadFile("data.csv")
	if err != nil {
		panic(err)
	}
	md, err := anydoc.ToMarkdownBytes(data, &anydoc.FormatCsv)
	// ...
	_ = md
}
```

Runnable examples live in [examples/](examples/):

```
go run ./examples/convert -file report.docx
go run ./examples/parse   -file report.docx
```

## Supported platforms

The module packages a prebuilt archive for each combination below; the Go
compiler selects the right one automatically with build tags. Each archive is
a single file under `lib/<platform>/`: `libanydoc_go.a` everywhere except
Windows, which ships the MSVC COFF archive `anydoc_go.lib`.

| Platform      | Rust target                 | C toolchain used for linking |
| ------------- | --------------------------- | ---------------------------- |
| linux/amd64   | `x86_64-unknown-linux-gnu`  | gcc                          |
| linux/arm64   | `aarch64-unknown-linux-gnu` | gcc (aarch64)                |
| darwin/amd64  | `x86_64-apple-darwin`       | clang (Xcode CLT)            |
| darwin/arm64  | `aarch64-apple-darwin`      | clang (Xcode CLT)            |
| windows/amd64 | `x86_64-pc-windows-msvc`    | mingw-w64 gcc                |
| windows/arm64 | `aarch64-pc-windows-msvc`   | mingw-w64 gcc                |

Notes:

- **Windows uses MSVC targets** (`x86_64-pc-windows-msvc` and
  `aarch64-pc-windows-msvc`), matching the WeKnora binding. The archives are
  built with the MSVC toolchain (which cannot be redistributed, so only a
  native Windows host or the CI Windows runner can build them) and produce
  `anydoc_go.lib`. Go's cgo still links with mingw gcc — GNU ld reads MSVC
  `.lib` archives — so users need the standard mingw-w64 toolchain, not
  Visual Studio.
- **Linux is glibc.** Alpine/musl users must rebuild the archive
  (`./scripts/build-all.sh linux_amd64` inside an Alpine image or with
  `cargo-zigbuild`, then commit it). musl cannot be auto-detected by Go build
  tags, so the reference WeKnora binding ships a parallel `-tags musl` variant;
  this module keeps the matrix minimal and can adopt the same pattern if
  Alpine support is needed.
- Building on any other platform (or with `CGO_ENABLED=0`) still compiles:
  see [Build tags and the stub](#build-tags-and-the-stub).

## API

```go
// Convert a document file to Markdown. The format is detected from the
// content; the extension is the fallback for signature-less formats (CSV).
func ToMarkdown(path string) (string, error)

// Convert an in-memory document to Markdown. Pass a Format to select the
// parser, or nil to detect it from the content.
func ToMarkdownBytes(data []byte, format *Format) (string, error)

// Convert with extended behavior. A nil Options behaves like ToMarkdown /
// ToMarkdownBytes; with Ocr = OcrHosted, a PDF whose pages need OCR is sent
// to Firecrawl Parse instead of failing with needs_ocr.
func ToMarkdownWithOptions(path string, opts *Options) (string, error)
func ToMarkdownBytesWithOptions(data []byte, format *Format, opts *Options) (string, error)

// Options: Ocr (OcrReject, the default / OcrHosted / OcrCustom with
// OcrHandler), APIKey (falls back to FIRECRAWL_API_KEY, then keyless),
// APIURL (falls back to FIRECRAWL_API_URL, then https://api.firecrawl.dev).

// Convert with embedded images rewritten as ![alt](images/image-N.ext) so
// they keep their original positions. PDF has no document model and is
// converted the same way as ToMarkdownBytes.
func ToMarkdownWithAssetLinks(data []byte, format *Format) (string, error)

// Parse an in-memory document into the document model, which also carries
// the embedded assets. Unsupported for PDF (no model form) — use
// ToMarkdownBytes for PDFs.
func ToDocument(data []byte, format *Format) (*Document, error)

// Format detection.
func FormatFromBytes(data []byte) (Format, bool)
func FormatFromExtension(ext string) (Format, bool)
func FormatFromPath(path string) (Format, bool)

// The bytes of the static archive packaged for this platform, and a helper
// that writes them to disk. See "Embedding the archive" below.
func EmbeddedLib() []byte
func ExtractLib(dir string) (string, error)

// Module version, kept in lockstep with the pinned anydoc crate.
const Version = "0.2.4"
```

**Formats** ([format.go](format.go)): `FormatDoc`, `FormatDocx`, `FormatOdt`,
`FormatPdf`, `FormatPpt`, `FormatPptx`, `FormatRtf`, `FormatEpub`,
`FormatXlsx`, `FormatOds`, `FormatOdp`, `FormatCsv`.

**Errors** ([errors.go](errors.go)): conversions return `*ConvertError` with a
`Kind` matching the Node/Python bindings — `unsupported`, `malformed`,
`encrypted`, `resource_limit`, `missing_part`, `io`, `pdf_no_model`,
`needs_ocr` — plus a human-readable `Detail`. `needs_ocr` means some pages of
a PDF are scanned or image-only (since anydoc 0.2.4 they fail the conversion
naming the pages instead of being silently dropped) and also carries the
structured `NeedsOcr{Pages, PageCount}` fields. An invalid explicit `Format`
reports `unknown_format`, and a failed Firecrawl Parse fallback reports
`hosted`.

**Document model** ([model.go](model.go)): `Document` (blocks, notes,
assets), `Block` (`heading`, `paragraph`, `list`, `table`, `block_quote`,
`code_block`, `rule`, `math`), `Inline` (`text`, `link`, `image`, `anchor`,
`note_ref`, `line_break`, `math`, `checkbox`), plus `Style`, `LinkTarget`,
`ImageSource`, `List`, `ListItem`, `Table`, `CellSlot`, `Cell`, `Note`, and
`Asset` — a full, self-contained representation with embedded asset bytes
carried in `Asset.Data`.

### OCR fallbacks (opt-in)

For a PDF whose pages are scanned or image-only, anydoc fails with
`needs_ocr` naming the pages. Two opt-in fallbacks can take over instead.

**`OcrHosted`** sends the whole document to
[Firecrawl Parse](https://firecrawl.dev/parse), which extracts the text:

```go
md, err := anydoc.ToMarkdownBytesWithOptions(pdf, &anydoc.FormatPdf, &anydoc.Options{
	Ocr: anydoc.OcrHosted,
})
```

It works keyless; set `FIRECRAWL_API_KEY` (or `Options.APIKey`) to raise the
rate limits, and `FIRECRAWL_API_URL`/`Options.APIURL` to point at a different
endpoint. Failures report kind `hosted`.

**`OcrCustom`** hands the document to an `OcrHandler` you inject — a local
OCR model (for example ONNX), a local OCR service, or any API of your own:

```go
type onnxOcr struct {
	session *ort.Session // your ONNX inference session
}

func (o *onnxOcr) OcrMarkdown(pdf []byte) (string, error) {
	return runOcr(o.session, pdf) // pages → text → Markdown, all local
}

md, err := anydoc.ToMarkdownBytesWithOptions(pdf, &anydoc.FormatPdf, &anydoc.Options{
	Ocr:        anydoc.OcrCustom,
	OcrHandler: &onnxOcr{session: session},
})

// OcrHandlerFunc adapts a plain function:
//   anydoc.OcrHandlerFunc(func(pdf []byte) (string, error) { ... })
```

The handler receives the whole PDF and returns the extracted Markdown; its
errors pass through unchanged. `OcrCustom` with a nil handler reports an
`unsupported` error.

Only documents anydoc cannot convert itself — PDFs that fail with
`needs_ocr` — ever reach a fallback; everything else stays in the local
conversion (and on the machine, except for `OcrHosted`).

## How it works

1. **Rust C ABI** — this repository is also a Rust crate
   ([Cargo.toml](Cargo.toml)) that wraps the published `anydoc` crate
   (pinned exactly, currently `=0.2.4`) and exposes a small C ABI
   ([src/lib.rs](src/lib.rs)): `anydoc_to_markdown*`, `anydoc_to_document`,
   format detection, and matching `_free` functions. A build script
   re-exports anydoc's crate-private `document_to_markdown` serializer so the
   official GFM rendering stays available.
2. **One static archive per platform** — `crate-type = ["staticlib"]` builds
   `libanydoc_go.a` (~20 MB). The six archives are committed under
   [lib/](lib/).
3. **Platform files with build tags** — one Go file per platform
   ([lib_linux_amd64.go](lib_linux_amd64.go), …) is selected by
   `//go:build linux && amd64 && cgo` and carries the `#cgo LDFLAGS` pointing
   at its archive:
   `-L${SRCDIR}/lib/linux_amd64 -lanydoc_go -lm -lstdc++ -ldl -lpthread`.
4. **`//go:embed`** — each platform file embeds its archive
   (`//go:embed lib/linux_amd64/libanydoc_go.a`). The directive makes the
   archive a _declared dependency of the package_, so `go mod vendor`, module
   zips, and `go list` can never silently drop it. The bytes are exposed by
   `EmbeddedLib()`/`ExtractLib()`; binaries that never call them pay nothing —
   the linker eliminates the unreferenced ~20 MB.
5. **A user-friendly API** — [anydoc.go](anydoc.go) hides every pointer and
   free call behind plain Go functions, pins the goroutine during ABI calls
   (the error slot is thread-local), and bounds-decodes the document buffer
   into Go structs.

## Build tags and the stub

| Build                                                     | Files compiled                             | Behavior                                                      |
| --------------------------------------------------------- | ------------------------------------------ | ------------------------------------------------------------- |
| cgo, supported platform                                   | platform file + [anydoc.go](anydoc.go)     | links the archive, full functionality                         |
| cgo, supported platform, `-tags anydoc_dynamic`                  | dynamic platform file + [anydoc.go](anydoc.go) | links the shared library downloaded from the release       |
| `CGO_ENABLED=0` or unsupported platform                   | [anydoc_stub.go](anydoc_stub.go)           | compiles; every function returns a helpful `UnavailableError` |

```go
md, err := anydoc.ToMarkdownBytes(data, nil)
if err != nil && anydoc.IsUnavailable(err) {
	// fall back to another converter or tell the operator why
	log.Printf("anydoc unavailable: %v", err)
}
```

The stub keeps the module importable everywhere — builds never fail because
of this package; they fail fast at run time with an explanation.

## Vendoring

`go mod vendor` copies the archives and the C header: the `//go:embed`
directives declare them as package dependencies, so vendored builds keep
working with `-mod=vendor` — the `#cgo LDFLAGS` paths are `${SRCDIR}`-relative
and resolve inside the vendor tree. Run `go mod vendor` with the default
cgo-enabled environment: it captures all six `lib/<platform>/` trees (for
every platform, not just the host). Vendoring with `CGO_ENABLED=0` captures
only the header — which is all a stub build needs.

Mind the module size: the six archives total ~120 MB, so a `go get` pulls a
large module zip (well under proxy limits, but worth knowing). If your
registry or bandwidth makes that uncomfortable, vendoring once is the usual
answer — or see the dynamic-library option below.

## Dynamic libraries (opt-in)

By default the module links the embedded static archive. Building with the
`anydoc_dynamic` build tag links a shared library instead:

```
go build -tags anydoc_dynamic ./...
```

Nothing dynamic is embedded in the module — the shared libraries are built
by the **Build dynamic libraries** workflow and published on the
[releases page](https://github.com/nbbug/anydoc-go/releases) as
`anydoc-dynlib-<platform>.tar.gz` (six archives, each laying out
`lib/<platform>/<shared library>` plus the C header). Download the one for
your platform and make it reachable through the cgo search path:

- Extract it into a `lib` directory and pass the platform directory through
  `CGO_LDFLAGS` (cgo appends these flags after the package's own):
  ```
  tar -xzf anydoc-dynlib-darwin_arm64.tar.gz
  CGO_LDFLAGS="-L$(pwd)/lib/darwin_arm64" go build -tags anydoc_dynamic ./...
  ```
- Or vendor the module (`go mod vendor`) and copy the extracted
  `lib/<platform>/` files into
  `vendor/github.com/nbbug/anydoc-go/dynlib/<platform>/`; the dynamic
  platform files already search `${SRCDIR}/dynlib/<platform>`.

At run time the OS must be able to load the library: set
`LD_LIBRARY_PATH` (Linux) or `DYLD_LIBRARY_PATH` (macOS) to the platform
directory, and on Windows place the DLL next to the executable or on `PATH`.

Trade-offs: dynamic builds keep the binary small and let several binaries
share one library, but the deployment must carry the library alongside, and
the binary is only as good as the library you downloaded. Prefer the static
default unless module size is the bottleneck.

## Building the archives from source

You only need this to upgrade anydoc, change the C ABI, or support a new
platform. Users of the module never do.

Requirements: Rust (rustup, any recent stable — anydoc 0.2.4 needs ≥ 1.88),
plus one cross toolchain: `cross` (with Docker) or `cargo-zigbuild`
(bundles zig, no Docker).

```bash
# Everything the current platform can build.
./scripts/build-all.sh

# A subset, by lib directory name.
./scripts/build-all.sh linux_amd64 darwin_arm64

# One target directly, choosing the driver:
TARGET=aarch64-unknown-linux-gnu BUILD_TOOL=cross ./scripts/build-anydoc-lib.sh
TARGET=x86_64-pc-windows-msvc ./scripts/build-anydoc-lib.sh   # on a Windows host only
```

[scripts/build-all.sh](scripts/build-all.sh) prefers `cross`, falls back to
`cargo-zigbuild`, then plain cargo. macOS archives build natively on a Mac
(Apple's SDK cannot be redistributed, so Linux hosts cannot build them — CI
uses macOS runners); the Windows archive builds natively on Windows with the
MSVC toolchain (CI uses a Windows runner) and is skipped elsewhere. The
scripts pin the exact `anydoc` version, refuse to build when
[version.go](version.go) disagrees with the Cargo.toml pin, and build with
`--locked` once [Cargo.lock](Cargo.lock) exists.

### Rebuilding in CI

The **Build static libraries** workflow
([.github/workflows/build-libs.yml](.github/workflows/build-libs.yml)) builds
all six archives, smoke-tests them by linking and converting through the Go
package, then commits and pushes them to the main branch:

**Actions → “Build static libraries” → Run workflow.**

Run it after changing the `anydoc = "=X.Y.Z"` pin, upgrading tooling, or
whenever the committed archives are stale. A separate **CI** workflow runs
gofmt, the version-pin agreement check, stub-mode tests, and — once archives
exist — the full cgo test suite on every push and pull request.

## Wasm vs. this binding

anydoc also ships official [WebAssembly bindings](https://github.com/firecrawl/anydoc/tree/main/wasm)
(`@firecrawl/anydoc-wasm`). This static-library binding exists because a
Go-native server gains nothing from the wasm runtime.

**Advantages over wasm**

- **No Rust or wasm toolchain for users.** Prebuilt archives arrive through
  `go get`; wasm in Go requires [wazero](https://github.com/tetratelabs/wazero)
  or similar, plus a build step to produce the `.wasm`.
- **Native speed.** The conversion runs as machine code in-process, with no
  wasm interpretation or JIT warm-up (typically 1.5–3× faster in practice).
- **Full filesystem API.** `ToMarkdown(path)` exists only here — wasm has no
  filesystem.
- **Concurrency.** Native threads and Go's scheduler; wasm modules run
  single-threaded on the calling thread, so busy servers must shuttle
  conversions to workers.
- **One less dependency.** The archive is linked into the binary; no runtime
  library to keep in lockstep with the Go module.

**Disadvantages vs. wasm**

- **cgo required.** Breaks `CGO_ENABLED=0` builds and needs a C toolchain per
  platform; some PaaS/buildpack setups and static-analysis tooling dislike
  cgo.
- **Per-platform archives.** Six files × ~20 MB in the module; wasm is a
  single portable artifact. The linked binary grows by the archive too.
- **Cross-compiling is more work.** One host cannot produce every archive
  without `cross`/zig (macOS targets need a Mac); wasm builds anywhere from
  anywhere.
- **No browser or edge use.** wasm runs in the browser and on serverless
  edge platforms; a native archive cannot.
- **Libc coupling.** The linux archive targets glibc; wasm is libc-agnostic.

**Rule of thumb:** in a Go service that converts documents server-side, use
this module. For browser/edge tooling, small CLI one-offs, or pure-Go
deployment pipelines, the wasm binding fits better.

## Security notes

anydoc parses untrusted, hostile input in your process:

- The Rust ABI catches panics and reports them as `malformed` errors rather
  than aborting the process ([src/lib.rs](src/lib.rs), `guarded`). This
  cannot contain stack overflows or allocation failures.
- The pinned anydoc version includes the pdf-inspector fixes that bound
  previously unbounded PDF parsing (see the anydoc changelog) — upgrade the
  pin with `--locked` rebuilds rather than ad-hoc versions.
- With `OcrHosted`, PDFs that fail with `needs_ocr` leave your process and
  are uploaded to Firecrawl Parse (keyless unless `FIRECRAWL_API_KEY` is
  set). Review its terms before enabling it on sensitive documents; the
  default never sends anything anywhere.
- Sandbox or resource-limit processes that convert untrusted uploads, as you
  would for any native parser.

## License

MIT. This module links the `anydoc` crate, MIT © Sideguide Technologies Inc.
See [LICENSE](LICENSE).

## Acknowledgments

The binding architecture follows the upstream Go bindings
([firecrawl/anydoc#30](https://github.com/firecrawl/anydoc/pull/30)) and the
hardened version vendored by [WeKnora](https://github.com/Tencent/WeKnora)
(`third_party/anydoc-go`), including the cgo thread-pinning, the panic guard,
the flat document-model serialization, and the build script's crate-patching
approach.
