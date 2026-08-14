package reader

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"strconv"

	"github.com/xuri/excelize/v2"
)

// normalizeNumeric reproduces the reference rendering of a numeric cell.
//
// The reference parses a numeric cell to a float and prints its shortest
// round-tripping form, so the stored literal "6.0" becomes "6". RawCellValue
// hands back the literal instead.
//
// The cell type must be consulted, not guessed: "01063476", "6.00" and "NAN" are
// all shared strings that survive ParseFloat, and renumbering them would drop a
// leading zero, drop a trailing zero, or recase NaN. GetCellType is only called
// when re-rendering would actually change the text, which keeps it off the hot
// path for the ~99% of cells that are already canonical or plainly non-numeric.
func normalizeNumeric(f *excelize.File, sheet string, col, row int, v string) string {
	if v == "" {
		return v
	}
	fv, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return v
	}
	out := strconv.FormatFloat(fv, 'f', -1, 64)
	if out == v {
		return v
	}
	axis, err := excelize.CoordinatesToCellName(col+1, row+1)
	if err != nil {
		return v
	}
	// OOXML omits the t attribute on numeric cells, and excelize has no map
	// entry for an empty t, so a plain number reports CellTypeUnset rather than
	// CellTypeNumber. Unset is only reachable here for a cell that exists and
	// parsed as a float — an absent cell is "" and returned above — so both
	// values mean "numeric". Shared strings report CellTypeSharedString and are
	// left alone, which is what protects "01063476", "6.00" and "NAN".
	if t, err := f.GetCellType(sheet, axis); err == nil &&
		(t == excelize.CellTypeNumber || t == excelize.CellTypeUnset) {
		return out
	}
	return v
}

// buildCRFixups maps a shared string's line-ending-normalised form back to its
// raw form, for the strings that contain a carriage return.
//
// Go's encoding/xml performs the line-ending normalisation the XML 1.0 spec
// mandates (CRLF and lone CR both become LF), so excelize returns "a\nb" where
// the reference — which reads the raw bytes — returns "a\r\nb". That difference
// reaches the database: in one 2016 file it affects 2,233 TEN_CUMTHI values,
// which populate the ten_cum_thi column.
//
// The trick is that character references are exempt from that normalisation, so
// rewriting literal CR bytes to &#13; before decoding round-trips them intact.
//
// Returns nil when the file has no CR at all, which is the common case.
func buildCRFixups(path string) map[string]string {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil
	}
	defer zr.Close()

	var entry *zip.File
	for _, zf := range zr.File {
		if zf.Name == "xl/sharedStrings.xml" {
			entry = zf
			break
		}
	}
	if entry == nil {
		return nil
	}
	rc, err := entry.Open()
	if err != nil {
		return nil
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil || !bytes.ContainsRune(data, '\r') {
		return nil
	}
	data = bytes.ReplaceAll(data, []byte{'\r'}, []byte("&#13;"))

	fixups := make(map[string]string)
	dec := xml.NewDecoder(bytes.NewReader(data))
	var cur bytes.Buffer
	inSI, inT := false, false
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "si":
				inSI, cur = true, bytes.Buffer{}
			case "t":
				inT = true
			}
		case xml.CharData:
			if inSI && inT {
				cur.Write(t)
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "t":
				inT = false
			case "si":
				if inSI {
					raw := cur.String()
					if norm := normalizeEOL(raw); norm != raw {
						fixups[norm] = raw
					}
				}
				inSI = false
			}
		}
	}
	if len(fixups) == 0 {
		return nil
	}
	return fixups
}

// normalizeEOL applies XML 1.0 line-ending normalisation: CRLF and lone CR
// both collapse to LF. This is what encoding/xml does to the raw bytes, so it
// reproduces the text excelize hands back.
func normalizeEOL(s string) string {
	if !bytes.ContainsRune([]byte(s), '\r') {
		return s
	}
	var b bytes.Buffer
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\r' {
			if i+1 < len(s) && s[i+1] == '\n' {
				continue // CRLF: the LF is emitted on the next pass
			}
			b.WriteByte('\n') // lone CR
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
