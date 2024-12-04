# soundcloak

wip alternative frontend for soundcloud

# [Instance list](https://maid.zone/soundcloak/instances.html)

# Already implemented

- Searching for songs, users, playlists
- Basic user overview (songs, playlists, albums, reposts, metadata)
- Basic song overview (author, metadata) & streaming (requires some JS if instance has `Restream` disabled)
- Basic playlist/set/album overview (songs list, author, metadata)
- Resolving shortened links (`https://on.soundcloud.com/boiKDP46fayYDoVK9` -> `https://sc.maid.zone/on/boiKDP46fayYDoVK9`)
- Content proxying (images, audio)
- View featured tracks, playlists
- Users can change their preferences (should proxying be enabled, what method of playing the song should be used etc)

## In the works

- Track player embeds (`https://w.soundcloud.com/player/` -> `https://sc.maid.zone/w/player/`)

The UI isn't really done yet. All parameters other than url are unsupported. You can also specify track without the `soundcloud.com` part: `https://sc.maid.zone/w/player/?url=<id>` or `https://sc.maid.zone/w/player/?url=<user>/<track>`

# Viewing instance settings

If the instance isn't outdated and has `InstanceInfo` setting enabled, you can navigate to `<instance>/_/info` to view useful instance settings. ([sc.maid.zone/_/info](https://sc.maid.zone/_/info) for example)

An easier way is to navigate to `<instance>/_/preferences`.

If some features are disabled by the instance, they won't show up there.

Available features:

