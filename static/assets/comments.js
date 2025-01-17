var comm = document.getElementById('comments');
function comments(self) {
    var xhr = new XMLHttpRequest();
    xhr.open('GET', '/_/partials/comments/'+self.getAttribute('data-id')+self.getAttribute('href'), true);
    xhr.onerror = function(e) {
        alert('Something went wrong. Check console');
        console.error(e);
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
        } else {
            self.setAttribute('href', next);
        }
        self.textContent = 'more comments';
    }
    xhr.send();
}