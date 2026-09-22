var comm = document.getElementById('comments');
var in_flight = false;
function comments(self) {
    if (in_flight) return; 
    var xhr = new XMLHttpRequest();
    xhr.open('GET', '/_/partials/comments/'+self.getAttribute('data-id')+self.getAttribute('href'), true);
    xhr.onerror = function(e) {
        alert('Something went wrong. Check console');
        console.error(e);
        in_flight = false;
    }
    xhr.onload = function() {
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
        comm.querySelectorAll('.link[data-timestamp]').forEach(function(el) {
			el.onclick = function() {
                if (!audio) var audio = document.getElementById("track");
				audio.currentTime = el.dataset.timestamp / 1000;
				audio.play();
			};
		});
        in_flight = false;
    }
    in_flight = true;
    xhr.send();
}