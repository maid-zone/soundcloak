var audio = document.getElementById('track');
var volume = audio.getAttribute('volume');
if (volume) {
    audio.volume = parseFloat(volume); 
}
var next = audio.getAttribute('data-next');
if (next) {
    audio.addEventListener('ended', function() {
        location = next + '&volume=' + audio.volume;
    });
}