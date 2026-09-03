var audio = document.getElementById("track");
if (Hls.isSupported()) {
    var opts = {};
    if (audio.getAttribute("preload") == "yes") {
        opts.maxBufferLength = Infinity;
    }
    var lic = audio.getAttribute("license");
    if (lic) {
        opts.emeEnabled = true;
        opts.drmSystems = {
            "com.widevine.alpha": { "licenseUrl": lic },
            "com.apple.fps": { "licenseUrl": lic, "serverCertificateUrl": lic }
        };
    }
    var hls = new Hls(opts);
    hls.loadSource(audio.src);
    hls.attachMedia(audio);
} else if (!audio.canPlayType("application/vnd.apple.mpegurl")) {
    alert("HLS is not supported! Audio playback will not work.");
}

var volume = audio.getAttribute("volume");
if (volume) {
    audio.volume = parseFloat(volume);
}

var next = audio.getAttribute("data-next");
if (next) {
    audio.addEventListener("ended", function () {
        location = next + "&volume=" + audio.volume;
    });
}
