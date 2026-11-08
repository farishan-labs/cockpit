package version

// Version is the cockpit build version. Overridden at build time via -ldflags.
var Version = "0.1.0-dev"

// Commit is the git commit. Overridden at build time.
var Commit = "none"

// Date is the build date. Overridden at build time.
var Date = "unknown"
