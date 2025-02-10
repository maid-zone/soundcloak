# Setup

## Docker

You need to have [Docker](https://docker.com) and [Git](https://git-scm.com) installed.

1. Clone this repository:

```sh
git clone https://git.maid.zone/stuff/soundcloak
```

2. Go into the cloned repository:

```sh
cd soundcloak
```

3. Make a copy of the example compose.yaml file:

```sh
cp compose.example.yaml compose.yaml
```

Make adjustments as needed.

4. *Optional.* Edit config:

Refer to [Configuration guide](#configuration-guide) for configuration information. Can be configured from environment variables or JSON file.

5. Run the container:

```sh
docker compose up -d
```

You might need to use `sudo` if you aren't running as root.

Use `docker-compose` instead of `docker compose` if that fails.

Soundcloak will now be up at `127.0.0.1:4664` (or the address you specified in your config). I recommend you run it through a reverse proxy (caddy, nginx, etc.)

## Regular method

** Not recommended for deployment. **

Refer to the [developer guide](DEV_GUIDE.md#setup)

# Updating your instance

## Docker

1. Retrieve the latest code:

```sh
git fetch origin
git pull
```

2. Stop the container:

```sh
docker container stop soundcloak
```

3. Build the container with updated source code:

```sh
docker compose build
```

4. Start the container:

```sh
docker compose up -d
```

Use `docker-compose` instead of `docker compose` if that fails.

## Regular method

Refer to the [developer guide](DEV_GUIDE.md#updating-your-local-setup)

# Configuration/customization

## Configuration guide

<details>
<summary>Click to view</summary>

You can only configure in one of the two ways:

- Using config file (`soundcloak.json` in current directory // your own path and filename)
- Using environment variables (`SOUNDCLOAK_CONFIG` must be set to `FROM_ENV`!)

Some notes:

- When specifying time, specify it in seconds.


| JSON key                | Environment variable       | Default value                                                                                                                                                                                                                                            | Description                                                                                                                                                                                                                                                                                                                                                         |
| ------------------------- | ---------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| None                    | SOUNDCLOAK_CONFIG          | soundcloak.json                                                                                                                                                                                                                                          | File to load soundcloak config from. If set to `FROM_ENV`, soundcloak loads the config from environment variables.                                                                                                                                                                                                                                                   |
| GetWebProfiles          | GET_WEB_PROFILES           | true                                                                                                                                                                                                                                                     | Retrieve links users set in their profile (social media, website, etc)                                                                                                                                                                                                                                                                                              |
| DefaultPreferences      | DEFAULT_PREFERENCES        | {"Player": "hls", "ProxyStreams": false, "FullyPreloadTrack": false, "ProxyImages": false, "ParseDescriptions": true, "AutoplayNextTrack": false, "DefaultAutoplayMode": "normal", "HLSAudio": "mpeg", "RestreamAudio": "mpeg", "DownloadAudio": "mpeg", "ShowAudio": false, "SearchSuggestions": false, "DynamicLoadComments": false} | see /_/preferences page. [Read more](PREFERENCES.md)                                                                                                                                                                                                                                                                                                                 |
| ProxyImages             | PROXY_IMAGES               | false                                                                                                                                                                                                                                                    | Enables proxying of images (user avatars, track covers etc)                                                                                                                                                                                                                                                                                                         |
| ImageCacheControl       | IMAGE_CACHE_CONTROL        | max-age=600, public, immutable                                                                                                                                                                                                                           | [Cache-Control](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control) header value for proxied images. Cached for 10 minutes by default.                                                                                                                                                                                                         |
| ProxyStreams            | PROXY_STREAMS              | false                                                                                                                                                                                                                                                    | Enables proxying of song parts and hls playlist files                                                                                                                                                                                                                                                                                                               |
| Restream                | RESTREAM                   | false                                                                                                                                                                                                                                                    | Enables Restream Player in settings and the /_/restream/:author/:track endpoint. This player can be used without JavaScript. Restream also enables the button for downloading songs.                                                                                                                                                                                |
| RestreamCacheControl    | RESTREAM_CACHE_CONTROL     | max-age=3600, public, immutable                                                                                                                                                                                                                          | [Cache-Control](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control) header value for restreamed songs. Cached for 1 hour by default.                                                                                                                                                                                                           |
| ClientIDTTL             | CLIENT_ID_TTL              | 4 hours                                                                                                                                                                                                                                                  | Time until ClientID cache expires. ClientID is used for authenticating with SoundCloud API                                                                                                                                                                                                                                                                          |
| UserTTL                 | USER_TTL                   | 20 minutes                                                                                                                                                                                                                                               | Time until User profile cache expires                                                                                                                                                                                                                                                                                                                               |
| UserCacheCleanDelay     | USER_CACHE_CLEAN_DELAY     | 5 minutes                                                                                                                                                                                                                                                | Time between each cleanup of the cache (to remove expired users)                                                                                                                                                                                                                                                                                                    |
| TrackTTL                | TRACK_TTL                  | 20 minutes                                                                                                                                                                                                                                               | Time until Track data cache expires                                                                                                                                                                                                                                                                                                                                 |
| TrackCacheCleanDelay    | TRACK_CACHE_CLEAN_DELAY    | 5 minutes                                                                                                                                                                                                                                                | Time between each cleanup of the cache (to remove expired tracks)                                                                                                                                                                                                                                                                                                   |
| PlaylistTTL             | PLAYLIST_TTL               | 20 minutes                                                                                                                                                                                                                                               | Time until Playlist data cache expires                                                                                                                                                                                                                                                                                                                              |
| PlaylistCacheCleanDelay | PLAYLIST_CACHE_CLEAN_DELAY | 5 minutes                                                                                                                                                                                                                                                | Time between each cleanup of the cache (to remove expired playlists)                                                                                                                                                                                                                                                                                                |
| UserAgent               | USER_AGENT                 | Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.3                                                                                                                                           | User-Agent header used for requests to SoundCloud                                                                                                                                                                                                                                                                                                                   |
| DNSCacheTTL             | DNS_CACHE_TTL              | 60 minutes                                                                                                                                                                                                                                               | Time until DNS cache expires                                                                                                                                                                                                                                                                                                                                        |
| Network                 | NETWORK                    | tcp4                                                                                                                                                                                                                                                     | Network to listen on. Can be tcp4, tcp6 or unix                                                                                                                                                                                                                                                                                                                     |
| Addr                    | ADDR                       | :4664                                                                                                                                                                                                                                                    | Address and port for soundcloak to listen on                                                                                                                                                                                                                                                                                                                        |
| Prefork                 | PREFORK                    | false                                                                                                                                                                                                                                                    | Run multiple instances of soundcloak locally to be able to handle more requests. Each one will be a separate process, so they will have separate cache.                                                                                                                                                                                                             |
| TrustedProxyCheck       | TRUSTED_PROXY_CHECK        | true                                                                                                                                                                                                                                                     | Use X-Forwarded-* headers if IP is in TrustedProxies list. When disabled, those headers will blindly be used.                                                                                                                                                                                                                                                       |
| TrustedProxies          | TRUSTED_PROXIES            | []                                                                                                                                                                                                                                                       | List of IPs or IP ranges of trusted proxies                                                                                                                                                                                                                                                                                                                         |
| CodegenConfig           | CODEGEN_CONFIG             | false                                                                                                                                                                                                                                                    | Highly recommended to enable. Embeds the config into the binary, which helps reduce size if you aren't using certain features and generally optimize the binary better. Keep in mind that you will have to rebuild the image/binary each time you want to change config. (Note: you need to run `soundcloakctl config codegen` or use docker, as it runs it for you) |
| EmbedFiles              | EMBED_FILES                | false                                                                                                                                                                                                                                                    | Embed files into the binary. Keep in mind that you will have to rebuild the image/binary each time  static files are changed (e.g. custom instance files)                                                                                                                                                                                                           |

</details>

## Tinkering with the frontend

<details>
<summary>Click to view</summary>

I will mainly talk about the static files here. Maybe about the templates too in the future

The static files are stored in `static/assets` folder

### Overriding files

1. Create a folder named `instance` in `static` folder
2. Create a file with the same name as the one you want to override
3. Put whatever you want there

### Basic theming

1. Create `instance.css` file in the `static/instance` folder
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

Refer to `static/assets/global.css` file for existing rules.

</details>

# Instance list

To get listed on [the instance list](https://maid.zone/soundcloak/instances.html), create a discussion with some information about your instance, or [reach out privately](https://laptopc.at)

Basic rules:

1. Do not collect user information (either yourself, or by including 3rd party tooling which does that)
2. If you are modifying the source code, publish those changes somewhere. Even if it's just static files, it would be best to publish those changes somewhere.

Also, keep in mind that the instance list will periodically hit the `/_/info` endpoint on your instance (usually each 10 minutes) in order to display the instance settings. If you do not want this to happen, state it in your discussion/message, and I will exclude your instance from this checking.

The source code powering the instance list can be found [here](https://git.maid.zone/stuff/soundcloak-instances)
