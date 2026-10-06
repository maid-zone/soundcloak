// @ts-check
function getCookie(/** @type {string} */ name) {
  const cookie = document.cookie
    .split("; ")
    .find((cookie) => cookie.startsWith(`${name}=`));

  return cookie ? decodeURIComponent(cookie.slice(name.length + 1)) : null;
}

class Bookmarks {
  /** @type {Set<string>} */
  #set = new Set(JSON.parse(getCookie("bookmarked-tracks") ?? "[]"));

  add(/** @type {string} */ id) {
    this.#set.add(id);
    this.#save();
  }

  remove(/** @type {string} */ id) {
    this.#set.delete(id);
    this.#save();
  }

  has(/** @type {string} */ id) {
    return this.#set.has(id);
  }

  getAll() {
    return [...this.#set];
  }

  #save() {
    document.cookie = `bookmarked-tracks=${encodeURIComponent(JSON.stringify([...this.#set]))}; path=/; max-age=31536000; samesite=lax`;
  }
}
const bookmarks = new Bookmarks();

const bookmarkButtonSlotEl = document.getElementById("bookmark-button-slot");

const trackId = bookmarkButtonSlotEl?.dataset.trackId;

const bookmarkButtonEl = document.createElement("button");
bookmarkButtonEl.className = "btn";

function rerender() {
  if (!trackId) return;
  const isBookmarked = bookmarks.has(trackId);

  bookmarkButtonEl.onclick = () => {
    if (isBookmarked) {
      bookmarks.remove(trackId);
    } else {
      bookmarks.add(trackId);
    }

    rerender();
  };

  bookmarkButtonEl.textContent = isBookmarked ? "unbookmark" : "bookmark";
}

bookmarkButtonSlotEl?.append(bookmarkButtonEl);

rerender();
