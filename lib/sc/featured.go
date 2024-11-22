package sc

import "github.com/maid-zone/soundcloak/lib/cfg"

// Functions/structions related to featured/suggested content

type Selection struct {
	Title string               `json:"title"`
	Kind  string               `json:"kind"`  // should always be "selection"!
	Items Paginated[*Playlist] `json:"items"` // ?? why
}

func GetFeaturedTracks(prefs cfg.Preferences, args string) (*Paginated[*Track], error) {
	cid, err := GetClientID()
	if err != nil {
		return nil, err
	}

	p := Paginated[*Track]{Next: "https://" + api + "/featured_tracks/top/all-music" + args + "&client_id=" + cid}
	// DO NOT UNFOLD
	// dangerous
	// seems to go in an infinite loop
	err = p.Proceed(false)
	if err != nil {
		return nil, err
	}

	for _, t := range p.Collection {
		t.Fix(prefs, false)
	}

	return &p, nil
}

func GetSelections(prefs cfg.Preferences) (*Paginated[*Selection], error) {
	cid, err := GetClientID()
	if err != nil {
		return nil, err
	}

	// There is no pagination
	p := Paginated[*Selection]{Next: "https://" + api + "/mixed-selections?limit=20&client_id=" + cid}
	err = p.Proceed(false)
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
		p.Fix(prefs, false)
	}
}
