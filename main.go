package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/earlydata"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/maid-zone/soundcloak/lib/sc"
	"github.com/maid-zone/soundcloak/templates"
)

func main() {
	app := fiber.New()
	app.Use(compress.New())
	app.Use(recover.New())
	app.Use(earlydata.New())

	app.Static("/", "assets", fiber.Static{Compress: true, MaxAge: 3600})
	app.Static("/js/hls.js/", "node_modules/hls.js/dist", fiber.Static{Compress: true, MaxAge: 3600})

	app.Get("/:user/:track", func(c *fiber.Ctx) error {
		track, err := sc.GetTrack(c.Params("user") + "/" + c.Params("track"))
		if err != nil {
			fmt.Printf("error getting %s from %s: %s\n", c.Params("track"), c.Params("user"), err)
			return c.SendStatus(404)
		}

		stream, err := track.GetStream()
		if err != nil {
			fmt.Printf("error getting %s stream from %s: %s\n", c.Params("track"), c.Params("user"), err)
		}

		c.Set("Content-Type", "text/html")
		return templates.Base(track.Title+" by "+track.Author.Username, templates.Track(track, stream)).Render(context.Background(), c)
	})

	app.Get("/:user", func(c *fiber.Ctx) error {
		//h := time.Now()
		usr, err := sc.GetUser(c.Params("user"))
		if err != nil {
			fmt.Printf("error getting %s: %s\n", c.Params("user"), err)
			return c.SendStatus(404)
		}
		//fmt.Println("getuser", time.Since(h))

		//h = time.Now()
		p, err := usr.GetTracks(c.Query("pagination", "?limit=20"))
		if err != nil {
			fmt.Printf("error getting %s tracks: %s\n", c.Params("user"), err)
			return c.SendStatus(404)
		}
		//fmt.Println("gettracks", time.Since(h))

		c.Set("Content-Type", "text/html")
		return templates.Base(usr.Username, templates.User(usr, p)).Render(context.Background(), c)
	})

	log.Fatal(app.Listen(":4664"))
}
