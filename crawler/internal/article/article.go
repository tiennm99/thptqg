// Package article reads download links out of a published web page.
//
// This is what makes the crawler a crawler: the file lists are not carried in
// the repository, they are read from the articles that published them, at run
// time. A source declares which page to read and how to name what it finds.
package article

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"golang.org/x/net/html"
)

// Link is one downloadable file found on a page.
type Link struct {
	// URL is absolute, resolved against the page it was found on.
	URL string
	// Text is the anchor's visible text, whitespace-collapsed. Some pages use
	// it for the thing being linked ("Bạc Liêu"), others for boilerplate
	// ("xem TẠI ĐÂY"), so a source decides whether it is worth anything.
	Text string
	// File is the last segment of the URL path, query string excluded.
	File string
}

// Extract returns every anchor on the page whose target ends in one of exts.
//
// Pure: no network, so the parsing rules can be tested against a saved copy of
// a real page. Order is document order, and duplicates are kept — deciding
// whether two links to the same file is a problem belongs to the caller, which
// knows what it would name them.
func Extract(pageURL string, body io.Reader, exts ...string) ([]Link, error) {
	base, err := url.Parse(pageURL)
	if err != nil {
		return nil, fmt.Errorf("bad page URL %q: %w", pageURL, err)
	}

	doc, err := html.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", pageURL, err)
	}

	var out []Link
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			if link, ok := linkOf(base, n, exts); ok {
				out = append(out, link)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return out, nil
}

// linkOf turns one <a> element into a Link, or reports that it is not one of
// the files being looked for.
func linkOf(base *url.URL, n *html.Node, exts []string) (Link, bool) {
	var href string
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, "href") {
			href = strings.TrimSpace(a.Val)
			break
		}
	}
	if href == "" {
		return Link{}, false
	}

	// Relative hrefs are the norm on the pages this reads; resolving against
	// the page URL is what lets a source name a mirror and get its files.
	ref, err := url.Parse(href)
	if err != nil {
		return Link{}, false
	}
	abs := base.ResolveReference(ref)

	file := path.Base(abs.Path)
	if !hasExt(file, exts) {
		return Link{}, false
	}

	return Link{URL: abs.String(), Text: textOf(n), File: file}, true
}

func hasExt(name string, exts []string) bool {
	lower := strings.ToLower(name)
	for _, e := range exts {
		if strings.HasSuffix(lower, strings.ToLower(e)) {
			return true
		}
	}
	return false
}

// textOf collects an element's visible text, collapsing whitespace. Anchors on
// these pages wrap the label in styling tags, so the text is rarely a single
// child node.
func textOf(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(b.String()), " ")
}

// Fetch downloads a page and runs Extract on it.
func Fetch(ctx context.Context, client *http.Client, pageURL string, headers map[string]string, exts ...string) ([]Link, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", pageURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: HTTP %d", pageURL, resp.StatusCode)
	}

	return Extract(pageURL, resp.Body, exts...)
}
