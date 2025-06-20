# Check enabled features

Just go to `/_/info` endpoint

# API

To make use of it, instance must have API enabled of course. Currently, there is only one functionality present. If you are working on some cool project and wanna have more functionality here, let me know

## Searching

You can search with endpoint `/_/api/search`. Query parameters are:
- `q`: the query
- `type`: `users`, `tracks`, or `playlists`. Required
- `pagination`: raw parameters to pass into soundcloud's api

The response is in JSON format. To go to next page, take the `next_href` from result, strip away everything until `?`, and pass that as `pagination` parameter.

For example: `/_/api/search?q=test&type=tracks` to search for `tracks` named `test`

# Other automation

Doesn't require API to be enabled

## Download songs

Restream must be enabled in the instance. The endpoint is `/_/restream/<author permalink>/<track permalink>`. Query parameters are:
- `metadata`: `true` or `false`. If `true`, soundcloak will inject metadata (author, track cover, track title, etc) into the audio file, but this may take a little bit more time
- `audio`: `best`, `aac`, `opus`, or `mpeg`. [Read more here](AUDIO_PRESETS.md)

For example: `/_/restream/lucybedroque/speakers?metadata=true&audio=opus` to get the `opus` audio with `metadata` for song `speakers` by author `lucybedroque`

## Get search suggestions

The endpoint is `/_/searchSuggestions`. Query parameters are:
- `q`: the query

The response is a list of search suggestions as strings in JSON format.

For example: `/_/searchSuggestions?q=hi` to get search suggestions for `hi`

## Proxy images

ProxyImages must be enabled in the instance. 

Endpoint for images: `/_/proxy/images`. Put image url into `url` query parameter. Of course, this only proxies images from soundcloud cdn

