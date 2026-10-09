(() => {
  function init() {
    var searchSuggestions = document.getElementById("search-suggestions");
    var input = document.getElementById("q");
    var timeout;
    var activeRequest;
    var generation = 0;
    var cache = new Map();

    var signal = window.soundcloakPageSignal;

    signal?.addEventListener(
      "abort",
      function () {
        clearTimeout(timeout);
        activeRequest?.abort();
      },
      { once: true },
    );

    function getSuggestions() {
      if (!input.value.trim()) {
        if (!searchSuggestions) return;
        searchSuggestions.style.display = "none";
        return;
      }

      var xhr = new XMLHttpRequest();
      activeRequest = xhr;

      var query = input.value;
      var id = generation;

      xhr.open(
        "GET",
        "/_/searchSuggestions?q=" + encodeURIComponent(input.value),
        true,
      );

      xhr.onload = function () {
        if (signal?.aborted || id !== generation) return;

        try {
          var cloned = searchSuggestions.cloneNode(false);

          if (xhr.status !== 200 && !cache.has(query)) return;
          var data = cache.get(query) || JSON.parse(xhr.responseText);

          if (!Array.isArray(data)) return;

          cache.set(query, data);

          if (cache.size > 30) {
            cache.delete(cache.keys().next().value);
          }

          if (data.length == 0) {
            searchSuggestions.style.display = "none";
            return;
          }

          for (var i = 0; i < data.length; i++) {
            var e = document.createElement("li");
            e.textContent = data[i];
            e.onclick = function () {
              input.value = this.textContent;
              input.form.requestSubmit();
              searchSuggestions.style.display = "none";
            };
            cloned.appendChild(e);
          }

          searchSuggestions.parentNode.replaceChild(cloned, searchSuggestions);
          searchSuggestions = cloned;
          searchSuggestions.style.display = "block";
        } catch {
          searchSuggestions.style.display = "none";
        }
      };

      xhr.onerror = function () {
        searchSuggestions.style.display = "none";
      };

      cache.has(query) ? xhr.onload() : xhr.send();
    }

    input.addEventListener(
      "input",
      function () {
        generation++;
        activeRequest?.abort();
        searchSuggestions.style.display = "none";
        if (timeout) {
          clearTimeout(timeout);
        }
        timeout = setTimeout(getSuggestions, 250);
      },
      { signal },
    );
  }
  (window.soundcloakInitializers ||= new Map()).set(
    document.currentScript.src,
    init,
  );
  init();
})();
