package gws

import (
	"compress/flate"
	"encoding/binary"
	"github.com/lxzan/gws/internal"
	"io"
	"math"
	"sync"
)

func newCompressor() *compressor {
	fw, _ := flate.NewWriter(nil, flate.BestSpeed)
	return &compressor{
		mu:          sync.Mutex{},
		writeBuffer: internal.NewBuffer(nil),
		fw:          fw,
	}
}

// 压缩器
type compressor struct {
	mu          sync.Mutex
	writeBuffer *internal.Buffer
	fw          *flate.Writer
}

// 压缩并构建WriteFrame
func (c *compressor) Compress(content []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.writeBuffer.Cap() > internal.Lv3 {
		c.writeBuffer = internal.NewBuffer(nil)
	}

	c.writeBuffer.Reset()
	c.fw.Reset(c.writeBuffer)
	if err := writeN(c.fw, content, len(content)); err != nil {
		return nil, err
	}
	if err := c.fw.Flush(); err != nil {
		return nil, err
	}

	contentLength := c.writeBuffer.Len()
	compressedContent := c.writeBuffer.Bytes()
	if n := c.writeBuffer.Len(); n >= 4 {
		if tail := compressedContent[n-4:]; binary.BigEndian.Uint32(tail) == math.MaxUint16 {
			contentLength -= 4
			compressedContent = compressedContent[:n-4]
		}
	}

	return compressedContent, nil
}

func newDecompressor() *decompressor {
	return &decompressor{
		fr: flate.NewReader(nil),
	}
}

type decompressor struct {
	fr io.ReadCloser
}

// 解压
func (c *decompressor) Decompress(msg *Message) error {
	msg.cbuf = msg.dbuf
	_, _ = msg.dbuf.Write(internal.FlateTail)
	resetter := c.fr.(flate.Resetter)
	if err := resetter.Reset(msg.dbuf, nil); err != nil {
		return err
	}

	msg.dbuf = _pool.Get(msg.dbuf.Len())
	_, err := io.Copy(msg.dbuf, c.fr)
	return err
}
