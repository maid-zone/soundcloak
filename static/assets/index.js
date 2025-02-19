var searchSuggestions = document.getElementById('search-suggestions');
var input = document.getElementById('q');
var form = document.querySelector('form[action="/search"]');
var timeout;

function getSuggestions() {
    if (input.value == 0) {
        searchSuggestions.style.display = 'none';
        return;
    }

    var xhr = new XMLHttpRequest();
    xhr.open('GET', '/_/searchSuggestions?q=' + encodeURIComponent(input.value), true);
    xhr.onload = function () {
        try {
            var cloned = searchSuggestions.cloneNode(false);

            var data = JSON.parse(xhr.responseText);
            if (data.length == 0) {
                searchSuggestions.style.display = 'none';
                return;
            }

            for (var i = 0; i < data.length; i++) {
                var e = document.createElement('li');
                e.textContent = data[i];
                e.onclick = function () {
                    input.value = this.textContent;
                    searchSuggestions.style.display = 'none';
                    form.submit();
                }
                cloned.appendChild(e);
            }

            searchSuggestions.parentNode.replaceChild(cloned, searchSuggestions);
            searchSuggestions = cloned;
            searchSuggestions.style.display = 'flex';
        } catch {
            searchSuggestions.style.display = 'none';
        }
    }

    xhr.onerror = function () {
        searchSuggestions.style.display = 'none';
    }

    xhr.send();
}

input.addEventListener('input', function () {
    if (!timeout) {
        timeout = setTimeout(getSuggestions, 250);
    } else {
        clearTimeout(timeout);
        timeout = setTimeout(getSuggestions, 250);
    }
});