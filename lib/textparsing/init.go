package textparsing

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
)

//var wordre = regexp.MustCompile(`\S+`)

// var urlre = regexp.MustCompile(`https?:\/\/[-a-zA-Z0-9@%._\+~#=]{2,256}\.[a-z]{1,6}[-a-zA-Z0-9@:%_\+.~#?&\/\/=]*`)
var emailre = regexp.MustCompile(`^[-a-zA-Z0-9%._\+~#=]+@[-a-zA-Z0-9%._\+~=&]{2,256}\.[a-z]{1,6}$`)

// var usernamere = regexp.MustCompile(`@[a-zA-Z0-9\-]+`)
var theregex = regexp.MustCompile(`@[a-zA-Z0-9\-]+|(?:https?:\/\/[-a-zA-Z0-9@%._\+~#=]{2,256}\.[a-z]{1,6}[-a-zA-Z0-9@:%_\+.~#?&\/\/=]*)|(?:[-a-zA-Z0-9%._\+~#=]+@[-a-zA-Z0-9%._\+~=&]{2,256}\.[a-z]{1,6})`)

func IsEmail(s string) bool {
	return emailre.MatchString(s)
}

func replacer(ent string) string {
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
	return theregex.ReplaceAllStringFunc(html.EscapeString(text), replacer)
}
