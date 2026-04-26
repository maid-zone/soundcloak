# API endpoints

<details>
    <summary><h2><code>/_/info</code></h2></summary>

Get details about instance's configuration. Some instances may hide this information, but it's enabled by default. Example response:
```json
{
  "DefaultPreferences": {
    "Player": "restream",
    "ProxyStreams": true,
    "FullyPreloadTrack": false,
    "ProxyImages": true,
    "ParseDescriptions": true,
    "AutoplayNextTrack": false,
    "AutoplayNextRelatedTrack": false,
    "DefaultAutoplayMode": "normal",
    "HLSAudio": "aac",
    "RestreamAudio": "mpeg",
    "DownloadAudio": "mpeg",
    "ShowAudio": false,
    "SearchSuggestions": false,
    "DynamicLoadComments": false,
    "KeepPlayerFocus": false,
    "Waveform": false
  },
  "Commit": "910261d",
  "Repo": "https://git.maid.zone/stuff/soundcloak",
  "ProxyImages": true,
  "ProxyStreams": true,
  "Restream": true,
  "GetWebProfiles": true,
  "EnableAPI": true
}
```

</details>

<details>
    <summary><h2><code>/_/searchSuggestions</code></h2></summary>

Get search suggestions. Query parameters:

* `q`: query to look up
* `format`: set to opensearch if you want it in opensearch JSON format

Example response for `/_/searchSuggestions?q=music`:
```json
[
  "music sounds better with you",
  "music is the answer",
  "musica para dormir",
  "music for soul 6",
  "musica cristiana",
  "musica de amor nunca mais",
  "music for soul 4 -pn ft tdat",
  "musica",
  "music on"
]
```

</details>

<details>
    <summary><h2><code>/_/rss/:user</code></h2></summary>

Generates an [RSS](https://en.wikipedia.org/wiki/RSS) feed for user tracks. Put a username instead of `:user`. Query parameters:

* `proxy_images`: if images should be proxied through instance. Instance must have `ProxyImages` enabled. By default, uses value from preferences

</details>

<details>
    <summary><h2><code>/_/proxy/images</code></h2></summary>

Proxy an image through this instance. Instance must have `ProxyImages` enabled. Query parameters:

* `url`: image URL. Only accepting images from `*.sndcdn.com`

</details>

<details>
    <summary><h2><code>/_/api/hls/:author/:track</code></h2></summary>

Get an [HLS](https://en.wikipedia.org/wiki/HTTP_Live_Streaming) playlist for streaming the track. Query parameters:

* `audio`: force the audio. Can be `aac` or `mpeg`. By default, uses value from preferences
* `redirect`: if should redirect to playlist on soundcloud's CDN. By default `false`
* `redirect_parts`: if should redirect to the track parts on soundcloud's CDN. If `ProxyStreams` is disabled on server, will be `true` by default, otherwise `false`

</details>

<details>
    <summary><h2><code>/_/api/progressive/:author/:track</code></h2></summary>

Get an MP3 file of the track. Query parameters:

* `redirect`: if should redirect to the MP3 on soundcloud's CDN. If `ProxyStreams` is disabled on server, will be `true` by default, otherwise `false`

</details>

<details>
    <summary><h2><code>/_/api/restream/:author/:track</code></h2></summary>

Get an MP3 or Fragmented M4A file of the track, with metadata injected if needed. Instance must have `Restream` enabled for this to work. Query parameters:

* `audio`: force the audio. Can be `aac` or `mpeg`. By default, uses value from preferences
* `metadata`: if should inject metadata. By default `false`. Metadata values are taken from track on soundcloud
* `title`: override title in metadata
* `genre`: override genre in metadata
* `author`: override author in metadata

</details>

<details>
    <summary><h2><code>/_/api/v2/...</code></h2></summary>

Proxy for soundcloud's `api-v2.soundcloud.com`. Instance must have `EnableAPI` enabled. Automatically adds latest `client_id` value to your requests, so you don't need to. Only GET requests are allowed, your supplied headers and body get ignored. List of allowed endpoints:

* `/resolve?url=...` (for getting info from url like `https://soundcloud.com/:user/:track`)
* `/charts/selections`
* `/users/...`
* `/tracks/...`
* `/search/...`
* `/playlists/...`
* `/featured_tracks/...`


So, for example, request like `https://api-v2.soundcloud.com/users/420953284?client_id=tkIWLs4MIowq7bCXP80TOwx6DnDa7UPc` will be turned into `<instance>/_/api/v2/users/420953284`. For more information about what endpoints there are, you can look at soundcloud's official frontend using devtools or read soundcloak/other tools source code. [Here is a big list of endpoints, extracted from the official frontend code](https://fs.maid.zone/sc_endpoints.json). Also keep in mind that there is no caching on this proxy, please cache things on your own

</details>

# Notes about API

Keep in mind that this is unofficial, using reversed not-public API, probably not compliant with any terms of service that soundcloud have. Soundcloud likes to break things like this once in a while, but I try to keep it working. If you wanna use this for your application, it would be great to host your own soundcloak server, or to spread traffic across the [public](https://maid.zone/soundcloak/instances.html) ones *(maybe you could also support the people running them)*

## Audio streaming

Soundcloud offers [two audio presets](docs/AUDIO_PRESETS.md). There is multiple methods available for streaming audio, each with it's own behavior:

* HLS (`/_/api/hls/:author/:track`)

[HLS](https://en.wikipedia.org/wiki/HTTP_Live_Streaming) breaks down an audio into multiple smaller files. This let's you load audio parts on demand, instead of start to finish. You can stream both MP3 and AAC audio with this, but you usually need some program/library to handle the streaming. For web, there is [hls.js](https://github.com/video-dev/hls.js). If you use `?redirect=true` option, keep in mind that this playlist link and parts inside it will automatically expire after track duration + 105s. Otherwise, soundcloak automatically handles renewing the audio playlist.

* Progressive (`/_/api/progressive/:author/:track`)

This only lets you stream tracks as mp3 128kb/s file. If you use `?redirect=true` option, keep in mind that this file link will automatically expire after track duration + 105s. Otherwise, soundcloak automatically handles renewing the audio file

* Restream (`/_/api/restream/:author/:track`)

This combines both HLS (automatically converting to regular audio file) and Progressive methods, and also adds metadata injection on the fly.

## Old API

Currently, it's still all working, but I plan to remove it sometime later, so please migrate everything to the new methods. Also `/_/restream/...` has been redirected to `/_/api/restream/...`
