package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/ngharo/sss_idmap_ad2unix/pkg/idmap"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var (
		showVersion = flag.Bool("version", false, "Show version information")
		verbose     = flag.Bool("v", false, "Verbose output")
		domainName  = flag.String("domain-name", "", "Domain name (required for offline mode)")
		domainSID   = flag.String("domain-sid", "", "Domain SID (required for offline mode)")
		rangeMin    = flag.Uint("range-min", 0, "Minimum Unix ID in range (required for offline mode)")
		rangeMax    = flag.Uint("range-max", 0, "Maximum Unix ID in range (required for offline mode)")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] SID\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Convert Windows SID to Unix UID/GID using SSS idmap.\n\n")
		fmt.Fprintf(os.Stderr, "This tool works offline without SSSD by using libsss_idmap directly.\n")
		fmt.Fprintf(os.Stderr, "You must provide domain configuration via command-line flags.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s -domain-name EXAMPLE -domain-sid S-1-5-21-3623811015-3361044348-30300820 \\\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "    -range-min 10000 -range-max 20000 \\\n")
		fmt.Fprintf(os.Stderr, "    S-1-5-21-3623811015-3361044348-30300820-1013\n")
	}

	flag.Parse()

	// Configure logging
	logLevel := slog.LevelInfo
	if *verbose {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	if *showVersion {
		fmt.Printf("sss-idmap version %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	// Validate required flags
	if *domainName == "" || *domainSID == "" || *rangeMin == 0 || *rangeMax == 0 {
		fmt.Fprintf(os.Stderr, "Error: All domain configuration flags are required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	sid := flag.Arg(0)
	slog.Debug("converting SID", "sid", sid)

	// Create domain configuration
	config := idmap.DomainConfig{
		DomainName: *domainName,
		DomainSID:  *domainSID,
		IDRange: idmap.IDRange{
			Min: uint32(*rangeMin),
			Max: uint32(*rangeMax),
		},
	}

	slog.Debug("domain configuration",
		"name", config.DomainName,
		"sid", config.DomainSID,
		"range_min", config.IDRange.Min,
		"range_max", config.IDRange.Max,
	)

	// Create context with domain
	ctx, err := idmap.NewIDMapContextWithDomain(config)
	if err != nil {
		slog.Error("failed to create idmap context", "error", err)
		os.Exit(1)
	}
	defer ctx.Close()

	// Convert SID to Unix ID
	unixID, err := ctx.SIDToUnixID(sid)
	if err != nil {
		slog.Error("failed to convert SID", "sid", sid, "error", err)
		os.Exit(1)
	}

	fmt.Printf("%d\n", unixID)
}
