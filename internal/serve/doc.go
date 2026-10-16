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
package serve