- Parse descriptions: Highlight `@username`, `https://example.com` and `email@example.com` in text as clickable links
- Proxy images: Retrieve images through the instance, instead of going to soundcloud's servers for them
- Player: In what way should the track be streamed. Can be Restream (does not require JS, better compatibility, can be a bit buggy client-side) or HLS (requires JS, more stable, less good compatibility (you'll be ok unless you are using a very outdated browser))
- Player-specific settings: They will only show up if you have selected HLS player currently.
- - Proxy streams: Retrieve song pieces through the instance, instead of going to soundcloud's servers for them
- - Fully preload track: Fully loads the track when you load the page instead of buffering a small part of it
- - Autoplay next track in playlists: self-explanatory
- - Default autoplay mode: Default mode for autoplaying. Can be normal (play songs in order) or random (play random song)
# Contributing

Contributions are appreciated! This includes feedback, feature requests, issues, pull requests and etc.
Feedback and feature requests are especially needed, since I (laptopcat) don't really know what to prioritize

You can contribute on:

- [GitHub](https://github.com/maid-zone/soundcloak)
- [Codeberg (mirror)](https://codeberg.org/maid-zone/soundcloak)

You can also [reach out to me privately](https://laptopc.at)

# Setting it up

<details>
<summary>1. Regular method</summary>

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

Refer to [Configuration guide](#configuration-guide) for configuration information. Can be configured from environment variables or JSON file.

8. Build binary:

```sh
go build main.go
```

9. Run the binary:

```sh
./main
```

This will run soundcloak on localhost, port 4664. (by default)

</details>

<details>
<summary>2. Docker image</summary>

The docker image was made by [vlnst](https://github.com/vlnst)

## Prerequisites:

1. [Docker](https://www.docker.com/)
2. [Git](https://git-scm.com)

## Setup:

1. Clone this repository:

```sh
git clone https://github.com/maid-zone/soundcloak
```

2. Go into the cloned repository:

```sh
cd soundcloak
```

3. Make a copy of the example `compose.yaml` file:

```sh
cp compose.example.yaml compose.yaml
```

Make adjustments as needed.

4. *Optional.* Edit config:

Refer to [Configuration guide](#configuration-guide) for configuration information. Can be configured from environment variables or JSON file.

5. Run the container

```sh
docker compose up -d
```

(if you get `docker: 'compose' is not a docker command.`, use `docker-compose up -d`)

This will run soundcloak as a daemon (remove the -d part of the command to just run it) on localhost, port 4664. (by default)

</details>

# Configuration guide

<details>
<summary>Click to view</summary>

You can only configure in one of the two ways:

- Using config file (`soundcloak.json` in current directory // your own path and filename)
- Using environment variables (`SOUNDCLOAK_CONFIG` must be set to `FROM_ENV`!)

Some notes:

- When specifying time, specify it in seconds.


| JSON key                | Environment variable       | Default value                                                                                                                                                                      | Description                                                                                                                                                                          |
| :------------------------ | ---------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| None                    | SOUNDCLOAK_CONFIG          | soundcloak.json                                                                                                                                                                    | File to load soundcloak config from. If set to`FROM_ENV`, soundcloak loads the config from environment variables.                                                                    |
| GetWebProfiles          | GET_WEB_PROFILES           | true                                                                                                                                                                               | Retrieve links users set in their profile (social media, website, etc)                                                                                                               |
| DefaultPreferences      | DEFAULT_PREFERENCES        | {"Player": "hls", "ProxyStreams": false, "FullyPreloadTrack": false, "ProxyImages": false, "ParseDescriptions": true, "AutoplayNextTrack": false, "DefaultAutoplayMode": "normal"} | see /_/preferences page, default values adapt to your config (Player: "restream" if Restream, else "hls", ProxyStreams and ProxyImages will be same as respective config values)     |
| ProxyImages             | PROXY_IMAGES               | false                                                                                                                                                                              | Enables proxying of images (user avatars, track covers etc)                                                                                                                          |
| ImageCacheControl       | IMAGE_CACHE_CONTROL        | max-age=600, public, immutable                                                                                                                                                     | [Cache-Control](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control) header value for proxied images. Cached for 10 minutes by default.                          |
| ProxyStreams            | PROXY_STREAMS              | false                                                                                                                                                                              | Enables proxying of song parts and hls playlist files                                                                                                                                |
| Restream                | RESTREAM                   | false                                                                                                                                                                              | Enables Restream Player in settings and the /_/restream/:author/:track endpoint. This player can be used without JavaScript. Restream also enables the button for downloading songs. |
| RestreamCacheControl    | RESTREAM_CACHE_CONTROL     | max-age=3600, public, immutable                                                                                                                                                    | [Cache-Control](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control) header value for restreamed songs. Cached for 1 hour by default.                            |
| ClientIDTTL             | CLIENT_ID_TTL              | 4 hours                                                                                                                                                                            | Time until ClientID cache expires. ClientID is used for authenticating with SoundCloud API                                                                                           |
| UserTTL                 | USER_TTL                   | 20 minutes                                                                                                                                                                         | Time until User profile cache expires                                                                                                                                                |
| UserCacheCleanDelay     | USER_CACHE_CLEAN_DELAY     | 5 minutes                                                                                                                                                                          | Time between each cleanup of the cache (to remove expired users)                                                                                                                     |
| TrackTTL                | TRACK_TTL                  | 20 minutes                                                                                                                                                                         | Time until Track data cache expires                                                                                                                                                  |
| TrackCacheCleanDelay    | TRACK_CACHE_CLEAN_DELAY    | 5 minutes                                                                                                                                                                          | Time between each cleanup of the cache (to remove expired tracks)                                                                                                                    |
| PlaylistTTL             | PLAYLIST_TTL               | 20 minutes                                                                                                                                                                         | Time until Playlist data cache expires                                                                                                                                               |
| PlaylistCacheCleanDelay | PLAYLIST_CACHE_CLEAN_DELAY | 5 minutes                                                                                                                                                                          | Time between each cleanup of the cache (to remove expired playlists)                                                                                                                 |
| UserAgent               | USER_AGENT                 | Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.3                                                                     | User-Agent header used for requests to SoundCloud                                                                                                                                    |
| DNSCacheTTL             | DNS_CACHE_TTL              | 60 minutes                                                                                                                                                                         | Time until DNS cache expires                                                                                                                                                         |
| Addr                    | ADDR                       | :4664                                                                                                                                                                              | Address and port for soundcloak to listen on                                                                                                                                         |
| Prefork                 | PREFORK                    | false                                                                                                                                                                              | Run multiple instances of soundcloak locally to be able to handle more requests. Each one will be a separate process, so they will have separate cache.                              |
| TrustedProxyCheck       | TRUSTED_PROXY_CHECK        | true                                                                                                                                                                               | Use X-Forwarded-* headers if IP is in TrustedProxies list. When disabled, those headers will blindly be used.                                                                        |
| TrustedProxies          | TRUSTED_PROXIES            | []                                                                                                                                                                                 | List of IPs or IP ranges of trusted proxies                                                                                                                                          |

</details>

# Maintenance-related stuffs

## Updating

Note: this guide works only if you install from source. If you used docker, you could probably do the 1st step (pulling the code), stop the container (`docker container stop soundcloak`) and start it again. (`docker compose up -d`)

<details>
<summary>Click to view</summary>

1. Retrieve the latest code:

```sh
git fetch origin
git pull
```

2. Remove compressed versions of files:

The webserver is configured to locally cache compressed versions of files. They have `.fiber.gz` extension and can be found in `assets` folder and `node_modules/hls.js/dist`. If any static files have been changed, you should purge these compressed files so the new versions can be served. Static files are also cached in user's browser, so you will need to clean your cache to get the new files (Ctrl + F5)

For example, you can clean these files from `assets` folder like so:

```sh
cd assets
rm *.fiber.gz
```

3. *Optional.* Edit config:

Sometimes, new updates add new config values or change default ones. Refer to [Configuration guide](#configuration-guide) for configuration information. Can be configured from environment variables or JSON file.

4. Regenerate templates (if they changed):

```sh
templ generate
```

5. Get latest Go modules:

```sh
go get
```

6. Update hls.js:

```sh
npm i
```

7. Build binary:

```sh
go build main.go
```

8. Run it:

```sh
./main
```

Congratulations! You have succesfully updated your soundcloak.

</details>

# Built with

## Backend

- [Go programming language](https://github.com/golang/go)
- [Fiber (v2)](https://github.com/gofiber/fiber/tree/v2)
- [templ](https://github.com/a-h/templ)
- [fasthttp](https://github.com/valyala/fasthttp)

## Frontend

- HTML, CSS and JavaScript
- [hls.js](https://github.com/video-dev/hls.js)
