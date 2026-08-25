package anydoc

// Format names a document format. The string values are the lowercase
// extension names, matching the Node and Python bindings ("doc", "docx",
// "pdf", ...). Use the exported constants rather than raw strings where
// possible: the constants are the values the C ABI round-trips.
type Format string

// Format constants. Every value is stable across versions.
const (
	FormatDoc  Format = "doc"
	FormatDocx Format = "docx"
	FormatOdt  Format = "odt"
	FormatPdf  Format = "pdf"
	FormatPpt  Format = "ppt"
	FormatPptx Format = "pptx"
	FormatRtf  Format = "rtf"
	FormatEpub Format = "epub"
	FormatXlsx Format = "xlsx"
	FormatOds  Format = "ods"
	FormatOdp  Format = "odp"
	FormatCsv  Format = "csv"
)

// C-side format tags mirroring include/anydoc.h (ANYDOC_FORMAT_*). See
// errors.go for why these are Go constants rather than C references.
const (
	formatNone = -1
	formatDoc  = 0
	formatDocx = 1
	formatOdt  = 2
	formatPdf  = 3
	formatPpt  = 4
	formatPptx = 5
	formatRtf  = 6
	formatEpub = 7
	formatXlsx = 8
	formatOds  = 9
	formatOdp  = 10
	formatCsv  = 11
)

// formatToTag maps a Format to its C ABI tag. Returns formatNone for an
// unknown format.
func formatToTag(f Format) int {
	switch f {
	case FormatDoc:
		return formatDoc
	case FormatDocx:
		return formatDocx
	case FormatOdt:
		return formatOdt
	case FormatPdf:
		return formatPdf
	case FormatPpt:
		return formatPpt
	case FormatPptx:
		return formatPptx
	case FormatRtf:
		return formatRtf
	case FormatEpub:
		return formatEpub
	case FormatXlsx:
		return formatXlsx
	case FormatOds:
		return formatOds
	case FormatOdp:
		return formatOdp
	case FormatCsv:
		return formatCsv
	}
	return formatNone
}

// tagToFormat maps a C ABI tag back to a Format. ok is false for formatNone
// or an unknown tag.
func tagToFormat(tag int) (Format, bool) {
	switch tag {
	case formatDoc:
		return FormatDoc, true
	case formatDocx:
		return FormatDocx, true
	case formatOdt:
		return FormatOdt, true
	case formatPdf:
		return FormatPdf, true
	case formatPpt:
		return FormatPpt, true
	case formatPptx:
		return FormatPptx, true
	case formatRtf:
		return FormatRtf, true
	case formatEpub:
		return FormatEpub, true
	case formatXlsx:
		return FormatXlsx, true
	case formatOds:
		return FormatOds, true
	case formatOdp:
		return FormatOdp, true
	case formatCsv:
		return FormatCsv, true
	}
	return "", false
}
