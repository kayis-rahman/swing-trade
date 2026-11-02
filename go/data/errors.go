package data

import "errors"

// ErrOptimisticLock signals an optimistic-lock conflict.
var ErrOptimisticLock = errors.New("optimistic lock conflict")
