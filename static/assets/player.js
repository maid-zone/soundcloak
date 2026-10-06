var audio = document.getElementById("track");
var player = audio.getAttribute("player");

function loadHls() {
  try {
    if (Hls.isSupported()) {
      var opts = {};
      if (audio.getAttribute("preload") == "yes") {
        opts.maxBufferLength = Infinity;
      }
      var lic = audio.getAttribute("license");
      if (lic) {
        opts.emeEnabled = true;
        opts.drmSystems = { "com.widevine.alpha": { licenseUrl: lic } };
      }
      var hls = new Hls(opts);
      hls.loadSource(audio.src);
      hls.attachMedia(audio);
    } else {
      return "hls not supported";
    }
  } catch (err) {
    return err;
  }
}

if (player === "hls") {
  loadHls();
} else if (player === "auto") {
  var fallback = audio.src;
  audio.src = "/_/api/hls" + location.pathname;
  var err = loadHls();
  if (err) {
    audio.src = fallback;
    if (fallback.startsWith("/_/api/restream")) {
      player = "restream";
    } else {
      player = "progressive";
    }
  } else {
    player = "hls";
  }
}

var volume = audio.getAttribute("volume");
if (volume) {
  audio.volume = parseFloat(volume);
}

var next = audio.getAttribute("next");
function gonext() {
  location = next + "&volume=" + audio.volume;
}
if (next) {
  audio.addEventListener("ended", gonext);
}

if (audio.hasAttribute("waveform")) {
  var svg = document.querySelector(".waveform");
  var wrapper = document.querySelector(".waveform-wrapper");
  var clip = document.querySelector("#wf-p rect");
  var path2 = svg.querySelector("path").cloneNode(false);
  var timeStart = document.querySelector(".waveform-start");
  var playButton = document.querySelector(".play-button");
  var volumeInput = document.querySelector(".track-volume");
  var repeatButton = document.querySelector(".track-repeat");
  var savedVolume = ~~JSON.parse(localStorage.getItem("volume") || "100");
  volumeInput.value = savedVolume;
  audio.volume = savedVolume / 100;
  volumeInput.addEventListener("input", function () {
    localStorage.setItem("volume", JSON.stringify(~~volumeInput.value));
    audio.volume = ~~volumeInput.value / 100;
  });

  var repeat = JSON.parse(localStorage.getItem("repeat") || "false");
  audio.loop = repeat;
  if (repeat) repeatButton.classList.add("active");
  repeatButton.addEventListener("click", function () {
    let newRepeat = !repeatButton.classList.contains("active");
    audio.loop = newRepeat;
    localStorage.setItem("repeat", JSON.stringify(newRepeat));
    if (newRepeat) {
      repeatButton.classList.add("active");
    } else {
      repeatButton.classList.remove("active");
    }
  });

  function pad(n) {
    n = n + "";
    if (n.length === 1) {
      n = "0" + n;
    }
    return n;
  }
  function formatTime(time) {
    var seconds = time % 60;
    var minutes = ~~(time / 60) % 60;
    var hours = ~~(time / 60 / 60);

    let res = hours ? hours + ":" : "";
    if (hours || minutes) {
      res += hours ? pad(minutes) : minutes;
    } else {
      res += "0";
    }

    res += ":" + pad(seconds);

    return res;
  }

  document.querySelectorAll(".waveform-time").forEach((el) => {
    el.innerHTML = formatTime(el.dataset.time);
  });

  path2.setAttribute("stroke", "var(--accent)");
  path2.setAttribute("clip-path", "url(#wf-p)");
  svg.appendChild(path2);

  if (audio && svg && clip) {
    audio.classList.add("hidden");

    clip.setAttribute("width", "0");
    wrapper.style.cursor = "pointer";

    var dragging = false;
    var raf = null;

    playButton.addEventListener("click", () => {
      if (audio.paused) {
        audio.play();
      } else {
        audio.pause();
      }
    });

    function updateUI() {
      clip.setAttribute("width", (audio.currentTime / audio.duration) * 200);
      timeStart.innerText = formatTime(~~audio.currentTime);
    }

    function updateProgress() {
      if (!dragging && audio.duration) {
        updateUI();
      }
      if (!audio.paused) {
        raf = requestAnimationFrame(updateProgress);
      } else {
        raf = null;
      }
    }

    audio.addEventListener("play", function () {
      if (!raf) raf = requestAnimationFrame(updateProgress);
      playButton.classList.remove("paused");
      playButton.classList.add("playing");
    });

    audio.addEventListener("pause", function () {
      playButton.classList.remove("playing");
      playButton.classList.add("paused");
    });

    // audio.addEventListener("pause", function() {
    //     if (raf) { cancelAnimationFrame(raf) raf = null }
    // })

    audio.addEventListener("seeked", function () {
      if (!dragging && audio.duration) {
        updateUI();
      }
    });

    function seek(e) {
      if (!audio.duration) return;
      var rect = wrapper.getBoundingClientRect();
      var pct = Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width));
      audio.currentTime = pct * audio.duration;
      updateUI();
    }

    wrapper.addEventListener("mousedown", function (e) {
      e.preventDefault();
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

    wrapper.addEventListener(
      "touchstart",
      function (e) {
        dragging = true;
        seek(e.touches[0]);
      },
      { passive: true },
    );

    wrapper.addEventListener(
      "touchmove",
      function (e) {
        if (dragging) seek(e.touches[0]);
      },
      { passive: true },
    );

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
}

function stoppb() {
  audio.removeEventListener("ended", gonext);
  document.getElementById("pbinfo").remove();
}

// please remove this later, keyboard controls would be way better
if (audio.hasAttribute("keepfocus")) {
  audio.onblur = function (e) {
    if (e.target != e.relatedTarget) {
      setTimeout(function () {
        e.target.focus({ preventScroll: true, focusVisible: false });
      });
    }
  };
  audio.focus({ focusVisible: false });
}
