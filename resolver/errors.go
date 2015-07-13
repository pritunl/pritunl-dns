package resolver

import (
	"github.com/dropbox/godropbox/errors"
)

type NotFoundError struct {
	errors.DropboxError
}

type ResolveError struct {
	errors.DropboxError
}
