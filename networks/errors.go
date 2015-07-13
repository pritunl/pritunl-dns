package networks

import (
	"github.com/dropbox/godropbox/errors"
)

type SystemError struct {
	errors.DropboxError
}
