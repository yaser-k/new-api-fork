package common

import (
	"slices"
	"strings"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// emailParts are the two forms a mail leaves in: its HTML body, and the same
// body as plain text.
type emailParts struct {
	html string
	text string
}

// renderEmail makes both parts of a mail from its body.
func renderEmail(content EmailBody) emailParts {
	// Parsing fails only when reading fails, and a string reader cannot.
	nodes, _ := xhtml.ParseFragment(strings.NewReader(string(content)), &xhtml.Node{Type: xhtml.ElementNode, Data: "body", DataAtom: atom.Body})
	var text emailText
	for _, node := range nodes {
		text.write(node)
	}
	text.endParagraph()
	return emailParts{html: string(content), text: strings.Join(text.paragraphs, "\n\n")}
}

// emailText writes a mail body as plain text: tags removed, entities decoded,
// each block on lines of its own with a blank line between blocks, a line
// break where the HTML has one, and a link as its text followed by its address.
type emailText struct {
	paragraphs []string
	lines      []string
	line       strings.Builder
}

var emailTextBlocks = []atom.Atom{
	atom.Article, atom.Blockquote, atom.Center, atom.Dd, atom.Div, atom.Dl, atom.Dt, atom.Footer,
	atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6, atom.Header, atom.Hr, atom.Li, atom.Ol,
	atom.P, atom.Pre, atom.Section, atom.Table, atom.Tr, atom.Ul,
}

func (t *emailText) write(node *xhtml.Node) {
	switch node.Type {
	case xhtml.TextNode:
		t.line.WriteString(node.Data)
		return
	case xhtml.ElementNode:
	default:
		return
	}
	switch node.DataAtom {
	case atom.Br:
		t.endLine()
		return
	case atom.Head, atom.Script, atom.Style, atom.Template, atom.Title:
		return
	}
	block := slices.Contains(emailTextBlocks, node.DataAtom)
	if block {
		t.endParagraph()
	}
	start := t.line.Len()
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		t.write(child)
	}
	if node.DataAtom == atom.A {
		for _, attribute := range node.Attr {
			if attribute.Key == "href" && attribute.Val != "" && t.line.Len() >= start &&
				strings.TrimSpace(t.line.String()[start:]) != attribute.Val {
				t.line.WriteString(" <" + attribute.Val + ">")
			}
		}
	}
	if block {
		t.endParagraph()
	}
}

// endLine ends the line being written, its HTML white space collapsed.
func (t *emailText) endLine() {
	words := strings.FieldsFunc(t.line.String(), func(r rune) bool { return strings.ContainsRune(" \t\n\f\r", r) })
	t.lines = append(t.lines, strings.Join(words, " "))
	t.line.Reset()
}

// endParagraph ends the block being written; a block without text adds none.
func (t *emailText) endParagraph() {
	t.endLine()
	lines := t.lines
	t.lines = nil
	for len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) > 0 {
		t.paragraphs = append(t.paragraphs, strings.Join(lines, "\n"))
	}
}
