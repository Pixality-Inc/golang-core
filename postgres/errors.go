package postgres

import (
	"database/sql"
	"errors"
)

var errNoRows = errors.New("no rows")

var ErrNoRows = errors.Join(sql.ErrNoRows, errNoRows)
