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

# Settings

## Viewing instance config
If the instance isn't outdated and has `InstanceInfo` setting enabled, you can navigate to `<instance>/_/info` to view useful instance settings. ([sc.maid.zone/_/info](https://sc.maid.zone/_/info) for example)

## Preferences

You can go to `/_/preferences` page to configure them. You can view default preferences using the method described above or by resetting your preferences.

You can also Export/Import your preferences, for backup purposes or for easily transfering them between instances.

- Parse descriptions: Highlight `@username`, `https://example.com` and `email@example.com` in text as clickable links
- Show current audio: shows what [preset](#audio-presets) is currently playing (mpeg, opus or aac)
- Proxy images: Retrieve images through the instance, instead of going to soundcloud's servers for them
- Player: In what way should the track be streamed. Can be Restream (does not require JS, better compatibility, can be a bit buggy client-side) or HLS (requires JS, more stable, less good compatibility (you'll be ok unless you are using a very outdated browser))
- Autoplay next track in playlists: self-explanatory
- Default autoplay mode: Default mode for autoplaying. Can be normal (play songs in order) or random (play random song)
- Player-specific settings:
- - HLS Player:
- - - Proxy streams: Retrieve song pieces through the instance, instead of going to soundcloud's servers for them
- - - Fully preload track: Fully loads the track when you load the page instead of buffering a small part of it
- - - Streaming audio: What [preset](#audio-presets) of audio should be streamed (Opus is not supported here)
- - Restream Player:
- - - Streaming audio: What [preset](#audio-presets) of audio should be streamed

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
2. [golang](https://go.dev) (I recommend version 1.22.10. Technically, you need 1.21.4 or higher)
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

5. Download regexp2cg:

Not really required, but helps speed up some parts of the code that use regular expressions. Keep in mind that the `build` script expects this to be installed.

```sh
go install github.com/dlclark/regexp2cg@main
```

*You might need to add go binaries to your PATH (add this line to your .bashrc / .zshrc / whatever)*

```sh
export PATH=${PATH}:`go env GOPATH`/bin
```

6. *Optional.* Edit config:

Refer to [Configuration guide](#configuration-guide) for configuration information. Can be configured from environment variables or JSON file.

7. Build binary:

This uses the `build` script, which generates code from templates, generates code for regular expiressions, and then builds the binary.

```sh
./build
```

8. Run the binary:

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
| None                    | SOUNDCLOAK_CONFIG          | soundcloak.json                                                                                                                                                                    | File to load soundcloak config from. If set to `FROM_ENV`, soundcloak loads the config from environment variables.                                                                    |
| GetWebProfiles          | GET_WEB_PROFILES           | true                                                                                                                                                                               | Retrieve links users set in their profile (social media, website, etc)                                                                                                               |
| DefaultPreferences      | DEFAULT_PREFERENCES        | {"Player": "hls", "ProxyStreams": false, "FullyPreloadTrack": false, "ProxyImages": false, "ParseDescriptions": true, "AutoplayNextTrack": false, "DefaultAutoplayMode": "normal", "HLSAudio": "mpeg", "RestreamAudio": "mpeg", "DownloadAudio": "mpeg"} | see /_/preferences page. [Read more](#preferences-1)  |
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

## Preferences

<details>
<summary>Click to view</summary>


| Name                | Default             | Description                                                                                                                                                                                          | Possible values               |
| --------------------- | --------------------- | :----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :------------------------------ |
| Player              | "restream" if Restream is enabled in config, otherwise - "hls"              | Method used to play the track in the frontend. HLS - requires JavaScript, loads the track in pieces. Restream - works without JavaScript, loads entire track right away. None - don't play the track | "hls", "restream", "none"     |
| ProxyStreams        | same as your config | Proxy track streams. Refer to configuration guide for more info. Not effective unless ProxyStreams is enabled in your config and you are using HLS player (Restream proxies songs by default)        | true, false                   |
| FullyPreloadTrack   | false               | Fully load track when the page is loaded. Only effective if you are using HLS player                                                                                                                 | true, false                   |
| ParseDescriptions   | true                | Highlight links, usernames and emails in track/user/playlist descriptions                                                                                                                            | true, false                   |
| AutoplayNextTrack   | false               | Automatically start playlist playback when you open a track from it                                                                                                                                  | true, false                   |
| DefaultAutoplayMode | "normal"            | Default mode for autoplay. Normal - play songs in order. Random - play random song next                                                                                                              | "normal", "random"            |
| HLSAudio            | "mpeg"              | What audio preset should be loaded when using HLS player. Note that "opus" is not supported here. [Read more](#audio-presets)                                                                         | "aac", "mpeg"                 |
| RestreamAudio       | "mpeg"              | What audio preset should be loaded when using Restream player. [Read more](#audio-presets)                                                                                                            | "best", "aac", "opus", "mpeg" |
| DownloadAudio       | "mpeg"              | What audio preset should be loaded when downloading audio with metadata. [Read more](#audio-presets)                                                                                                  | "best", "aac", "opus", "mpeg" |
| ShowAudio           | false               | Show what audio preset was loaded on the track page                                                                                                                                                  | true, false                   |

</details>

## Audio presets

<details>
<summary>Click to view</summary>


| Name | Container  | Codec | Bitrate | Note                                                                                   |
| ---- | ---------- | ----- | ------- | -------------------------------------------------------------------------------------- |
| Best |            |       |         | Prefer AAC over Opus over MPEG. Not supported for HLS player (use AAC for same effect) |
| AAC  | mp4 (m4a)  | AAC   | 160kbps | Rarely available. Falls back to MPEG if unavailable                                    |
| Opus | ogg        | Opus  | 72kbps  | Usually available. Falls back to MPEG if unavailable. Not supported for HLS player     |
| MP3  | mpeg (mp3) | MP3   | 128kbps | Always available. Good for compatibility                                               |

</details>

## Tinkering with the frontend

<details>
<summary>Click to view</summary>

I will mainly talk about the static files here. Maybe about the templates too in the future

The static files are stored in `assets` folder

### Overriding files

You can override files by putting identically named files in the `instance` folder.

### Basic theming

1. Create `instance.css` file in `instance` folder
2. Put your CSS rules there:
```css
/* Some basic CSS to change colors of the frontend. Put your own colors here as this one probably looks horrible (I did not test it) */
:root {
    --accent: #ffffff;
    --primary: #000000;
    --secondary: #00010a;
    --0: #fafafa; /* Used for things, such as border color for buttons, etc */
    --text: green;
}
```

Refer to `assets/global.css` file for existing rules.

</details>

# Maintenance-related stuffs

## Updating

Note: this guide works only if you install from source. If you used docker, you could probably do the 1st step (pulling the code), stop the container (`docker container stop soundcloak`), build it (`docker compose build`) and start it again. (`docker compose up -d`)

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

5. Update hls.js:

```sh
npm i
```

6. Build binary:

```sh
./build
```

7. Run it:

```sh
./main
```

Congratulations! You have succesfully updated your soundcloak.

</details>
