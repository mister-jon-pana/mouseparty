package main

// Windows has no counterpart here for most of what native_darwin.go asks of AppKit: the About
// panel is a message box (mac.go), and the rest does nothing.

func nativeAbout(string) {}
func nativeHideOthers()  {}
func nativeShowAll()     {}
func nativeSend(string)  {} // the web view does Cut, Copy, Paste and Select All itself
func nativeKeepFrame()   {}
func nativeBadge(string) {}

// nativeActive: true, as where it can't be told (so no notifications while it's in front)
func nativeActive() bool { return true }
