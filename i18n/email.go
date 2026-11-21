package i18n

import (
	"html"
	"slices"
	"strings"

	"golang.org/x/text/language"

	"github.com/QuantumNous/new-api/common"
)

// rightToLeftScripts are the scripts in current use that are written right to
// left (Unicode bidirectional class R or AL). A language tag names its script
// or implies its likely one: fa and ckb imply Arab, he implies Hebr, while ku
// and az imply Latn.
var rightToLeftScripts = []string{"Adlm", "Arab", "Aran", "Hebr", "Mand", "Mend", "Nkoo", "Rohg", "Samr", "Syrc", "Thaa", "Yezi"}

// EmailBody builds the HTML body of a mail from paragraphs translated for lang
// (the language given to Translate). The wrapper names the language the text
// is written in, and the wrapper and every paragraph carry its direction as a
// dir attribute and inline direction and text-align styles, which desktop mail
// clients honour where they ignore inherited styles. A code from EmailCode is
// a block of its own and is placed as it is.
func EmailBody(lang string, paragraphs ...string) common.EmailBody {
	return emailBodyIn(messageLang(lang), paragraphs)
}

// emailBodyIn builds the body of paragraphs written in the language tag lang.
func emailBodyIn(lang string, paragraphs []string) common.EmailBody {
	direction, align := "ltr", "left"
	if script, _ := language.Make(lang).Script(); slices.Contains(rightToLeftScripts, script.String()) {
		direction, align = "rtl", "right"
	}
	block := ` dir="` + direction + `" style="direction:` + direction + `;text-align:` + align + `"`
	var body strings.Builder
	body.WriteString(`<div lang="` + html.EscapeString(lang) + `"` + block + `>`)
	for _, paragraph := range paragraphs {
		if strings.HasPrefix(paragraph, emailCodeOpen) {
			body.WriteString(paragraph)
			continue
		}
		body.WriteString(`<p` + block + `>` + paragraph + `</p>`)
	}
	body.WriteString(`</div>`)
	return common.EmailBody(body.String())
}

// EmailLTR escapes text that is always written left to right (a code, a link,
// an e-mail address, a host name) and keeps it whole inside a mail of either
// direction. dir="ltr" on a span is HTML 4: desktop clients that do not know
// <bdi> honour it, and browsers isolate it. It adds no characters, so copying
// a code or a link copies nothing else.
func EmailLTR(text string) string {
	return `<span dir="ltr">` + html.EscapeString(text) + `</span>`
}

// emailCodeOpen begins the block EmailCode writes: centred and left to right,
// large and bold in a monospace stack, its characters spaced by letter-spacing.
// The background colour is set only with a text colour, so a client's dark
// mode that changes one changes both.
const emailCodeOpen = `<div dir="ltr" style="direction:ltr;text-align:center;margin:16px 0;padding:12px 16px;` +
	`background-color:#f3f4f6;color:#111827;font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,'Liberation Mono','Courier New',monospace;` +
	`font-size:28px;font-weight:bold;line-height:1.4;letter-spacing:6px">`

// EmailCode writes a code the reader types or pastes as a block of its own in
// a mail of either direction; the label goes in the paragraph before it. The
// block holds the escaped code and no other character, so copying it copies
// the code alone, and the plain-text part has the code on a line of its own.
func EmailCode(code string) string {
	return emailCodeOpen + html.EscapeString(code) + `</div>`
}
