package mic

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/gordonklaus/portaudio"
)

type Stream struct {
	stream   *portaudio.Stream
	nSamples int
	buffer   []int32
}

func Open() (*Stream, error) {
	if err := portaudio.Initialize(); err != nil {
		return nil, err
	}
	in := make([]int32, 64)
	stream, err := portaudio.OpenDefaultStream(1, 0, 44100, len(in), in)
	if err != nil {
		return nil, err
	}

	s := &Stream{
		stream:   stream,
		nSamples: 0,
		buffer:   in,
	}

	return s, nil
}

func (s *Stream) writeHeader(f *Buffer) error {
	if _, err := f.WriteString("FORM"); err != nil {
		return err
	}
	if err := (binary.Write(f, binary.BigEndian, int32(0))); err != nil { // total bytes
		return err
	}
	if _, err := f.WriteString("AIFF"); err != nil {
		return err
	}

	// common chunk
	if _, err := f.WriteString("COMM"); err != nil {
		return err
	}
	if err := (binary.Write(f, binary.BigEndian, int32(18))); err != nil { //size
		return err
	}
	if err := binary.Write(f, binary.BigEndian, int16(1)); err != nil {
		return err
	}
	// Write number of samples (int32(0))
	if err := binary.Write(f, binary.BigEndian, int32(0)); err != nil {
		return err
	}
	// Write bits per sample (int16(32))
	if err := binary.Write(f, binary.BigEndian, int16(32)); err != nil {
		return err
	}
	// Write 80-bit sample rate (44100)
	_, err := f.Write([]byte{0x40, 0x0e, 0xac, 0x44, 0, 0, 0, 0, 0, 0})
	if err != nil {
		return err
	}

	if _, err := f.WriteString("SSND"); err != nil {
		return err
	}
	// Write size (int32(0))
	if err := binary.Write(f, binary.BigEndian, int32(0)); err != nil {
		return err
	}
	// Write offset (int32(0))
	if err := binary.Write(f, binary.BigEndian, int32(0)); err != nil {
		return err
	}
	// Write block (int32(0))
	if err := binary.Write(f, binary.BigEndian, int32(0)); err != nil {
		return err
	}
	return nil
}

func (s *Stream) Read(f *Buffer, done <-chan struct{}) error {
	if err := s.stream.Start(); err != nil {
		return err
	}
	if err := s.writeHeader(f); err != nil {
		return err
	}

	for {
		select {
		case <-done:
			if err := s.updateHeader(f); err != nil {
				return err
			}
			return s.stream.Stop()
		default:
			if err := s.stream.Read(); err != nil {
				return err
			}
			if err := binary.Write(f, binary.BigEndian, s.buffer); err != nil {
				return err
			}
			s.nSamples += len(s.buffer)
		}
	}
}

func (s *Stream) updateHeader(f *Buffer) error {
	nSamples := s.nSamples

	totalBytes := 4 + 8 + 18 + 8 + 8 + 4*nSamples
	if _, err := f.Seek(4, 0); err != nil {
		return err
	}
	if err := binary.Write(f, binary.BigEndian, int32(totalBytes)); err != nil {
		return err
	}
	if _, err := f.Seek(22, 0); err != nil {
		return err
	}
	if err := (binary.Write(f, binary.BigEndian, int32(nSamples))); err != nil {
		return err
	}
	if _, err := f.Seek(42, 0); err != nil {
		return err
	}
	if err := (binary.Write(f, binary.BigEndian, int32(4*nSamples+8))); err != nil {
		return err
	}
	return nil
}

func (s *Stream) Close() error {
	return s.stream.Close()
}

func Terminate() error {
	return portaudio.Terminate()
}

type Buffer struct {
	buf []byte
	pos int
}

func NewBuffer() *Buffer {
	return &Buffer{
		buf: make([]byte, 0, 1024),
	}
}

func (m *Buffer) Write(p []byte) (n int, err error) {
	minCap := m.pos + len(p)
	if minCap > cap(m.buf) {
		buf2 := make([]byte, len(m.buf), minCap+len(p))
		copy(buf2, m.buf)
		m.buf = buf2
	}
	if minCap > len(m.buf) {
		m.buf = m.buf[:minCap]
	}
	copy(m.buf[m.pos:], p)
	m.pos += len(p)
	return len(p), nil
}

func (m *Buffer) WriteString(s string) (n int, err error) {
	return m.Write([]byte(s))
}

func (m *Buffer) Seek(offset int64, whence int) (int64, error) {
	newPos, offs := 0, int(offset)
	switch whence {
	case io.SeekStart:
		newPos = offs
	case io.SeekCurrent:
		newPos = m.pos + offs
	case io.SeekEnd:
		newPos = len(m.buf) + offs
	}
	if newPos < 0 {
		return 0, errors.New("negative result pos")
	}
	m.pos = newPos
	return int64(newPos), nil
}

func (m *Buffer) Bytes() []byte {
	return m.buf
}