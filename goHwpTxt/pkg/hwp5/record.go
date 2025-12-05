package hwp5

import (
	"encoding/binary"
	"io"
)

type Record struct {
	TagID   uint32
	Level   uint32
	Size    uint32
	Payload []byte
}

func ReadRecord(r io.Reader) (*Record, error) {
	var header uint32
	err := binary.Read(r, binary.LittleEndian, &header)
	if err != nil {
		return nil, err
	}

	tagID := header & 0x3ff
	level := (header >> 10) & 0x3ff
	size := (header >> 20) & 0xfff

	if size == 0xfff {
		var realSize uint32
		err := binary.Read(r, binary.LittleEndian, &realSize)
		if err != nil {
			return nil, err
		}
		size = realSize
	}

	payload := make([]byte, size)
	_, err = io.ReadFull(r, payload)
	if err != nil {
		return nil, err
	}

	return &Record{
		TagID:   tagID,
		Level:   level,
		Size:    size,
		Payload: payload,
	}, nil
}
