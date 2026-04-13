var audio = document.getElementById("track");
var svg = document.querySelector(".waveform");
var clip = document.querySelector("#wf-p rect");
var path2 = svg.querySelector("path").cloneNode(false);
path2.setAttribute("stroke", "var(--accent)");
path2.setAttribute("clip-path", "url(#wf-p)");
svg.appendChild(path2);

if (audio && svg && clip) {
    clip.setAttribute("width", "0");
    svg.style.cursor = "pointer";

    var dragging = false;
    var raf = null;

    function updateProgress() {
        if (!dragging && audio.duration) {
            clip.setAttribute(
                "width",
                (audio.currentTime / audio.duration) * 200,
            );
        }
        if (!audio.paused) {
            raf = requestAnimationFrame(updateProgress);
        } else {
            raf = null;
        }
    }

    audio.addEventListener("play", function () {
        if (!raf) raf = requestAnimationFrame(updateProgress);
    });

    // audio.addEventListener('pause', function() {
    //     if (raf) { cancelAnimationFrame(raf); raf = null; }
    // });

    audio.addEventListener("seeked", function () {
        if (!dragging && audio.duration) {
            clip.setAttribute(
                "width",
                (audio.currentTime / audio.duration) * 200,
            );
        }
    });

    function seek(e) {
        if (!audio.duration) return;
        var rect = svg.getBoundingClientRect();
        var pct = Math.max(
            0,
            Math.min(1, (e.clientX - rect.left) / rect.width),
        );
        audio.currentTime = pct * audio.duration;
        clip.setAttribute("width", pct * 200);
    }

    svg.addEventListener("mousedown", function (e) {
        dragging = true;
        seek(e);
    });

    document.addEventListener("mousemove", function (e) {
        if (dragging) seek(e);
    });

    document.addEventListener("mouseup", function () {
        if (dragging && audio.paused) {
            audio.play();
        }
        dragging = false;
    });

    svg.addEventListener("touchstart", function (e) {
        dragging = true;
        seek(e.touches[0]);
    }, { passive: true });

    svg.addEventListener("touchmove", function (e) {
        if (dragging) seek(e.touches[0]);
    }, { passive: true });

    svg.addEventListener("touchend", function () {
        if (dragging && audio.paused) {
            audio.play();
        }
        dragging = false;
    });

    window.addEventListener("focus", function () {
        if (!raf) raf = requestAnimationFrame(updateProgress);
    });

    // kick off if already playing (e.g. autoplay)
    if (!audio.paused) raf = requestAnimationFrame(updateProgress);
}
