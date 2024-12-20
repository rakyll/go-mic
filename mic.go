package mic

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/gordonklaus/portaudio"
)

type Stream struct {
	stream        *portaudio.Stream
	totalSamples  int
	buffer        []int32
	encodedBuffer *Buffer
	stopCh        chan struct{}
	err           error
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
		stream:        stream,
		totalSamples:  0,
		buffer:        in,
		encodedBuffer: NewBuffer(),
		stopCh:        make(chan struct{}),
	}

	return s, nil
}

func writeBigEndian(w io.Writer, data ...any) error {
	for _, d := range data {
		if err := binary.Write(w, binary.BigEndian, d); err != nil {
			return err
		}
	}
	return nil
}

func (s *Stream) writeHeader(w *Buffer) error {
	if _, err := w.WriteString("FORM"); err != nil {
		return err
	}
	if err := writeBigEndian(w, int32(0)); err != nil { // total bytes
		return err
	}
	if _, err := w.WriteString("AIFF"); err != nil {
		return err
	}

	// common chunk
	if _, err := w.WriteString("COMM"); err != nil {
		return err
	}
	if err := writeBigEndian(w,
		int32(18), // size
		int16(1),  // channels
		int32(0),  // number of samples,
		int16(32), // bits per sample
	); err != nil {
		return err
	}
	// Write 80-bit sample rate (44100)
	_, err := w.Write([]byte{0x40, 0x0e, 0xac, 0x44, 0, 0, 0, 0, 0, 0})
	if err != nil {
		return err
	}

	if _, err := w.WriteString("SSND"); err != nil {
		return err
	}
	return writeBigEndian(w,
		int32(0),
		int32(0), // offset
		int32(0), // block
	)
}

func (s *Stream) Start() error {
	if err := s.writeHeader(s.encodedBuffer); err != nil {
		return err
	}
	if err := s.stream.Start(); err != nil {
		return err
	}
	go func() {
		for {
			select {
			case <-s.stopCh:
				if err := s.stream.Stop(); err != nil {
					s.err = err
					return
				}
				if err := s.updateHeader(s.encodedBuffer); err != nil {
					s.err = err
					return
				}
			default:
				if err := s.stream.Read(); err != nil {
					s.err = err
					return
				}
				if err := binary.Write(s.encodedBuffer, binary.BigEndian, s.buffer); err != nil {
					s.err = err
					return
				}
				s.totalSamples += len(s.buffer)
			}
		}
	}()
	return nil
}

func (s *Stream) stop() error {
	close(s.stopCh)
	return s.err
}

func (s *Stream) updateHeader(w *Buffer) error {
	totalSamples := s.totalSamples

	totalBytes := 4 + 8 + 18 + 8 + 8 + 4*totalSamples
	if _, err := w.Seek(4, 0); err != nil {
		return err
	}
	if err := writeBigEndian(w, int32(totalBytes)); err != nil {
		return err
	}
	if _, err := w.Seek(22, 0); err != nil {
		return err
	}
	if err := writeBigEndian(w, int32(totalSamples)); err != nil {
		return err
	}
	if _, err := w.Seek(42, 0); err != nil {
		return err
	}
	if err := writeBigEndian(w, int32(4*totalSamples+8)); err != nil {
		return err
	}
	return nil
}

func (s *Stream) Close() error {
	return s.stream.Close()
}

func (s *Stream) EncodedBytes() []byte {
	return s.encodedBuffer.Bytes()
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
