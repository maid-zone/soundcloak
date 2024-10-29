var audio = document.getElementById('track');
if (Hls.isSupported()) {
    var hls = new Hls({ maxBufferLength: Infinity });
    hls.loadSource(audio.src);
    hls.attachMedia(audio);
} else if (!audio.canPlayType('application/vnd.apple.mpegurl')) {
    alert('HLS is not supported! Audio playback will not work.');
}