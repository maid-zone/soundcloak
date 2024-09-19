# soundcloak
wip alternative frontend for soundcloud

# [Instance list](https://maid.zone/soundcloak/instances.html)

# Already implemented
- Searching for songs, users, playlists
- Basic user overview (songs, metadata)
- Basic song overview (author, metadata) & streaming (requires javascript (which requires support for [Media Source Extensions](https://caniuse.com/mediasource)) if no [browser support for HLS](https://caniuse.com/http-live-streaming))
- Basic playlist/set/album overview (songs list, author, metadata)
- Resolving shortened links (`https://on.soundcloud.com/boiKDP46fayYDoVK9` -> `https://sc.maid.zone/on/boiKDP46fayYDoVK9`)

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

5. Download other required go modules:
```sh
go get
```

6. *Optional.* Edit config:
You can change some values in `lib/cfg/init.go` if you want. Keep in mind that you need to rebuild the binary each time you want to update the config.

7. Generate code from templates & build binary:

*You might need to add go binaries to your PATH (add this line to your .bashrc / .zshrc / whatever)*
```sh
export PATH=${PATH}:`go env GOPATH`/bin
```

```sh
templ generate && go build main.go
```

8. Run the binary:
```sh
./main
```

This will run soundcloak on localhost, port 4664. (by default)

# Built with
## Backend
- [Go programming language](https://github.com/golang/go)
- [Fiber (v2)](https://github.com/gofiber/fiber/tree/v2)
- [templ](https://github.com/a-h/templ)
- [fasthttp](https://github.com/valyala/fasthttp)

## Frontend
- HTML, CSS and JavaScript
- [hls.js](https://github.com/video-dev/hls.js)
