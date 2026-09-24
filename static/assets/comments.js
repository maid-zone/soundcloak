var comm = document.getElementById('comments');
var in_flight = false;
if (!audio) {
    var audio = document.getElementById("track");
}
function comments(self) {
    if (in_flight) return;
    var xhr = new XMLHttpRequest();
    xhr.open('GET', '/_/partials/comments/' + self.getAttribute('data-id') + self.getAttribute('href'), true);
    xhr.onerror = function (e) {
        alert('Something went wrong. Check console');
        console.error(e);
        in_flight = false;
    }
    xhr.onload = function () {
        if (xhr.status != 200) {
            alert(xhr.responseText);
            return;
        }

        comm.innerHTML += xhr.responseText;
        var next = xhr.getResponseHeader('next');
        if (next == 'done') {
            self.remove();
            return;
        }
        self.setAttribute('href', next);
        self.textContent = 'more comments';
        comm.querySelectorAll('.link[data-timestamp]').forEach(function (el) {
            el.onclick = function (event) {
                event.preventDefault();
                var ts = el.dataset.timestamp / 1000;
                if (window.Hls) {
                    audio.play();
                    audio.currentTime = ts;
                } else {
                    // fix for restream aac on firefox as it's fmp4 but doesn't work on chrome \(^-^)/
                    if (audio.duration < ts) {
                        audio.ondurationchange = function () {
                            if (audio.currentTime < ts) {
                                audio.currentTime = ts;
                            }
                            if (audio.currentTime >= ts) {
                                audio.ondurationchange = undefined;
                                audio.currentTime = ts;
                                audio.play();
                            }
                        }
                        audio.currentTime = ts;
                    } else {
                        audio.currentTime = ts;
                        audio.play();
                    }
                }
            };
        });
        in_flight = false;
    }
    in_flight = true;
    xhr.send();
}