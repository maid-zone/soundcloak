package sc

import "encoding/xml"

// vendored type definitions from github.com/gorilla/feeds
// the only thing used from that module so why not drop some dead weight :)

type RssFeedXml struct {
	XMLName          xml.Name `xml:"rss"`
	Version          string   `xml:"version,attr"`
	ContentNamespace string   `xml:"xmlns:content,attr"`
	Channel          *RssFeed
}

type RssFeed struct {
	XMLName     xml.Name `xml:"channel"`
	Title       string   `xml:"title"`       // required
	Link        string   `xml:"link"`        // required
	Description string   `xml:"description"` // required
	//Language       string   `xml:"language,omitempty"`
	//Copyright      string   `xml:"copyright,omitempty"`
	ManagingEditor string `xml:"managingEditor,omitempty"` // Author used
	//WebMaster      string   `xml:"webMaster,omitempty"`
	PubDate       string `xml:"pubDate,omitempty"`       // created or updated
	LastBuildDate string `xml:"lastBuildDate,omitempty"` // updated used
	Category      string `xml:"category,omitempty"`
	Generator     string `xml:"generator,omitempty"`
	//Docs           string   `xml:"docs,omitempty"`
	//Cloud          string   `xml:"cloud,omitempty"`
	Ttl int `xml:"ttl,omitempty"`
	//Rating         string   `xml:"rating,omitempty"`
	//SkipHours      string   `xml:"skipHours,omitempty"`
	//SkipDays       string   `xml:"skipDays,omitempty"`
	//Image          *RssImage
	//TextInput      *RssTextInput
	Items []*RssItem `xml:"item"`
}

type RssItem struct {
	XMLName     xml.Name `xml:"item"`
	Title       string   `xml:"title"`       // required
	Link        string   `xml:"link"`        // required
	Description string   `xml:"description"` // required
	//Content     *RssContent
	//Author    string `xml:"author,omitempty"`
	Category string `xml:"category,omitempty"`
	//Comments  string `xml:"comments,omitempty"`
	//Enclosure *RssEnclosure
	Guid    *RssGuid // Id used
	PubDate string   `xml:"pubDate,omitempty"` // created or updated
	//Source  string   `xml:"source,omitempty"`
}

type RssGuid struct {
	//RSS 2.0 <guid isPermaLink="true">http://inessential.com/2002/09/01.php#a2</guid>
	XMLName     xml.Name `xml:"guid"`
	Id          string   `xml:",chardata"`
	IsPermaLink string   `xml:"isPermaLink,attr,omitempty"` // "true", "false", or an empty string
}
