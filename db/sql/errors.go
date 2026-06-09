package sqldb

import "errors"

// Returned by New until a real database/sql implementation is wired in.
var ErrNotImplemented = errors.New("not yet implemented")
