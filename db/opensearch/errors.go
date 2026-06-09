package opensearch

import "errors"

// Returned by New until a real opensearch-go implementation is wired in.
var ErrNotImplemented = errors.New("not yet implemented")
