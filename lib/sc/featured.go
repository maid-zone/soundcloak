package sc

import (
	"git.maid.zone/stuff/soundcloak/lib/cfg"
)

// Functions/structures related to featured/suggested content

type Selection struct {
	Title string                        `json:"title"`
	Kind  string                        `json:"kind"`  // should always be "selection"!
	Items Paginated[*UserPlaylistTrack] `json:"items"` // ?? why
}

func GetSelections(prefs cfg.Preferences) (*Paginated[*Selection], error) {
	uri := baseUri()
	uri.SetPath("/mixed-selections")
	uri.QueryArgs().Set("limit", "20")

	// There is no pagination
	p := Paginated[*Selection]{Next: uri}
	err := p.Proceed(false)
	if err != nil {
		return nil, err
	}

	for _, t := range p.Collection {
		t.Fix(prefs)
	}

	return &p, nil
}

func (s *Selection) Fix(prefs cfg.Preferences) {
	for _, p := range s.Items.Collection {
		p.Fix(prefs)
	}
}
