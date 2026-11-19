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
// clients honour where they ignore inherited styles.
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
