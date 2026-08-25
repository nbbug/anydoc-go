//go:build cgo && ((linux && (amd64 || arm64)) || (darwin && (amd64 || arm64)) || (windows && amd64))

package anydoc

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// ---- Fixtures -------------------------------------------------------------
//
// The fixtures are built in memory rather than checked in: they are the
// smallest OOXML packages each parser accepts, and keep the repository free
// of binary test data. If a parser grows stricter, extend the fixture here.

// zipFixture packs name → content entries into a ZIP archive.
func zipFixture(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range entries {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}

// docxFixture is a minimal WordprocessingML package: one Heading1 paragraph
// and one body paragraph.
func docxFixture(t *testing.T) []byte {
	t.Helper()
	return zipFixture(t, map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`,
		"word/document.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:body>
<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>Title</w:t></w:r></w:p>
<w:p><w:r><w:t>Hello world</w:t></w:r></w:p>
</w:body>
</w:document>`,
	})
}

// xlsxFixture is a minimal SpreadsheetML package: one worksheet with a header
// row and one data row.
func xlsxFixture(t *testing.T) []byte {
	t.Helper()
	return zipFixture(t, map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`,
		"xl/workbook.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<sheets><sheet name="Sheet1" sheetId="1" r:id="rId1"/></sheets>
</workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
</Relationships>`,
		"xl/worksheets/sheet1.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
<sheetData>
<row r="1"><c r="A1" t="inlineStr"><is><t>Name</t></is></c><c r="B1" t="inlineStr"><is><t>Age</t></is></c></row>
<row r="2"><c r="A2" t="inlineStr"><is><t>Alice</t></is></c><c r="B2"><v>30</v></c></row>
</sheetData>
</worksheet>`,
	})
}

const csvFixture = "name,age\nAlice,30\n"

// ---- Conversion -----------------------------------------------------------

func TestToMarkdownBytesCSV(t *testing.T) {
	md, err := ToMarkdownBytes([]byte(csvFixture), formatPtr(string(FormatCsv)))
	if err != nil {
		t.Fatalf("ToMarkdownBytes csv: %v", err)
	}
	for _, want := range []string{"name", "Alice", "30"} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q:\n%s", want, md)
		}
	}
}

func TestToMarkdownBytesDocx(t *testing.T) {
	md, err := ToMarkdownBytes(docxFixture(t), nil)
	if err != nil {
		t.Fatalf("ToMarkdownBytes docx: %v", err)
	}
	for _, want := range []string{"Title", "Hello world"} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q:\n%s", want, md)
		}
	}
	if !strings.Contains(md, "# Title") {
		t.Errorf("expected the Heading1 paragraph as '# Title':\n%s", md)
	}
}

func TestToMarkdownBytesXlsx(t *testing.T) {
	md, err := ToMarkdownBytes(xlsxFixture(t), nil)
	if err != nil {
		t.Fatalf("ToMarkdownBytes xlsx: %v", err)
	}
	for _, want := range []string{"Name", "Alice", "30"} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q:\n%s", want, md)
		}
	}
}

func TestToMarkdownFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.csv")
	if err := os.WriteFile(path, []byte(csvFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	md, err := ToMarkdown(path)
	if err != nil {
		t.Fatalf("ToMarkdown: %v", err)
	}
	if !strings.Contains(md, "Alice") {
		t.Errorf("markdown missing data:\n%s", md)
	}
}

// ---- Document model -------------------------------------------------------

func TestToDocumentDocx(t *testing.T) {
	doc, err := ToDocument(docxFixture(t), nil)
	if err != nil {
		t.Fatalf("ToDocument docx: %v", err)
	}
	if len(doc.Blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d: %+v", len(doc.Blocks), doc.Blocks)
	}
	heading := doc.Blocks[0]
	if heading.Kind != "heading" || heading.Level == nil || *heading.Level != 1 {
		t.Errorf("expected a level-1 heading, got %+v", heading)
	}
	if len(heading.Content) == 0 || heading.Content[0].Kind != "text" ||
		heading.Content[0].Text == nil || *heading.Content[0].Text != "Title" {
		t.Errorf("expected heading text 'Title', got %+v", heading.Content)
	}
	para := doc.Blocks[1]
	if para.Kind != "paragraph" || len(para.Content) == 0 ||
		para.Content[0].Text == nil || *para.Content[0].Text != "Hello world" {
		t.Errorf("expected paragraph text 'Hello world', got %+v", para)
	}
}

func TestToDocumentXlsxTable(t *testing.T) {
	doc, err := ToDocument(xlsxFixture(t), nil)
	if err != nil {
		t.Fatalf("ToDocument xlsx: %v", err)
	}
	if len(doc.Blocks) == 0 || doc.Blocks[0].Kind != "table" || doc.Blocks[0].Table == nil {
		t.Fatalf("expected a table block, got %+v", doc.Blocks)
	}
	grid := doc.Blocks[0].Table.Grid
	if len(grid) == 0 || len(grid[0]) == 0 || grid[0][0].Kind != "origin" {
		t.Fatalf("expected an origin cell at 0,0, got %+v", grid)
	}
	if got := cellText(grid[0][0]); got != "Name" {
		t.Errorf("expected cell text 'Name', got %q", got)
	}
}

func cellText(slot CellSlot) string {
	if slot.Cell == nil || len(slot.Cell.Blocks) == 0 || len(slot.Cell.Blocks[0].Content) == 0 {
		return ""
	}
	txt := slot.Cell.Blocks[0].Content[0].Text
	if txt == nil {
		return ""
	}
	return *txt
}

// ---- Format detection -----------------------------------------------------

func TestFormatFromBytes(t *testing.T) {
	if f, ok := FormatFromBytes(docxFixture(t)); !ok || f != FormatDocx {
		t.Errorf("expected docx, got %q (ok=%v)", f, ok)
	}
	if f, ok := FormatFromBytes(xlsxFixture(t)); !ok || f != FormatXlsx {
		t.Errorf("expected xlsx, got %q (ok=%v)", f, ok)
	}
	// CSV carries no signature and must be named explicitly.
	if f, ok := FormatFromBytes([]byte(csvFixture)); ok {
		t.Errorf("expected no detection for csv, got %q", f)
	}
	if f, ok := FormatFromBytes([]byte("garbage bytes")); ok {
		t.Errorf("expected no detection for garbage, got %q", f)
	}
}

func TestFormatFromExtension(t *testing.T) {
	for ext, want := range map[string]Format{
		"docx": FormatDocx, ".XLSX": FormatXlsx, "pdf": FormatPdf, "epub": FormatEpub,
	} {
		if f, ok := FormatFromExtension(ext); !ok || f != want {
			t.Errorf("FormatFromExtension(%q) = %q, %v; want %q", ext, f, ok, want)
		}
	}
	if _, ok := FormatFromExtension("nope"); ok {
		t.Error("expected ok=false for unknown extension")
	}
}

func TestFormatFromPath(t *testing.T) {
	if f, ok := FormatFromPath("/data/report.epub"); !ok || f != FormatEpub {
		t.Errorf("expected epub, got %q (ok=%v)", f, ok)
	}
	if _, ok := FormatFromPath("/data/noext"); ok {
		t.Error("expected ok=false for extension-less path")
	}
}

// ---- Errors ---------------------------------------------------------------

func TestErrorKinds(t *testing.T) {
	cases := []struct {
		name   string
		data   []byte
		format *Format
		kind   string
	}{
		{"empty input", nil, nil, "unsupported"},
		{"garbage bytes", []byte("not a document"), nil, "unsupported"},
		{"unknown format", []byte("x"), formatPtr("bogus"), "unknown_format"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			if tc.data == nil {
				_, err = ToMarkdownBytes(nil, tc.format)
			} else {
				_, err = ToMarkdownBytes(tc.data, tc.format)
			}
			ce, ok := err.(*ConvertError)
			if !ok {
				t.Fatalf("expected *ConvertError, got %T: %v", err, err)
			}
			if ce.Kind != tc.kind {
				t.Errorf("expected kind %q, got %q (detail: %q)", tc.kind, ce.Kind, ce.Detail)
			}
			if ce.Detail == "" {
				t.Error("expected a non-empty detail message")
			}
		})
	}
}

func TestPDFNoModel(t *testing.T) {
	// Naming the format explicitly guarantees the PDF branch regardless of
	// signature detection.
	_, err := ToDocument([]byte("%PDF-1.4\n%%EOF\n"), formatPtr(string(FormatPdf)))
	ce, ok := err.(*ConvertError)
	if !ok || ce.Kind != "pdf_no_model" {
		t.Fatalf("expected pdf_no_model, got %v", err)
	}
	// Markdown conversion of the same bytes must not hit that branch.
	if _, err := ToMarkdownBytes([]byte("%PDF-1.4\n%%EOF\n"), formatPtr(string(FormatPdf))); err == nil {
		t.Skip("pdf markdown conversion succeeded; full parse of a stub PDF")
	} else if ce, ok := err.(*ConvertError); ok && ce.Kind == "pdf_no_model" {
		t.Errorf("ToMarkdownBytes must not return pdf_no_model")
	}
}

func formatPtr(f string) *Format {
	v := Format(f)
	return &v
}

// ---- Embedding ------------------------------------------------------------

func TestEmbeddedLibAndExtract(t *testing.T) {
	lib := EmbeddedLib()
	if len(lib) < 1<<20 {
		t.Fatalf("expected the embedded archive to be at least 1 MB, got %d bytes", len(lib))
	}
	dir := t.TempDir()
	path, err := ExtractLib(dir)
	if err != nil {
		t.Fatalf("ExtractLib: %v", err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if st.Size() != int64(len(lib)) {
		t.Errorf("extracted %d bytes, embedded %d", st.Size(), len(lib))
	}
	if !strings.Contains(path, Version) {
		t.Errorf("extracted path should record the version, got %s", path)
	}
}

// ---- Wire constants -------------------------------------------------------

// TestWireConstsMatchHeader pins the pure-Go tag and error constants to the
// committed C header. Both sides are generated by hand from the same Rust
// definitions, so this is the tripwire that catches a drift at compile time.
// The header is parsed instead of importing C because go vet does not support
// cgo in test files.
func TestWireConstsMatchHeader(t *testing.T) {
	header, err := os.ReadFile("include/anydoc.h")
	if err != nil {
		t.Fatalf("read include/anydoc.h: %v", err)
	}
	values := make(map[string]int)
	for _, line := range strings.Split(string(header), "\n") {
		var name string
		var value int
		if _, err := fmt.Sscanf(strings.TrimSpace(line), "#define %s %d", &name, &value); err == nil {
			values[name] = value
		}
	}
	cases := []struct {
		name string
		goV  int
		c    string
	}{
		{"errOK", errOK, "ERR_OK"},
		{"errUnsupported", errUnsupported, "ERR_UNSUPPORTED"},
		{"errMalformed", errMalformed, "ERR_MALFORMED"},
		{"errEncrypted", errEncrypted, "ERR_ENCRYPTED"},
		{"errResourceLimit", errResourceLimit, "ERR_RESOURCE_LIMIT"},
		{"errMissingPart", errMissingPart, "ERR_MISSING_PART"},
		{"errIO", errIO, "ERR_IO"},
		{"errPDFNoModel", errPDFNoModel, "ERR_PDF_NO_MODEL"},
		{"errInvalidArg", errInvalidArg, "ERR_INVALID_ARG"},
		{"errUnknownFormat", errUnknownFormat, "ERR_UNKNOWN_FORMAT"},
		{"formatNone", formatNone, "ANYDOC_FORMAT_NONE"},
		{"formatDoc", formatDoc, "ANYDOC_FORMAT_DOC"},
		{"formatCsv", formatCsv, "ANYDOC_FORMAT_CSV"},
		{"blockHeading", blockHeading, "BLOCK_HEADING"},
		{"blockRule", blockRule, "BLOCK_RULE"},
		{"inlineMath", inlineMath, "INLINE_MATH"},
		{"inlineCheckbox", inlineCheckbox, "INLINE_CHECKBOX"},
		{"imgUnavailable", imgUnavailable, "IMG_UNAVAILABLE"},
		{"markerUpperRoman", markerUpperRoman, "MARKER_UPPER_ROMAN"},
		{"tableLayout", tableLayout, "TABLE_LAYOUT"},
		{"slotCovered", slotCovered, "SLOT_COVERED"},
		{"noteEndnote", noteEndnote, "NOTE_ENDNOTE"},
	}
	for _, tc := range cases {
		cV, ok := values[tc.c]
		if !ok {
			t.Errorf("%s: header define %s not found", tc.name, tc.c)
			continue
		}
		if tc.goV != cV {
			t.Errorf("%s: Go constant %d != header %d", tc.name, tc.goV, cV)
		}
	}
}

// ---- Decoder bounds -------------------------------------------------------

// TestDecoderRejectsCorruptBuffers feeds truncated and corrupt buffers to the
// decoder. The Rust side always produces a well-formed buffer, so a short
// read is a version skew — it must error, never panic or exhaust memory.
func TestDecoderRejectsCorruptBuffers(t *testing.T) {
	// A plausible document buffer: 0 blocks, 0 notes, 0 assets.
	valid := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	for i := 0; i < len(valid); i++ {
		_, err := decodeDocument(valid[:i])
		if err == nil {
			t.Errorf("expected error for truncated buffer of length %d", i)
		}
	}
	// A block count that claims 4 billion entries must fail fast, not
	// preallocate 4 billion Blocks.
	huge := append([]byte{0xff, 0xff, 0xff, 0xff}, make([]byte, 64)...)
	if _, err := decodeDocument(huge); err == nil {
		t.Error("expected error for implausible block count")
	}
}

// ---- Concurrency ----------------------------------------------------------

// TestErrorDetailSurvivesConcurrency hammers conversions from many goroutines
// and checks each result belongs to its own input. The ABI reports error
// detail through a thread-local slot, so a missed LockOSThread would surface
// here as wrong or empty output.
func TestConcurrencyDoesNotCrossThreads(t *testing.T) {
	const goroutines = 16
	const iterations = 20

	var wg sync.WaitGroup
	errs := make(chan error, goroutines)
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				marker := fmt.Sprintf("goroutine-%d-iteration-%d", g, i)
				csv := "name\n" + marker + "\n"
				md, err := ToMarkdownBytes([]byte(csv), formatPtr(string(FormatCsv)))
				if err != nil {
					errs <- fmt.Errorf("%s: %v", marker, err)
					return
				}
				if !strings.Contains(md, marker) {
					errs <- fmt.Errorf("output for %s does not contain it:\n%s", marker, md)
					return
				}
			}
		}(g)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}
