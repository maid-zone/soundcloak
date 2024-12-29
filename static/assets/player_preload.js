var audio = document.getElementById('track');
if (Hls.isSupported()) {
    var hls = new Hls({ maxBufferLength: Infinity });
    hls.loadSource(audio.src);
    hls.attachMedia(audio);

    var volume = audio.getAttribute('volume');
    if (volume) {
        audio.volume = parseFloat(volume); 
    }
} else if (!audio.canPlayType('application/vnd.apple.mpegurl')) {
    alert('HLS is not supported! Audio playback will not work.');
}

var next = audio.getAttribute('data-next');
if (next) {
    audio.addEventListener('ended', function() {
        location = next + '&volume=' + audio.volume;
    });
}