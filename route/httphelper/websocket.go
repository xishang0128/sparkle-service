package httphelper

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

const (
	webSocketGUID       = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	webSocketMaxPayload = 1 << 20

	WebSocketOpText  = 0x1
	WebSocketOpClose = 0x8
	WebSocketOpPing  = 0x9
	WebSocketOpPong  = 0xA
)

func AcceptWebSocket(w http.ResponseWriter, r *http.Request) (net.Conn, *bufio.ReadWriter, error) {
	if r.Method != http.MethodGet {
		return nil, nil, BadRequest("websocket 仅支持 GET")
	}
	if !headerContainsToken(r.Header, "Connection", "upgrade") ||
		!strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return nil, nil, BadRequest("缺少 websocket upgrade 请求头")
	}
	if r.Header.Get("Sec-WebSocket-Version") != "13" {
		return nil, nil, BadRequest("不支持的 websocket 版本")
	}

	key := strings.TrimSpace(r.Header.Get("Sec-WebSocket-Key"))
	if key == "" {
		return nil, nil, BadRequest("缺少 Sec-WebSocket-Key")
	}
	if decoded, err := base64.StdEncoding.DecodeString(key); err != nil || len(decoded) != 16 {
		return nil, nil, BadRequest("无效的 Sec-WebSocket-Key")
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("当前连接不支持 websocket")
	}

	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return nil, nil, err
	}

	accept := webSocketAccept(key)
	if _, err := rw.WriteString(
		"HTTP/1.1 101 Switching Protocols\r\n" +
			"Upgrade: websocket\r\n" +
			"Connection: Upgrade\r\n" +
			"Sec-WebSocket-Accept: " + accept + "\r\n\r\n",
	); err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	if err := rw.Flush(); err != nil {
		_ = conn.Close()
		return nil, nil, err
	}

	return conn, rw, nil
}

func webSocketAccept(key string) string {
	sum := sha1.Sum([]byte(key + webSocketGUID))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func headerContainsToken(header http.Header, name string, token string) bool {
	for _, value := range header.Values(name) {
		for part := range strings.SplitSeq(value, ",") {
			if strings.EqualFold(strings.TrimSpace(part), token) {
				return true
			}
		}
	}
	return false
}

func WriteWebSocketFrame(w io.Writer, opcode byte, payload []byte) error {
	header := []byte{0x80 | opcode}
	payloadLen := len(payload)
	switch {
	case payloadLen <= 125:
		header = append(header, byte(payloadLen))
	case payloadLen <= 0xffff:
		header = append(header, 126, byte(payloadLen>>8), byte(payloadLen))
	default:
		header = append(header, 127)
		var lenBuf [8]byte
		binary.BigEndian.PutUint64(lenBuf[:], uint64(payloadLen))
		header = append(header, lenBuf[:]...)
	}

	if _, err := w.Write(header); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

func ReadWebSocketFrame(r io.Reader) (byte, []byte, error) {
	var header [2]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return 0, nil, err
	}

	opcode := header[0] & 0x0f
	if header[0]&0x80 == 0 {
		return 0, nil, fmt.Errorf("websocket 不支持分片帧")
	}
	masked := header[1]&0x80 != 0
	if !masked {
		return 0, nil, fmt.Errorf("websocket 客户端帧必须 mask")
	}
	payloadLen := uint64(header[1] & 0x7f)

	switch payloadLen {
	case 126:
		var buf [2]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return 0, nil, err
		}
		payloadLen = uint64(binary.BigEndian.Uint16(buf[:]))
	case 127:
		var buf [8]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return 0, nil, err
		}
		payloadLen = binary.BigEndian.Uint64(buf[:])
	}

	if payloadLen > webSocketMaxPayload {
		return 0, nil, fmt.Errorf("websocket payload 过大")
	}

	var maskKey [4]byte
	if masked {
		if _, err := io.ReadFull(r, maskKey[:]); err != nil {
			return 0, nil, err
		}
	}

	payload := make([]byte, payloadLen)
	if _, err := io.ReadFull(r, payload); err != nil {
		return 0, nil, err
	}
	if masked {
		for i := range payload {
			payload[i] ^= maskKey[i%4]
		}
	}

	return opcode, payload, nil
}
