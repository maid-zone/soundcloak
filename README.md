# soundcloak
wip alternative frontend for soundcloud

# [Instance list](https://maid.zone/soundcloak/instances.html)

# Already implemented
- Searching for songs, users, playlists
- Basic user overview (songs, playlists, albums, metadata)
- Basic song overview (author, metadata) & streaming (requires javascript (which requires support for [Media Source Extensions](https://caniuse.com/mediasource)) if no [browser support for HLS](https://caniuse.com/http-live-streaming))
- Basic playlist/set/album overview (songs list, author, metadata)
- Resolving shortened links (`https://on.soundcloud.com/boiKDP46fayYDoVK9` -> `https://sc.maid.zone/on/boiKDP46fayYDoVK9`)

## In the works
- Track player embeds (`https://w.soundcloud.com/player/?url=https%3A//api.soundcloud.com/tracks/id` -> `https://sc.maid.zone/w/player/?url=https%3A//api.soundcloud.com/tracks/id`)

The UI isn't really done yet. All parameters other than url are unsupported. You can also specify track by permalink instead of id (`https://sc.maid.zone/w/player/?url=username/track`)

# Contributing
Contributions are appreciated! This includes feedback, feature requests, issues, pull requests and etc.
Feedback and feature requests are especially needed, since I (laptopcat) don't really know what to prioritize

You can contribute on:
- [GitHub](https://github.com/maid-zone/soundcloak)
- [Codeberg (mirror)](https://codeberg.org/maid-zone/soundcloak)

You can also [reach out to me privately](https://laptopc.at)

# Setting it up
## Prerequisites:
1. [node.js + npm](https://nodejs.org) (any recent enough version should do, it's just used for getting hls.js builds)
2. [golang](https://go.dev) (1.21 or higher was tested, others might work too)
3. [git](https://git-scm.com)

## Setup:
1. Clone this repository:
```sh
git clone https://github.com/maid-zone/soundcloak
```

2. Go into the cloned repository:
```sh
cd soundcloak
```

3. Download hls.js:
```sh
npm i
```

4. Download templ:
```sh
go install github.com/a-h/templ/cmd/templ@latest
```

*You might need to add go binaries to your PATH (add this line to your .bashrc / .zshrc / whatever)*
```sh
export PATH=${PATH}:`go env GOPATH`/bin
```

5. Generate code from templates:

```sh
templ generate
```

6. Download other required go modules:
```sh
go get
```

7. *Optional.* Edit config:

You can change some values in `lib/cfg/init.go` if you want. Keep in mind that you need to rebuild the binary each time you want to update the config.

8. Build binary:

```sh
go build main.go
```

9. Run the binary:
```sh
./main
```

This will run soundcloak on localhost, port 4664. (by default)

# Maintenance-related stuffs
## Updating
1. Retrieve the latest code:

```sh
git fetch origin
git pull
```

2. Remove compressed versions of files:
The webserver is configured to locally cache compressed versions of files. They have `.fiber.gz` extension and can be found in `assets` folder and `node_modules/hls.js/dist`. If any static files have been changed, you should purge these files so the new versions can be served. Static files are also cached in user's browser, so you will need to clean your cache to get the new files (Ctrl + F5)

For example, you can clean these files from `assets` folder like so:
```sh
cd assets
rm *.fiber.gz
```

3. *Optional.* Edit config:

Sometimes, new updates add new config values. You can change some values in `lib/cfg/init.go` if you want.


4. Regenerate templates (if they changed):

```sh
templ generate
```

5. Build binary:
```sh
go build main.go
```

6. Run it:
```sh
./main
```

Congratulations! You have succesfully updated your soundcloak. (hopefully this guide works)

# Built with
## Backend
- [Go programming language](https://github.com/golang/go)
- [Fiber (v2)](https://github.com/gofiber/fiber/tree/v2)
- [templ](https://github.com/a-h/templ)
- [fasthttp](https://github.com/valyala/fasthttp)

## Frontend
- HTML, CSS and JavaScript
- [hls.js](https://github.com/video-dev/hls.js)
