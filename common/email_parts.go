package common

import (
	"html"
	"maps"
	"slices"
	"strings"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"golang.org/x/text/language"
)

// EmailFrameOption holds the frames an administrator sets for mails: a JSON
// object whose keys are language tags, or "*" for every other language, and
// whose values are HTML kept as written. A mail is placed in the frame of its
// language tag, else of its base language, else of "*", else in none. A frame
// marks where the mail goes with {{body}}, once, and may use {{lang}}, {{dir}}
// (ltr or rtl), {{align}} (left or right), {{system_name}} and
// {{server_address}}, which are written HTML-escaped. Anything else stays as
// written.
const EmailFrameOption = "EmailFrame"

// EmailFrameMaxBytes bounds the EmailFrame option. Gmail clips a mail whose
// HTML passes about 100 KB.
const EmailFrameMaxBytes = 32 << 10

const (
	emailFrameAnyLanguage = "*"
	emailFrameBody        = "{{body}}"
)

// ParseEmailFrames reads the EmailFrame option into frames keyed by canonical
// language tag, and "*". An empty value sets no frame. Each error says which
// rule the value breaks.
func ParseEmailFrames(value string) (map[string]string, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	if len(value) > EmailFrameMaxBytes {
		return nil, NewMessage("The mail frame must be at most {{max}} bytes", map[string]any{"max": EmailFrameMaxBytes})
	}
	var written map[string]string
	if err := UnmarshalJsonStr(value, &written); err != nil || written == nil {
		return nil, NewMessage("The mail frame must be a JSON object that maps language tags, or * for every other language, to HTML")
	}
	frames := make(map[string]string, len(written))
	for _, key := range slices.Sorted(maps.Keys(written)) {
		tag := key
		if key != emailFrameAnyLanguage {
			parsed, err := language.Parse(key)
			if err != nil {
				return nil, NewMessage("The mail frame key {{key}} is not a language tag or *", map[string]any{"key": key})
			}
			tag = parsed.String()
		}
		if _, ok := frames[tag]; ok {
			return nil, NewMessage("The mail frame names the language {{lang}} more than once", map[string]any{"lang": tag})
		}
		switch strings.Count(written[key], emailFrameBody) {
		case 0:
			return nil, NewMessage("The mail frame for {{key}} has no {{placeholder}} placeholder for the mail body", map[string]any{"key": key, "placeholder": emailFrameBody})
		case 1:
		default:
			return nil, NewMessage("The mail frame for {{key}} has the {{placeholder}} placeholder more than once", map[string]any{"key": key, "placeholder": emailFrameBody})
		}
		frames[tag] = written[key]
	}
	return frames, nil
}

// emailParts are the two forms a mail leaves in: its body in the
// administrator's frame, and the same body as plain text, without the frame.
type emailParts struct {
	html string
	text string
}

// renderEmail makes both parts of a mail from its body. frameOption is the
// saved EmailFrameOption and serverAddress the saved server address. The body
// names its language and direction on its outer element (see i18n.EmailBody);
// a body that names none gets the frame for "*", left to right. A frame option
// that cannot be read never stops a mail: the mail goes without a frame.
func renderEmail(content EmailBody, frameOption, serverAddress string) emailParts {
	// Parsing fails only when reading fails, and a string reader cannot.
	nodes, _ := xhtml.ParseFragment(strings.NewReader(string(content)), &xhtml.Node{Type: xhtml.ElementNode, Data: "body", DataAtom: atom.Body})
	var text emailText
	for _, node := range nodes {
		text.write(node)
	}
	text.endParagraph()
	parts := emailParts{html: string(content), text: strings.Join(text.paragraphs, "\n\n")}

	frames, err := ParseEmailFrames(frameOption)
	if err != nil {
		SysError(LogText("the mail frame option cannot be used, the mail is sent without a frame: %v", err))
		return parts
	}
	lang, direction := "", "ltr"
	if len(nodes) > 0 && nodes[0].Type == xhtml.ElementNode {
		for _, attribute := range nodes[0].Attr {
			switch {
			case attribute.Key == "lang":
				lang = attribute.Val
			case attribute.Key == "dir" && attribute.Val == "rtl":
				direction = "rtl"
			}
		}
	}
	frame, ok := emailFrameFor(frames, lang)
	if !ok {
		return parts
	}
	align := "left"
	if direction == "rtl" {
		align = "right"
	}
	// One pass over the frame: a value that contains a placeholder, the body
	// included, stays as it is.
	parts.html = strings.NewReplacer(
		emailFrameBody, string(content),
		"{{lang}}", html.EscapeString(lang),
		"{{dir}}", direction,
		"{{align}}", align,
		"{{system_name}}", html.EscapeString(SystemName),
		"{{server_address}}", html.EscapeString(serverAddress),
	).Replace(frame)
	return parts
}

// emailFrameFor chooses the frame of a mail written in lang: the frame of its
// tag, else of its base language, else the frame for every other language.
func emailFrameFor(frames map[string]string, lang string) (string, bool) {
	if tag, err := language.Parse(lang); err == nil {
		if frame, ok := frames[tag.String()]; ok {
			return frame, true
		}
		if base, confidence := tag.Base(); confidence == language.Exact {
			if frame, ok := frames[base.String()]; ok {
				return frame, true
			}
		}
	}
	frame, ok := frames[emailFrameAnyLanguage]
	return frame, ok
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
