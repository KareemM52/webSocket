package main

import (
    "fmt"
    "net"
    "net/http"
    "time"
)

// WebSocket connection structure
type WebSocketConnection struct {
    conn net.Conn
}

func main() {
    // Create a TCP listener
    listener, err := net.Listen("tcp", "localhost:8080")
    if err != nil {
        fmt.Println("Error creating listener:", err)
        return
    }
    defer listener.Close()
    fmt.Println("WebSocket server is listening on :8080")

    for {
        // Accept incoming TCP connections
        conn, err := listener.Accept()
        if err != nil {
            fmt.Println("Error accepting connection:", err)
            continue
        }

        // Handle the WebSocket connection
        go handleWebSocket(conn)
    }
}

func handleWebSocket(conn net.Conn) {
    defer conn.Close()

    // Upgrade the TCP connection to a WebSocket connection
    wsConn := WebSocketConnection{conn}
    request, err := http.ReadRequest(bufio.NewReader(conn))
    if err != nil {
        fmt.Println("Error reading WebSocket request:", err)
        return
    }

    // Accept the WebSocket request
    response := http.Response{
        Status:     "101 Switching Protocols",
        StatusCode: 101,
        Proto:      request.Proto,
        ProtoMajor: request.ProtoMajor,
        ProtoMinor: request.ProtoMinor,
        Header:     http.Header{},
    }
    response.Header.Set("Upgrade", "websocket")
    response.Header.Set("Connection", "Upgrade")
    response.Header.Set("Sec-WebSocket-Accept", calculateWebSocketAccept(request))

    // Send the response to complete the WebSocket handshake
    response.Write(conn)

    // Start handling WebSocket messages
    for {
        // Read the WebSocket frame
        _, payload, err := wsConn.readWebSocketFrame()
        if err != nil {
            fmt.Println("Error reading WebSocket frame:", err)
            return
        }

        // Handle the WebSocket message (in this example, we just print it)
        fmt.Println("Received WebSocket message:", string(payload))
    }
}

// Helper function to read a WebSocket frame
func (conn WebSocketConnection) readWebSocketFrame() (byte, []byte, error) {
    // Read the frame header
    header := make([]byte, 2)
    _, err := conn.conn.Read(header)
    if err != nil {
        return 0, nil, err
    }

    fin := (header[0] & 0x80) == 0x80
    opcode := header[0] & 0x0F
    payloadLength := int(header[1] & 0x7F)

    // Read the extended payload length if necessary
    if payloadLength == 126 {
        extendedLength := make([]byte, 2)
        _, err := conn.conn.Read(extendedLength)
        if err != nil {
            return 0, nil, err
        }
        payloadLength = int(extendedLength[0])<<8 | int(extendedLength[1])
    } else if payloadLength == 127 {
        extendedLength := make([]byte, 8)
        _, err := conn.conn.Read(extendedLength)
        if err != nil {
            return 0, nil, err
        }
        payloadLength = int(extendedLength[0])<<56 |
            int(extendedLength[1])<<48 |
            int(extendedLength[2])<<40 |
            int(extendedLength[3])<<32 |
            int(extendedLength[4])<<24 |
            int(extendedLength[5])<<16 |
            int(extendedLength[6])<<8 |
            int(extendedLength[7])
    }

    // Read the mask if present
    mask := make([]byte, 4)
    _, err = conn.conn.Read(mask)
    if err != nil {
        return 0, nil, err
    }

    // Read the payload
    payload := make([]byte, payloadLength)
    _, err = conn.conn.Read(payload)
    if err != nil {
        return 0, nil, err
    }

    // Unmask the payload
    for i := 0; i < payloadLength; i++ {
        payload[i] ^= mask[i%4]
    }

    return opcode, payload, nil
}

// Helper function to calculate the WebSocket Sec-WebSocket-Accept header
func calculateWebSocketAccept(request *http.Request) string {
    key := request.Header.Get("Sec-WebSocket-Key")
    const websocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
    return base64.StdEncoding.EncodeToString(sha1.Sum([]byte(key + websocketGUID))[:])
}
