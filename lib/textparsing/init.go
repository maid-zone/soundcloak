package textparsing

import (
	"html"
	"net/url"

	"github.com/dlclark/regexp2/v2"
)

//go:generate go tool regexp2cg -package textparsing -o regexp2_codegen.go
var emailre = regexp2.MustCompile(`^[-a-zA-Z0-9%._\+~#=]+@[-a-zA-Z0-9%._\+~=&]{2,256}\.[a-z]{1,6}$`, regexp2.None)
var theregex = regexp2.MustCompile(`@[a-zA-Z0-9\-_]+|(?:https?:\/\/[-a-zA-Z0-9@%._\+~#=]{2,256}\.[a-z]{1,6}[-a-zA-Z0-9@:%_\+.~#?&\/\/=]*)|(?:[-a-zA-Z0-9%._\+~#=]+@[-a-zA-Z0-9%._\+~=&]{2,256}\.[a-z]{1,6})`, regexp2.None)

func IsEmail(s string) bool {
	t, _ := emailre.MatchString(s)
	return t
}

func replacer(m regexp2.Match) string {
	ent := m.Capture.String()

	if ent[0] == '@' {
		return `<a class="link" href="/` + ent[1:] + `">` + ent + `</a>`
	}

	if len(ent) > len("https://") && ent[:len("https://")] == "https://" || ent[:len("http://")] == "http://" {
		ent = html.UnescapeString(ent)
		parsed, err := url.Parse(ent)
		if err == nil {
			href := ent
			if parsed.Host == "soundcloud.com" || parsed.Host == "on.soundcloud.com" || parsed.Host == "m.soundcloud.com" || parsed.Host == "www.soundcloud.com" {
				idx := 0
				for i := range len(href) {
					if href[i] == '/' {
						idx++
					}
					if idx == 3 {
						idx = i
						break
					}
				}
				href = ent[idx:]
				if parsed.Host == "on.soundcloud.com" {
					href = "/on" + href
				}
			}

			return `<a class="link" href="` + href + `" referrerpolicy="no-referrer" rel="external nofollow noopener noreferrer ugc" target="_blank">` + ent + `</a>`
		}
	}

	// Otherwise, it can only be an email
	return `<a class="link" href="mailto:` + ent + `">` + ent + `</a>`
}

func Format(text string) string {
	text, _ = theregex.ReplaceFunc(html.EscapeString(text), replacer, -1, -1)
	return text
}
