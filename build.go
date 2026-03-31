package sparrow

// Version is set at build time via -ldflags "-X github.com/sarathsp06/sparrow.Version=..."
// Defaults to "-NOVERSION-" when built without ldflags (e.g. `go run`).
var Version = "-NOVERSION-"
