# Preferences


| Name                             | Key                 | Default                                                                | Possible values               | Description                                                                                                                                                                                                              |
| :--------------------------------- | --------------------- | ------------------------------------------------------------------------ | ------------------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Parse descriptions                   | ParseDescriptions          | true                                                                   | true, false                   | Turn @mentions, external links (https://example.org) and emails (hello@example.org) inside descriptions into clickable links                                                                                             |
| Show current audio                   | ShowAudio                  | false                                                                  | true, false                   | Show what [audio preset](AUDIO_PRESETS.md) is being streamed below the audio player                                                                                                                                       |
| Proxy images                         | ProxyImages                | same as ProxyImages in backend config                                  | true, false                   | Proxy images through the backend. ProxyImages must be enabled on the backend                                                                                                                                             |
| Download audio                       | DownloadAudio              | "mpeg"                                                                 | "mpeg", "opus", "aac", "best" | What [audio preset](AUDIO_PRESETS.md) should be loaded when downloading audio with metadata. Restream must be enabled on the backend                                                                                      |
| Autoplay next track in playlists     | AutoplayNextTrack          | false                                                                  | true, false                   | Automatically start playlist playback when you open a track from the playlist. Requires JS                                                                                                                               |
| Default autoplay mode (in playlists) | DefaultAutoplayMode        | "normal"                                                               | "normal", "random"            | Default mode for playlist autoplay. Normal - play songs in order. Random - play random song next                                                                                                                                  |
| Autoplay next related track          | AutoplayNextRelatedTrack   | false                                                                  | true, false                   | Automatically play a related track next. Requires JS
| Fetch search suggestions             | SearchSuggestions          | false                                                                  | true, false                   | Load search suggestions on main page when you type. Requires JS                                                                                                                                                          |
| Dynamically load comments            | DynamicLoadComments        | false                                                                  | true, false                   | Dynamically load track comments, without leaving the page. Requires JS                                                                                                                                                   |
| Player                               | Player                     | "restream" if Restream is enabled in backend config, otherwise - "hls" | "restream", "hls", "none"     | Method used to play the track in the frontend. HLS - requires JavaScript, loads the track in pieces. Restream - works without JavaScript, loads entire track through the backend right away. None - don't play the track |

## Player-specific preferences

### HLS Player

| Name                | Key               | Default                                | Possible values | Description                                                                         |
| :-------------------- | ------------------- | ---------------------------------------- | ----------------- | :------------------------------------------------------------------------------------ |
| Proxy song streams  | ProxyStreams      | same as ProxyStreams in backend config | true, false     | Proxy song streams through the backend. ProxyStreams must be enabled on the backend |
| Fully preload track | FullyPreloadTrack | false                                  | true, false     | Fully load track when the page is loaded (track stream expires in ~5 minutes)       |
| Streaming audio     | HLSAudio          | "mpeg"                                 | "mpeg", "aac"   | What [audio preset](AUDIO_PRESETS.md) should be loaded when streaming audio          |

### Restream Player


| Name            | Key           | Default | Possible values               | Description                                                                |
| :---------------- | --------------- | --------- | ------------------------------- | :--------------------------------------------------------------------------- |
| Streaming audio | RestreamAudio | "mpeg"  | "mpeg", "opus", "aac", "best" | What [audio preset](AUDIO_PRESETS.md) should be loaded when streaming audio |
