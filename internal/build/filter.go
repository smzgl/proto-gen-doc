package build

import (
	"fmt"
	"html/template"
	"regexp"
	"strings"
)

var funcMap = map[string]interface{}{
	"p":      PFilter,
	"para":   ParaFilter,
	"nobr":   NoBrFilter,
	"anchor": AnchorFilter,
	"raw":    RawFilter,
	"inc":    IncFilter,
}

var (
	paraPattern         = regexp.MustCompile(`(\n|\r|\r\n)\s*`)
	spacePattern        = regexp.MustCompile("( )+")
	multiNewlinePattern = regexp.MustCompile(`(\r\n|\r|\n){2,}`)
	specialCharsPattern = regexp.MustCompile(`[^a-zA-Z0-9_-]`)
)

// RawFilter TODO
func RawFilter(content string) template.HTML {
	return template.HTML(content)
}

// IncFilter TODO
func IncFilter(content int) template.HTML {
	return template.HTML(fmt.Sprintf("%d", content+1))
}

// PFilter splits the content by new lines and wraps each one in a <p> tag.
func PFilter(content string) template.HTML {
	paragraphs := paraPattern.Split(content, -1)
	return template.HTML(fmt.Sprintf("<p>%s</p>", strings.Join(paragraphs, "</p><p>")))
}

// ParaFilter splits the content by new lines and wraps each one in a <para> tag.
func ParaFilter(content string) string {
	paragraphs := paraPattern.Split(content, -1)
	return fmt.Sprintf("<para>%s</para>", strings.Join(paragraphs, "</para><para>"))
}

// NoBrFilter removes single CR and LF from content.
func NoBrFilter(content string) template.HTML {
	normalized := strings.Replace(content, "\r\n", "\n", -1)
	paragraphs := multiNewlinePattern.Split(normalized, -1)
	for i, p := range paragraphs {
		withoutCR := strings.Replace(p, "\r", " ", -1)
		withoutLF := strings.Replace(withoutCR, "\n", " ", -1)
		paragraphs[i] = spacePattern.ReplaceAllString(withoutLF, " ")
		paragraphs[i] = template.HTMLEscaper(paragraphs[i])
	}
	// return strings.Join(paragraphs, "\n\n")
	return template.HTML(strings.Join(paragraphs, "<br/>"))
}

// NoBrFilter2 removes single CR and LF from content.
// func NoBrFilter2(content string) string {
// 	normalized := strings.Replace(content, "\r\n", "\n", -1)
// 	paragraphs := multiNewlinePattern.Split(normalized, -1)
// 	for i, p := range paragraphs {
// 		withoutCR := strings.Replace(p, "\r", " ", -1)
// 		withoutLF := strings.Replace(withoutCR, "\n", " ", -1)
// 		paragraphs[i] = spacePattern.ReplaceAllString(withoutLF, " ")
// 	}
// 	return strings.Join(paragraphs, "<br/>")
// }

// AnchorFilter replaces all special characters with URL friendly dashes
func AnchorFilter(str string) string {
	return strings.ToLower(specialCharsPattern.ReplaceAllString(strings.ReplaceAll(str, "/", "_"), "-"))
}
