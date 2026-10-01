//go:build !js

package device

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A browser grants fullscreen only to a page the player has just pressed a
// key in, clicked or touched, and the backend used to ask only inside the
// handler of one of those. A game reads the key in its next update, after the
// handler is over, so a settings screen's "Fullscreen" row switched nothing
// until the player pressed another key; leaving fullscreen, which needs no key
// at all, waited for one too. And once the player had left fullscreen with Esc,
// the game's next ask was taken for the one already made, and did nothing.
//
// This check puts the real web.js in a page in a browser with no window and
// plays the browser's part by hand: a browser with no window has no screen to
// fill, and a key a page dispatches is not one the browser counts as the
// player's. The page says whether the player has just pressed a key, as
// navigator.userActivation does, shows fullscreen whatever the backend asks to
// show, and can refuse.
//
// It is skipped where there is no browser to run it in.

// webFullscreenStep is what the page saw after one step.
type webFullscreenStep struct {
	What       string `json:"what"`
	Fullscreen bool   `json:"fullscreen"` // the page is shown fullscreen
	Lost       bool   `json:"lost"`       // the backend says the player left it
}

type webFullscreenResult struct {
	Failed string              `json:"failed"`
	Steps  []webFullscreenStep `json:"steps"`
}

const webFullscreenPage = `<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><title>Fullscreen</title>
<style>
	html, body { margin: 0; height: 100%; overflow: hidden; background: #000; }
	#game { display: block; width: 640px; height: 360px; outline: none; }
</style>
</head>
<body>
<canvas id="game"></canvas>
<script>
	// The browser's part. active is whether the player pressed a key, clicked
	// or touched a moment ago; refuse makes the next asks fail, as a browser
	// does when it doesn't count one.
	var shown = null, active = false, refuse = false;
	Object.defineProperty(navigator, 'userActivation', {
		configurable: true, get: function () { return { isActive: active, hasBeenActive: true }; },
	});
	Object.defineProperty(document, 'fullscreenElement', {
		configurable: true, get: function () { return shown; },
	});
	function changed() { document.dispatchEvent(new Event('fullscreenchange')); }
	Element.prototype.requestFullscreen = function () {
		if (refuse) return Promise.reject(new TypeError('Permissions check failed'));
		shown = this;
		changed();
		return Promise.resolve();
	};
	document.exitFullscreen = function () { shown = null; changed(); return Promise.resolve(); };
</script>
<script src="golib.js"></script>
<script>
	const steps = [];
	function report(result) { fetch('/result', { method: 'POST', body: JSON.stringify(result) }); }
	function note(what) { steps.push({ what: what, fullscreen: shown !== null, lost: window.golib.fullscreenLost() }); }
	function key() {
		window.dispatchEvent(new KeyboardEvent('keydown', { code: 'KeyX', key: 'x' }));
		window.dispatchEvent(new KeyboardEvent('keyup', { code: 'KeyX', key: 'x' }));
	}
	// A refused ask is answered a moment later, as a browser answers.
	function later(then) { setTimeout(then, 50); }

	try {
		window.golib.open(640, 360, 'fullscreen check');

		window.golib.setFullscreen(true);
		note('the game asks for fullscreen at the start, with no key pressed yet');
		key();
		note('the first key');

		window.golib.setFullscreen(false);
		note('the game leaves fullscreen, with no key');

		active = true;
		window.golib.setFullscreen(true);
		note('the game asks for fullscreen on the key it just read');
		active = false;

		shown = null;
		changed();
		note('the player leaves fullscreen with Esc');
		key();
		note('a key, with the game not asking again');
		window.golib.setFullscreen(true);
		note('the game asks for fullscreen again, with no key a moment ago');
		key();
		note('the next key');

		window.golib.setFullscreen(false);
		refuse = true;
		active = true;
		window.golib.setFullscreen(true);
		later(function () {
			note('the browser refuses an ask on a key');
			refuse = false;
			active = false;
			key();
			note('the next key');
			report({ steps: steps });
		});
	} catch (e) {
		report({ failed: String(e && e.message ? e.message : e) });
	}
</script>
</body></html>
`

func TestTheWebBackendSwitchesFullscreenWhenTheGameAsks(t *testing.T) {
	browser := findWebBrowser()
	if browser == "" {
		t.Skip("no browser on this machine to run the check in")
	}
	backend, err := os.ReadFile("web.js")
	if err != nil {
		t.Fatalf("cannot read the web backend: %v", err)
	}

	reported := make(chan webFullscreenResult, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/result", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var result webFullscreenResult
		if err := json.Unmarshal(body, &result); err != nil {
			result.Failed = "the page sent " + string(body)
		}
		select {
		case reported <- result:
		default:
		}
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			io.WriteString(w, webFullscreenPage)
		case "/golib.js":
			w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
			w.Write(backend)
		default:
			http.NotFound(w, r)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	stop, err := startWebBrowser(browser, server.URL+"/")
	if err != nil {
		t.Skipf("cannot start %s with no window: %v", filepath.Base(browser), err)
	}
	defer stop()

	var result webFullscreenResult
	select {
	case result = <-reported:
	case <-time.After(webFocusTimeout):
		t.Fatalf("the page never reported back in %v", webFocusTimeout)
	}
	if strings.Contains(result.Failed, "WebGL 2") {
		t.Skip("this browser has no WebGL 2 with no window: " + result.Failed)
	}
	if result.Failed != "" {
		t.Fatal("the check could not be made: " + result.Failed)
	}

	want := []webFullscreenStep{
		// Nothing the player did yet: the ask waits for the first key.
		{"the game asks for fullscreen at the start, with no key pressed yet", false, false},
		{"the first key", true, false},
		// Leaving needs no key.
		{"the game leaves fullscreen, with no key", false, false},
		// A settings screen switches on the key it reads: the browser still
		// counts that key, so the ask is made at once.
		{"the game asks for fullscreen on the key it just read", true, false},
		// The player leaving is final until the game asks again, and then the
		// ask is a new one.
		{"the player leaves fullscreen with Esc", false, true},
		{"a key, with the game not asking again", false, true},
		{"the game asks for fullscreen again, with no key a moment ago", false, false},
		{"the next key", true, false},
		// A refused ask is made again at the next key.
		{"the browser refuses an ask on a key", false, false},
		{"the next key", true, false},
	}
	if len(result.Steps) != len(want) {
		t.Fatalf("the page went through %d steps, want %d: %+v", len(result.Steps), len(want), result.Steps)
	}
	for i, got := range result.Steps {
		if got != want[i] {
			t.Errorf("after %q: fullscreen %v and lost %v, want fullscreen %v and lost %v",
				got.What, got.Fullscreen, got.Lost, want[i].Fullscreen, want[i].Lost)
		}
	}
}
