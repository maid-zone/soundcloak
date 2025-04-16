package sc

import "git.maid.zone/stuff/soundcloak/lib/cfg"

// Functions/structions related to featured/suggested content

type Selection struct {
	Title string               `json:"title"`
	Kind  string               `json:"kind"`  // should always be "selection"!
	Items Paginated[*Playlist] `json:"items"` // ?? why
}

func GetSelections(cid string, prefs cfg.Preferences) (*Paginated[*Selection], error) {
	// There is no pagination
	p := Paginated[*Selection]{Next: "https://" + api + "/mixed-selections?limit=20"}
	err := p.Proceed(cid, false)
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
		p.Fix("", false, false)
		p.Postfix(prefs, false, false)
	}
}
