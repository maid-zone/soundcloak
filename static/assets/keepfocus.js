var audio = document.getElementById('track');
audio.onblur = function (e) {
    if (e.target != e.relatedTarget) {
        setTimeout(function() {
            e.target.focus({preventScroll: true, focusVisible: false});
        })
    }
}
audio.focus({focusVisible: false});