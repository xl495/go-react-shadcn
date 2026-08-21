package mailer

import (
	"bytes"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var dropTags = map[atom.Atom]bool{
	atom.Script:   true,
	atom.Iframe:   true,
	atom.Object:   true,
	atom.Embed:    true,
	atom.Link:     true,
	atom.Meta:     true,
	atom.Style:    true,
	atom.Form:     true,
	atom.Input:    true,
	atom.Button:   true,
	atom.Textarea: true,
	atom.Select:   true,
	atom.Base:     true,
}

func sanitizeHTML(body string) string {
	nodes, err := html.ParseFragment(strings.NewReader(body), &html.Node{
		Type:     html.ElementNode,
		Data:     "div",
		DataAtom: atom.Div,
	})
	if err != nil {
		return html.EscapeString(body)
	}
	var b bytes.Buffer
	for _, n := range nodes {
		if n.Type == html.ElementNode && dropTags[n.DataAtom] {
			continue
		}
		if n.Type == html.ElementNode {
			n.Attr = filterAttrs(n.Attr)
		}
		sanitizeNode(n)
		if err := html.Render(&b, n); err != nil {
			return html.EscapeString(body)
		}
	}
	return b.String()
}

func sanitizeNode(n *html.Node) {
	if n == nil {
		return
	}
	next := n.FirstChild
	for next != nil {
		child := next
		next = child.NextSibling
		if child.Type == html.ElementNode && dropTags[child.DataAtom] {
			unlink(child)
			continue
		}
		if child.Type == html.ElementNode {
			child.Attr = filterAttrs(child.Attr)
		}
		sanitizeNode(child)
	}
}

func unlink(n *html.Node) {
	if n.Parent == nil {
		return
	}
	if n.PrevSibling != nil {
		n.PrevSibling.NextSibling = n.NextSibling
	} else {
		n.Parent.FirstChild = n.NextSibling
	}
	if n.NextSibling != nil {
		n.NextSibling.PrevSibling = n.PrevSibling
	} else {
		n.Parent.LastChild = n.PrevSibling
	}
	n.Parent, n.PrevSibling, n.NextSibling = nil, nil, nil
}

func filterAttrs(attrs []html.Attribute) []html.Attribute {
	out := attrs[:0]
	for _, a := range attrs {
		key := strings.ToLower(strings.TrimSpace(a.Key))
		if strings.HasPrefix(key, "on") {
			continue
		}
		if key == "href" || key == "src" || key == "xlink:href" {
			if dangerousURL(a.Val) {
				continue
			}
		}
		out = append(out, a)
	}
	return out
}

func dangerousURL(v string) bool {
	s := strings.ToLower(strings.TrimSpace(v))
	return strings.HasPrefix(s, "javascript:") || strings.HasPrefix(s, "data:") || strings.HasPrefix(s, "vbscript:")
}
