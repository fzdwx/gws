package main

import (
	"fmt"
	"github.com/lxzan/gws"
	"github.com/lxzan/gws/internal"
	"net/http"
)

func main() {
	var upgrader = gws.NewUpgrader(new(WebSocket), &gws.ServerOption{
		CompressEnabled:     true,
		CheckUtf8Enabled:    true,
		ReadMaxPayloadSize:  32 * 1024 * 1024,
		WriteMaxPayloadSize: 32 * 1024 * 1024,
		ReadAsyncEnabled:    true,
		ReadBufferSize:      8 * 1024,
		WriteBufferSize:     8 * 1024,
	})

	http.HandleFunc("/connect", func(writer http.ResponseWriter, request *http.Request) {
		socket, err := upgrader.Accept(writer, request)
		if err != nil {
			return
		}
		socket.Listen()
	})

	_ = http.ListenAndServe(":3000", nil)
}

type WebSocket struct{}

func (c *WebSocket) OnClose(socket *gws.Conn, code uint16, reason []byte) {
	fmt.Printf("onclose: code=%d, payload=%s\n", code, string(reason))
}

func (c *WebSocket) OnError(socket *gws.Conn, err error) {
	fmt.Printf("onerror: err=%s\n", err.Error())
}

func (c *WebSocket) OnOpen(socket *gws.Conn) {
	println("connected")
}

func (c *WebSocket) OnPing(socket *gws.Conn, payload []byte) {
	fmt.Printf("onping: payload=%s\n", string(payload))
	socket.WritePong(payload)
}

func (c *WebSocket) OnPong(socket *gws.Conn, payload []byte) {}

func (c *WebSocket) OnMessage(socket *gws.Conn, message *gws.Message) {
	defer message.Close()
	if internal.AlphabetNumeric.Uint32()&1 == 1 {
		socket.WriteAsync(message.Opcode, cloneBytes(message.Data.Bytes()))
	} else {
		socket.WriteMessage(message.Opcode, message.Data.Bytes())
	}
}

func cloneBytes(b []byte) []byte {
	var d = make([]byte, len(b))
	copy(d, b)
	return d
}
