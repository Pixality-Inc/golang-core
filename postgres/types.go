//nolint:recvcheck
package postgres

import (
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/pixality-inc/golang-core/json"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/wkb"
	"github.com/paulmach/orb/encoding/wkt"
)

// Errors

var (
	ErrStringExpected      = errors.New("string expected")
	ErrParseUuid           = errors.New("parse uuid")
	errExpectedBytesToJSON = errors.New("expected []byte to unmarshal to json")
	errExpectedSlice       = errors.New("expected slice type to unmarshal")
	errInvalidString       = errors.New("value is not a valid string")
)

// Identified Model

type Identifier interface {
	String() string
}

type ModelWithId[T Identifier] interface {
	GetId() T
}

// Utils

func ScanTypedIdUuid[T Identifier](
	value any,
	constructor func(value uuid.UUID) T,
	out **T,
) error {
	val, ok := value.(string)
	if !ok {
		return fmt.Errorf("%w: got %T", ErrStringExpected, val)
	}

	uuidValue, err := uuid.Parse(val)
	if err != nil {
		return fmt.Errorf("%w: %s: %w", ErrParseUuid, val, err)
	}

	**out = constructor(uuidValue)

	return nil
}

func ScanTypedIdStr[T Identifier](
	value any,
	constructor func(value string) T,
	out **T,
) error {
	val, ok := value.(string)
	if !ok {
		return fmt.Errorf("%w: got %T", ErrStringExpected, val)
	}

	**out = constructor(val)

	return nil
}

func RenderSqlDriverValue[T Identifier](value T) (driver.Value, error) {
	return value.String(), nil
}

// Sql Null

func NewSqlNull[T any](value *T) sql.Null[T] {
	if value == nil {
		return sql.Null[T]{
			Valid: false,
		}
	} else {
		return sql.Null[T]{
			V:     *value,
			Valid: true,
		}
	}
}

func NewSqlNullNull[T any]() sql.Null[T] {
	return NewSqlNull[T](nil)
}

// Json

type Json[T any] struct {
	Data T
}

func NewJson[T any](data T) Json[T] {
	return Json[T]{
		Data: data,
	}
}

func NewJsonRef[T any](data *T) *Json[T] {
	if data == nil {
		return nil
	}

	j := NewJson[T](*data)

	return &j
}

func (j *Json[T]) Scan(value any) error {
	val, ok := value.([]byte)

	if !ok {
		return fmt.Errorf("%w: got %T", errExpectedBytesToJSON, val)
	}

	if err := json.Unmarshal(val, &j.Data); err != nil {
		return err
	}

	return nil
}

func (j Json[T]) Value() (driver.Value, error) {
	buf, err := json.Marshal(j.Data)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// Slice

type Slice[T any] []T

func NewSlice[T any](data []T) Slice[T] {
	return Slice[T](data)
}

func NewSliceRef[T any](data *[]T) *Slice[T] {
	if data == nil {
		return nil
	}

	s := NewSlice[T](*data)

	return &s
}

func (s *Slice[T]) Scan(value any) error {
	val, ok := value.([]T)

	if !ok {
		return fmt.Errorf("%w: expected slice of %T, got %T", errExpectedSlice, s, val)
	}

	*s = val

	return nil
}

func (s Slice[T]) Value() (driver.Value, error) {
	return []T(s), nil
}

// Json Object

type JsonObject = json.Object

// Geometry

type Geometry struct {
	Geometry orb.Geometry
}

func (g *Geometry) Scan(value any) error {
	var valueStr string

	if val, ok := value.(string); !ok {
		return fmt.Errorf("%w: '%v'", errInvalidString, value)
	} else {
		valueStr = val
	}

	bytes, err := hex.DecodeString(valueStr)
	if err != nil {
		return err
	}

	geometry, err := wkb.Unmarshal(bytes)
	if err != nil {
		return err
	}

	g.Geometry = geometry

	return nil
}

func (g Geometry) Value() (driver.Value, error) {
	return string(wkt.Marshal(g.Geometry)), nil
}
