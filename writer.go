package gws

import (
	"bytes"
	"errors"
	"github.com/lxzan/gws/internal"
	"sync/atomic"
)

// WriteClose proactively close the connection
// code: https://developer.mozilla.org/zh-CN/docs/Web/API/CloseEvent#status_codes
// 通过emitError发送关闭帧, 将连接状态置为关闭, 用于服务端主动断开连接
// 没有特殊原因的话, 建议code=0, reason=nil
func (c *Conn) WriteClose(code uint16, reason []byte) {
	var err = internal.NewError(internal.StatusCode(code), internal.GwsError(""))
	if len(reason) > 0 {
		err.Err = errors.New(string(reason))
	}
	c.emitError(err)
}

// WritePing write ping frame
func (c *Conn) WritePing(payload []byte) error {
	return c.WriteMessage(OpcodePing, payload)
}

// WritePong write pong frame
func (c *Conn) WritePong(payload []byte) error {
	return c.WriteMessage(OpcodePong, payload)
}

// WriteString write text frame
func (c *Conn) WriteString(s string) error {
	return c.WriteMessage(OpcodeText, internal.StringToBytes(s))
}

// WriteMessage writes message
// 发送消息
func (c *Conn) WriteMessage(opcode Opcode, payload []byte) error {
	if atomic.LoadUint32(&c.closed) == 1 {
		return internal.ErrConnClosed
	}
	err := c.doWrite(opcode, payload)
	c.emitError(err)
	return err
}

// 关闭状态置为1后还能写, 以便发送关闭帧
func (c *Conn) doWrite(opcode Opcode, payload []byte) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()

	var useCompress = c.compressEnabled && opcode.IsDataFrame() && len(payload) >= c.config.CompressThreshold
	if useCompress {
		compressedContent, err := c.compressor.Compress(bytes.NewBuffer(payload))
		if err != nil {
			return internal.NewError(internal.CloseInternalServerErr, err)
		}
		payload = compressedContent.Bytes()
	}
	if len(payload) > c.config.WriteMaxPayloadSize {
		return internal.CloseMessageTooLarge
	}

	var header = frameHeader{}
	var n = len(payload)
	var headerLength = header.GenerateServerHeader(true, useCompress, opcode, n)
	if err := internal.WriteN(c.wbuf, header[:headerLength], headerLength); err != nil {
		return err
	}
	if err := internal.WriteN(c.wbuf, payload, n); err != nil {
		return err
	}
	return c.wbuf.Flush()
}

// WriteAsync 异步非阻塞地写入消息
// Write messages asynchronously and non-blockingly
func (c *Conn) WriteAsync(opcode Opcode, payload []byte) error {
	if atomic.LoadUint32(&c.closed) == 1 {
		return internal.ErrConnClosed
	}
	return c.writeQueue.Push(func() { c.emitError(c.doWrite(opcode, payload)) })
}
