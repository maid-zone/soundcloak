var audio = document.getElementById('track');
var volume = audio.getAttribute('volume');
if (volume) {
    audio.volume = parseFloat(volume); 
}
var next = audio.getAttribute('data-next');
function gonext() {
    location = next + '&volume=' + audio.volume;
}
if (next) {
    audio.addEventListener('ended', gonext);
}

function stoppb() {
    audio.removeEventListener('ended', gonext);
    document.getElementById('pbinfo').remove();
}