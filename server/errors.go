package server

import (
	"github.com/dropbox/godropbox/errors"
)

type ServerError struct {
	errors.DropboxError
}
