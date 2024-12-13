package textparsing

import (
	"fmt"
	"html"
	"net/url"
	"strings"

	"github.com/dlclark/regexp2"
)

//go:generate regexp2cg -package textparsing -o regexp2_codegen.go
var emailre = regexp2.MustCompile(`^[-a-zA-Z0-9%._\+~#=]+@[-a-zA-Z0-9%._\+~=&]{2,256}\.[a-z]{1,6}$`, 0)
var theregex = regexp2.MustCompile(`@[a-zA-Z0-9\-_]+|(?:https?:\/\/[-a-zA-Z0-9@%._\+~#=]{2,256}\.[a-z]{1,6}[-a-zA-Z0-9@:%_\+.~#?&\/\/=]*)|(?:[-a-zA-Z0-9%._\+~#=]+@[-a-zA-Z0-9%._\+~=&]{2,256}\.[a-z]{1,6})`, 0)

func IsEmail(s string) bool {
	t, _ := emailre.MatchString(s)
	return t
}

func replacer(m regexp2.Match) string {
	ent := m.Capture.String()

	if strings.HasPrefix(ent, "@") {
		return fmt.Sprintf(`<a class="link" href="/%s">%s</a>`, ent[1:], ent)
	}

	if strings.HasPrefix(ent, "https://") || strings.HasPrefix(ent, "http://") {
		ent = html.UnescapeString(ent)
		parsed, err := url.Parse(ent)
		if err == nil {
			href := ent
			if parsed.Host == "soundcloud.com" || strings.HasSuffix(parsed.Host, ".soundcloud.com") {
				href = "/" + strings.Join(strings.Split(ent, "/")[3:], "/")
				if parsed.Host == "on.soundcloud.com" {
					href = "/on" + href
				}
			}

			return fmt.Sprintf(`<a class="link" href="%s" referrerpolicy="no-referrer" rel="external nofollow noopener noreferrer ugc" target="_blank">%s</a>`, href, ent)
		}
	}

	// Otherwise, it can only be an email
	return fmt.Sprintf(`<a class="link" href="mailto:%s">%s</a>`, ent, ent)
}

func Format(text string) string {
	text, _ = theregex.ReplaceFunc(html.EscapeString(text), replacer, -1, -1)
	return text
}
