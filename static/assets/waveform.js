var audio = document.getElementById("track");
var svg = document.querySelector(".waveform");
var wrapper = document.querySelector('.waveform-wrapper')
var clip = document.querySelector("#wf-p rect");
var path2 = svg.querySelector("path").cloneNode(false);
var timeStart = document.querySelector('.waveform-start')
const playButton = document.querySelector('.play-button')

function formatTime(time) {
	const seconds = time % 60
	const minutes = ~~(time / 60) % 60
	const hours = ~~(time / 60 / 60)
	const pad = n => (n + '').padStart(2, '0')

	let res = hours ? hours + ':' : ''
	if(hours || minutes) {
		res += hours ? pad(minutes) : minutes
	} else {
		res += '0'
	}

	res += ':' + pad(seconds)

	return res
}

document.querySelectorAll('.waveform-time').forEach(el => {
	el.innerHTML = formatTime(el.dataset.time)
})

path2.setAttribute("stroke", "var(--accent)");
path2.setAttribute("clip-path", "url(#wf-p)");
svg.appendChild(path2);

if (audio && svg && clip) {
	//audio.classList.add('hidden')
	wrapper.classList.remove('hidden')

    clip.setAttribute("width", "0");
    wrapper.style.cursor = "pointer";

    var dragging = false;
    var raf = null;

	playButton.addEventListener('click', () => {
		if(audio.paused) {
			audio.play()
		} else {
			audio.pause()
		}
	})

	function updateUI() {
		clip.setAttribute(
			"width",
			(audio.currentTime / audio.duration) * 200,
		);
		timeStart.innerText = formatTime(~~audio.currentTime)
	}

    function updateProgress() {
        if (!dragging && audio.duration) {
			updateUI()
        }
        if (!audio.paused) {
            raf = requestAnimationFrame(updateProgress);
        } else {
            raf = null;
        }
    }

    audio.addEventListener("play", function () {
        if (!raf) raf = requestAnimationFrame(updateProgress);
		playButton.classList.remove('paused')
		playButton.classList.add('playing')
    });

	audio.addEventListener('pause', function() {
		playButton.classList.remove('playing')
		playButton.classList.add('paused')
	})

    // audio.addEventListener('pause', function() {
    //     if (raf) { cancelAnimationFrame(raf); raf = null; }
    // });

    audio.addEventListener("seeked", function () {
        if (!dragging && audio.duration) {
            updateUI()
        }
    });

    function seek(e) {
        if (!audio.duration) return;
        var rect = wrapper.getBoundingClientRect();
        var pct = Math.max(
            0,
            Math.min(1, (e.clientX - rect.left) / rect.width),
        );
        audio.currentTime = pct * audio.duration;
        updateUI()
    }

    wrapper.addEventListener("mousedown", function (e) {
		e.preventDefault()
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

    wrapper.addEventListener("touchstart", function (e) {
        dragging = true;
        seek(e.touches[0]);
    }, { passive: true });

    wrapper.addEventListener("touchmove", function (e) {
        if (dragging) seek(e.touches[0]);
    }, { passive: true });

    wrapper.addEventListener("touchend", function () {
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
