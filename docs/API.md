# Check enabled features

Just go to `/_/info` endpoint

# API

To make use of it, instance must have API enabled of course. All responses are in JSON format if the request is successful. Errors are just returned as plaintext. Currently, there is few functionality present. If you are working on some cool project and wanna have more functionality here, let me know

## Paginated endpoints

Those endpoints return an object with properties:
- `collection`: List of the results on this page
- `total_results`: total results duh, maybe 0 on some endpoints
- `next_href`: Link to next page

To go to next page, take the `next_href`, strip away everything until `?`, and pass that as `pagination` query parameter. Note that not all endpoints may support going to next page.

You can also use `pagination` parameter to pass raw arguments into soundcloud api (for example, more advanced search filters, different initial result limit, etc)

## GET `/_/api/search`

Search for users, tracks or playlists. Query parameters are:
- `q`: the query
- `type`: `users`, `tracks`, or `playlists`. Required
- `pagination`: [Read above](#paginated-endpoints)

For example: `/_/api/search?q=test&type=tracks` to search for `tracks` named `test`

## GET `/_/api/track/:id`

Get track by ID.

For example: `/_/api/track/2014143543` to get track with ID `2014143543`

## GET `/_/api/track/:id/related`

Get related tracks by ID. Pagination is supported here. Initial request returns upto 20 tracks

For example: `/_/api/track/2014143543/related` to get tracks related to track with ID `2014143543`

## GET `/_/api/tracks`

Get tracks by ID in bulk. Pass the IDs comma-separated as `ids` query parameter. You can't request more than 50 tracks at once. The result is a list, which only contains the tracks which were successfully resolved

For example: `/_/api/tracks?ids=2014143543,476907846`. This will only return one track, since 2nd ID is not a track ID

## GET `/_/api/playlistByPermalink/:author/sets/:playlist`

Get playlist by permalinks. 

For example: `/_/api/playlistByPermalink/lucybedroque/sets/unmusique` to get `unmusique` playlist from `lucybedroque`

## GET `/_/api/playlistByPermalink/:author/sets/:playlist/tracks`

Get list of track IDs in playlist.

For example: `/_/api/playlistByPermalink/lucybedroque/sets/unmusique/tracks` to get all IDs of the tracks in playlist `unmusique` from `lucybedroque`

# Other automation

Doesn't require API to be enabled

## GET `/_/restream/:author/:track`

Restream must be enabled in the instance. This endpoint can be used to download or stream tracks. Query parameters are:
- `metadata`: `true` or `false`. If `true`, soundcloak will inject metadata (author, track cover, track title, etc) into the audio file, but this may take a little bit more time
- `audio`: `best`, `aac`, `opus`, or `mpeg`. [Read more here](AUDIO_PRESETS.md)

Restream converts the HLS playlist to an audio file on the fly serverside, optionally adding metadata. Please note that when `audio` is `opus` and `metadata` is `true`, it's not done on the fly, as metadata injection is a bit tricky there.

For example: `/_/restream/lucybedroque/speakers?metadata=true&audio=opus` to get the `opus` audio with `metadata` for song `speakers` by author `lucybedroque`

## GET `/_/searchSuggestions`

Pass your query as `q` query parameter

For example: `/_/searchSuggestions?q=hi` to get search suggestions for `hi`

## GET `/_/proxy/images`

ProxyImages must be enabled in the instance. Put image url into `url` query parameter. Of course, this only proxies images from soundcloud cdn

