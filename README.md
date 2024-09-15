# soundcloak
wip alternative frontend for soundcloud

# [official public instance](https://sc.maid.zone)
there is no image and audio proxy for now so beware

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
