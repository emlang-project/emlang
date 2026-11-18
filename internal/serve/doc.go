// Package serve runs the live-reload server of emlang diagram --serve.
//
// The server renders the diagram of one file and serves it as a full HTML
// page. A goroutine checks the file's modification time every 500 ms and
// renders it again when it changes. The page asks /hash for the hash of
// the current HTML every second and reloads when it differs. Polling on
// both sides keeps the package on the standard library alone.
//
// A file that fails to parse keeps the last good diagram on screen. It is
// rendered again on every check, since an editor may still be writing it,
// but each error is printed on stderr only once. Ctrl+C shuts the server
// down.
//
// With diagram.serve.minimap, the page also shows a minimap in its top
// right corner. A script draws one box per element in a canvas, with the
// position the browser gave the element and its background color, so the
// colors set in diagram.css show up too. The page holds the diagram only
// once, so its ids and anchor names stay unique. The minimap is drawn
// again on scroll, on resize (browser zoom included) and when the body
// changes size. It is hidden while the whole diagram fits in the window.
package serve
