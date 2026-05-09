package iterator

import (
	"errors"
	"io"
)

type ReaderIterator struct {
	Iterator[byte]

	buffer    byte
	reader    io.Reader
	hasBuffer bool
	done      bool
	err       error
}

func NewReaderIterator(reader io.Reader) Iterator[byte] {
	return &ReaderIterator{
		reader: reader,
	}
}

func (i *ReaderIterator) HasNext() bool {
	if i.hasBuffer {
		return true
	}

	if i.done {
		return false
	}

	return i.readNext()
}

func (i *ReaderIterator) Next() byte {
	if !i.HasNext() {
		return 0
	}

	i.hasBuffer = false

	return i.buffer
}

func (i *ReaderIterator) Err() error {
	return i.err
}

func (i *ReaderIterator) readNext() bool {
	var buffer [1]byte

	_, err := io.ReadFull(i.reader, buffer[:])
	if err == nil {
		i.buffer = buffer[0]
		i.hasBuffer = true

		return true
	}

	i.done = true

	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return false
	}

	i.err = err

	return false
}
