package api

import (
	"git.maid.zone/stuff/soundcloak/lib/cfg"
	"git.maid.zone/stuff/soundcloak/lib/sc"
	"github.com/gofiber/fiber/v3"
)

func Load(a *fiber.App) {
	r := a.Group("/_/api")
	r.Use("/v2", func(c fiber.Ctx) error {
		req := c.Request()
		p := req.URI().Path()[len("/_/api/v2"):]
		if string(p) == "/tracks" ||
			string(p) == "/resolve" ||
			string(p) == "/mixed-selections" ||
			string(p) == "/charts/selections" ||
			(len(p) > len("/users/") &&
				string(p[:len("/users/")]) == "/users/") ||
			(len(p) > len("/tracks/") &&
				(string(p[:len("/tracks/")]) == "/tracks/" || string(p[:len("/search/")]) == "/search/")) ||
			(len(p) > len("/playlists/") &&
				string(p[:len("/playlists/")]) == "/playlists/") ||
			(len(p) > len("/featured_tracks/") &&
				string(p[:len("/featured_tracks/")]) == "/featured_tracks/") {
			goto great
		}
		return c.SendStatus(404)
	great:
		// api-v2 only has gzip, so we use only gzip
		gzip := req.Header.HasAcceptEncoding("gzip")
		req.Header.Reset()
		req.ResetBody()
		req.Header.SetUserAgent(cfg.UserAgent)
		req.Header.Set("Accept-Encoding", "gzip")

		req.URI().SetScheme("https")
		req.URI().SetHost("api-v2.soundcloud.com")
		if !req.URI().QueryArgs().Has("client_id") {
			req.URI().QueryArgs().Set("client_id", sc.ClientID)
		}
		req.URI().SetPathBytes(p)

		resp := c.Response()
		err := sc.DoWithRetry(sc.Httpc, req, resp)
		if err != nil {
			return err
		}

		// very big shame on you for not even implementing gzip compression
		if !gzip && string(resp.Header.ContentEncoding()) == "gzip" {
			p, err = resp.BodyGunzip()
			if err == nil {
				resp.SetBody(p)
			}
		}
		return err
	})

	// DEPRECATED
	legacy(r)
}
