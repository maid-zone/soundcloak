# [Instances List](https://maid.zone/soundcloak/instances.html)

# Viewing instance settings

If the instance is in the [instances list](https://maid.zone/soundcloak/instances.html), then you can view a few of the settings there (ProxyStreams, ProxyImages, Restream)

You can also visit `<instance>/_/info` to view some of the settings. This endpoint is only available if `InstanceInfo` is enabled by the maintainer.

# Preferences

Click the "Preferences" button on the main page, or navigate to `<instance>/_/preferences` to get started.

Instance maintainers can set their own default preferences.

[Refer to here](PREFERENCES.md) for a full list of preferences. If you don't see certain preferences on the page, it means that you have to enable a different preference to configure this one, or this feature has been disabled on the backend.

Those preferences are saved in a cookie.

## Management

Scroll down to the end of the preferences page. There you can see a management tab for preferences. You can export your preferences as a JSON file, import the preferences, or reset them.

# Redirecting from SoundCloud to soundcloak

soundcloak tries to keep the URL schemes same to SoundCloud's, so you can just replace `soundcloud.com` with your instance URL. For short links: `https://on.soundcloud.com/boiKDP46fayYDoVK9` -> `<instance>/on/boiKDP46fayYDoVK9`

To automatically redirect, you can use [LibRedirect](https://libredirect.github.io/) extension. Soundcloak is supported 