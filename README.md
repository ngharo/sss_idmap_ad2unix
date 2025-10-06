# sss_idmap_ad2unix

A Go library and command-line tool for converting Windows Active Directory SIDs to Unix UIDs/GIDs using the SSS (System Security Services) idmap library.

## Requirements

- Go 1.23.0 or later
- `libsss-idmap-dev` (Debian/Ubuntu) or `sssd-devel` (RHEL/Fedora)
- pkg-config

### Installing Dependencies

**Debian/Ubuntu:**
```bash
sudo apt-get install libsss-idmap-dev pkg-config
```

**RHEL/Fedora/CentOS:**
```bash
sudo dnf install sssd-devel pkg-config
```

## Installation

### Install the CLI Tool

```bash
go install github.com/ngharo/sss_idmap_ad2unix/cmd/sss-idmap@latest
```

### Use as a Library

```bash
go get github.com/ngharo/sss_idmap_ad2unix/pkg/idmap
```

## Usage

### Command-Line Tool

The CLI tool requires you to specify the domain configuration:

```bash
# Convert a SID to Unix ID
sss-idmap \
  -domain-name EXAMPLE \
  -domain-sid S-1-5-21-3623811015-3361044348-30300820 \
  -range-min 10000 \
  -range-max 20000 \
  S-1-5-21-3623811015-3361044348-30300820-1013

# Output: 11013

# Verbose output for debugging
sss-idmap -v \
  -domain-name EXAMPLE \
  -domain-sid S-1-5-21-3623811015-3361044348-30300820 \
  -range-min 10000 \
  -range-max 20000 \
  S-1-5-21-3623811015-3361044348-30300820-1013

# Show version
sss-idmap -version
```

**Required Flags:**
- `-domain-name`: Name of your AD domain (e.g., "EXAMPLE", "CONTOSO")
- `-domain-sid`: The domain's SID (the part before the RID in user/group SIDs)
- `-range-min`: Minimum Unix UID/GID to allocate
- `-range-max`: Maximum Unix UID/GID to allocate

### As a Go Library

#### Offline Mode with Domain Configuration (Recommended)

```go
package main

import (
    "fmt"
    "log"

    "github.com/ngharo/sss_idmap_ad2unix/pkg/idmap"
)

func main() {
    // Configure your domain and ID range
    config := idmap.DomainConfig{
        DomainName: "EXAMPLE",
        DomainSID:  "S-1-5-21-3623811015-3361044348-30300820",
        IDRange: idmap.IDRange{
            Min: 10000,
            Max: 20000,
        },
    }

    // Create context with domain configuration
    ctx, err := idmap.NewIDMapContextWithDomain(config)
    if err != nil {
        log.Fatal(err)
    }
    defer ctx.Close()

    // Convert SIDs to Unix IDs
    unixID, err := ctx.SIDToUnixID("S-1-5-21-3623811015-3361044348-30300820-1013")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Unix ID: %d\n", unixID)
}
```

#### Adding Multiple Domains

```go
ctx, err := idmap.NewIDMapContext()
if err != nil {
    log.Fatal(err)
}
defer ctx.Close()

// Add first domain
err = ctx.AddDomain(idmap.DomainConfig{
    DomainName: "DOMAIN1",
    DomainSID:  "S-1-5-21-1111111111-2222222222-3333333333",
    IDRange:    idmap.IDRange{Min: 10000, Max: 20000},
})
if err != nil {
    log.Fatal(err)
}

// Add second domain
err = ctx.AddDomain(idmap.DomainConfig{
    DomainName: "DOMAIN2",
    DomainSID:  "S-1-5-21-4444444444-5555555555-6666666666",
    IDRange:    idmap.IDRange{Min: 20001, Max: 30000},
})
if err != nil {
    log.Fatal(err)
}

// Now convert SIDs from either domain
unixID1, _ := ctx.SIDToUnixID("S-1-5-21-1111111111-2222222222-3333333333-1001")
unixID2, _ := ctx.SIDToUnixID("S-1-5-21-4444444444-5555555555-6666666666-2001")
```

### Error Handling

The library provides typed errors for common scenarios:

```go
import (
    "errors"
    "github.com/ngharo/sss_idmap_ad2unix/pkg/idmap"
)

unixID, err := ctx.SIDToUnixID(sid)
if err != nil {
    switch {
    case errors.Is(err, idmap.ErrInvalidSID):
        // Handle invalid SID format
    case errors.Is(err, idmap.ErrNotFound):
        // Handle SID not found (domain not configured)
    case errors.Is(err, idmap.ErrInvalidRange):
        // Handle invalid ID range configuration
    case errors.Is(err, idmap.ErrInternal):
        // Handle internal SSS library errors
    default:
        // Handle other errors
    }
}
```

## Development

### Building

```bash
make build
```

### Testing

```bash
make test
```

### Formatting

```bash
make fmt
```

### Linting

```bash
make lint
```

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
