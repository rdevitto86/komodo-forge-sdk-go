package sqldb

import "errors"

// ErrNotImplemented is returned by New until a real database/sql implementation is wired in.
var ErrNotImplemented = errors.New("not yet implemented")
