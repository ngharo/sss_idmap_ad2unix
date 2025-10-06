package idmap

/*
#cgo pkg-config: sss_idmap
#include <stdlib.h>
#include <sss_idmap.h>
*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"
)

var (
	// ErrInvalidSID indicates that the provided SID string is invalid
	ErrInvalidSID = errors.New("invalid SID format")
	// ErrNotFound indicates that no mapping was found for the given SID
	ErrNotFound = errors.New("SID not found in idmap")
	// ErrInternal indicates an internal error in the SSS library
	ErrInternal = errors.New("internal SSS idmap error")
	// ErrInvalidRange indicates that the provided ID range is invalid
	ErrInvalidRange = errors.New("invalid ID range")
)

// IDRange represents a Unix ID range for SID mapping
type IDRange struct {
	Min uint32
	Max uint32
}

// DomainConfig holds the configuration for a domain's ID mapping
type DomainConfig struct {
	DomainName string
	DomainSID  string
	IDRange    IDRange
}

// IDMapContext wraps the sss_idmap_ctx C structure
type IDMapContext struct {
	ctx *C.struct_sss_idmap_ctx
}

// NewIDMapContext creates a new ID mapping context
func NewIDMapContext() (*IDMapContext, error) {
	var ctx *C.struct_sss_idmap_ctx

	err := C.sss_idmap_init(nil, nil, nil, &ctx)
	if err != C.IDMAP_SUCCESS {
		return nil, fmt.Errorf("%w: failed to initialize idmap context (code: %d)", ErrInternal, err)
	}

	return &IDMapContext{ctx: ctx}, nil
}

// NewIDMapContextWithDomain creates a new ID mapping context with a preconfigured domain
func NewIDMapContextWithDomain(config DomainConfig) (*IDMapContext, error) {
	ctx, err := NewIDMapContext()
	if err != nil {
		return nil, err
	}

	if err := ctx.AddDomain(config); err != nil {
		ctx.Close()
		return nil, err
	}

	return ctx, nil
}

// AddDomain adds a domain configuration to the ID mapping context
func (c *IDMapContext) AddDomain(config DomainConfig) error {
	if c.ctx == nil {
		return fmt.Errorf("%w: context is nil", ErrInternal)
	}

	if config.IDRange.Min >= config.IDRange.Max {
		return fmt.Errorf("%w: min (%d) must be less than max (%d)", ErrInvalidRange, config.IDRange.Min, config.IDRange.Max)
	}

	cDomainName := C.CString(config.DomainName)
	defer C.free(unsafe.Pointer(cDomainName))

	cDomainSID := C.CString(config.DomainSID)
	defer C.free(unsafe.Pointer(cDomainSID))

	cRange := C.struct_sss_idmap_range{
		min: C.uint32_t(config.IDRange.Min),
		max: C.uint32_t(config.IDRange.Max),
	}

	err := C.sss_idmap_add_domain(c.ctx, cDomainName, cDomainSID, &cRange)
	if err != C.IDMAP_SUCCESS {
		switch err {
		case C.IDMAP_SID_INVALID:
			return fmt.Errorf("%w: invalid domain SID %s", ErrInvalidSID, config.DomainSID)
		case C.IDMAP_COLLISION:
			return fmt.Errorf("%w: domain %s already exists or range conflicts", ErrInternal, config.DomainName)
		default:
			return fmt.Errorf("%w: failed to add domain %s (code: %d)", ErrInternal, config.DomainName, err)
		}
	}

	return nil
}

// Close frees the ID mapping context
func (c *IDMapContext) Close() error {
	if c.ctx != nil {
		err := C.sss_idmap_free(c.ctx)
		c.ctx = nil
		if err != C.IDMAP_SUCCESS {
			return fmt.Errorf("%w: failed to free idmap context (code: %d)", ErrInternal, err)
		}
	}
	return nil
}

// SIDToUnixID converts a Windows SID to a Unix UID or GID
// Returns the Unix ID and an error if the conversion fails
func (c *IDMapContext) SIDToUnixID(sid string) (uint32, error) {
	if c.ctx == nil {
		return 0, fmt.Errorf("%w: context is nil", ErrInternal)
	}

	cSID := C.CString(sid)
	defer C.free(unsafe.Pointer(cSID))

	var unixID C.uint32_t

	err := C.sss_idmap_sid_to_unix(c.ctx, cSID, &unixID)
	if err != C.IDMAP_SUCCESS {
		switch err {
		case C.IDMAP_SID_INVALID:
			return 0, fmt.Errorf("%w: %s", ErrInvalidSID, sid)
		case C.IDMAP_NO_DOMAIN:
			return 0, fmt.Errorf("%w: %s", ErrNotFound, sid)
		default:
			return 0, fmt.Errorf("%w: failed to convert SID %s (code: %d)", ErrInternal, sid, err)
		}
	}

	return uint32(unixID), nil
}

// SIDToUnixID is a convenience function that creates a context, performs the conversion, and cleans up
func SIDToUnixID(sid string) (uint32, error) {
	ctx, err := NewIDMapContext()
	if err != nil {
		return 0, err
	}
	defer ctx.Close()

	return ctx.SIDToUnixID(sid)
}
