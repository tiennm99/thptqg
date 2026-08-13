package sources

import (
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/text/unicode/norm"

	"github.com/tiennm99/thptqg/crawler/internal/article"
)

// browserUA and the Referer are both required: the CDN 403s an unrecognised
// User-Agent, and rejects requests that do not carry the article as Referer.
const browserUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36"

const article2017 = "https://baotintuc.vn/tuyen-sinh/tra-cuu-diem-thi-thpt-2017-cua-63-tinh-thanh-pho-tren-baotintucvn-20170706073512672.htm"

// source2017 fetches the 2017 dataset: one .xls per province, still served from
// the CDN that originally published it.
//
// Dest cannot use the CDN's filenames the way 2016 does, because they are
// inconsistent — Angiang.xls, 1BaRiaVungTau.xls, 23HaiPhong.xls, Gia-Lai.xls —
// carrying upload-order prefixes and arbitrary capitalisation. The province name
// in the link text is the stable identifier, so the local name is derived from
// that instead, which is what produced the files in data/2017/.
var source2017 = Source{
	ID:        "2017",
	Summary:   "63 province .xls files from the baotintuc.vn CDN",
	Article:   article2017,
	Exts:      []string{".xls"},
	WantFiles: 63,
	Headers: map[string]string{
		"User-Agent": browserUA,
		"Referer":    article2017,
	},
	Dest: func(l article.Link) (string, error) {
		s := slug(l.Text)
		if s == "" {
			return "", fmt.Errorf("link %s has no usable label to name it by", l.URL)
		}
		return s + ".xls", nil
	},
}

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

// slug turns a province name into a filename stem: transliterated to ASCII,
// lowercased, with every run of other characters collapsed to one hyphen.
//
//	"Bà Rịa -Vũng Tàu" -> "ba-ria-vung-tau"
//	"Bắc Kạn"          -> "bac-kan"
//
// The article writes the names with diacritics, so this has to strip them.
// Combining marks are removed by the literal range U+0300–U+036F rather than by
// the unicode.Mn category, matching what go-parser does to build ho_ten_ascii.
func slug(name string) string {
	var b strings.Builder
	for _, r := range norm.NFD.String(name) {
		switch {
		case r >= 0x0300 && r <= 0x036F:
			// combining mark: drop
		case r == 'đ' || r == 'Đ':
			b.WriteByte('d')
		default:
			b.WriteRune(r)
		}
	}
	return strings.Trim(nonAlphanumeric.ReplaceAllString(strings.ToLower(b.String()), "-"), "-")
}
