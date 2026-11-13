# Mouse Party on Windows

A port of the Mac app so a party can be hosted from a Windows PC (x64). Guests join
exactly as before, in a browser on anything (an iPad, a phone, a smart TV's browser).

Built by `.github/workflows/windows-app.yml` on a push to the `windows` branch:
`MouseParty-windows-x64.zip` is the run's artifact. Unzip it anywhere and open
`MouseParty\MouseParty.exe`. It needs Windows 10 or 11 with the WebView2 runtime (both
have it). The first time, SmartScreen warns (More info > Run anyway) and Windows Firewall
asks about MouseParty and MediaMTX: allow Private networks, so guests on the same Wi-Fi can
watch.

What differs from the Mac app:

- No sandbox. Windows has nothing like Seatbelt, so the viewer site runs inside the app
  (as `no_sandbox` does on a Mac) and MediaMTX, cloudflared and ffmpeg run unboxed. Only
  invite people you trust, and only play files you trust.
- No updates: a new version is a new zip (`selfUpdates`, proc_windows.go).
- No Datasette: its WebAssembly Python is built on a Mac only, so a SQLite database
  dropped in says it can't be shown. Pandoc (documents) is there.
- ffmpeg is BtbN's GPL Windows build (its `latest` tag, so not pinned), not the app's own.
- The menus are Windows-shaped: no Hide/Show All or Window menu, Exit in place of Quit,
  and Cut/Copy/Paste left to the web view. About is a message box. No taskbar badge,
  no notifications, and the window's place isn't remembered.
- Tools stop with the app through a job object (kill on close) instead of the shell
  watchdog; ffmpeg is stopped outright instead of with Ctrl+C.

Layout of the unzipped folder:

    MouseParty\MouseParty.exe
    MouseParty\Resources\bin\       ffmpeg.exe, mediamtx.exe, cloudflared.exe
    MouseParty\Resources\wasm\      pandoc.wasm
    MouseParty\Resources\host_web\  the viewer page (read on every load, as on the Mac)
    MouseParty\Licenses\

The party file is `%APPDATA%\Mouse Party\party.sqlite`; downloads and the WebAssembly
cache are under `%LOCALAPPDATA%\Mouse Party`.
