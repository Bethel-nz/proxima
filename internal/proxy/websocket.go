package proxy

import (
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
    writeWait      = 10 * time.Second    // Time allowed to write a message
    pongWait       = 60 * time.Second    // Time allowed to read the next pong message
    pingPeriod     = (pongWait * 9) / 10 // Send pings to peer with this period
    maxMessageSize = 512 * 1024          // Maximum message size allowed
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        return true // Allow all origins for now
    },
}

func (p *Proxy) handleWebSocket(w http.ResponseWriter, r *http.Request, proxyReq *http.Request, logger *zap.Logger) {
    // Create target URL
    targetURL := url.URL{
        Scheme: "wss",
        Host:   p.target.Host,
        Path:   r.URL.Path,
    }
    if r.URL.RawQuery != "" {
        targetURL.RawQuery = r.URL.RawQuery
    }

    logger.Info("initiating websocket connection",
        zap.String("target_url", targetURL.String()),
    )

    // Connect to upstream
    upstreamConn, resp, err := websocket.DefaultDialer.Dial(targetURL.String(), proxyReq.Header)
    if err != nil {
        statusCode := http.StatusBadGateway
        if resp != nil {
            statusCode = resp.StatusCode
        }
        logger.Error("failed to connect to upstream websocket",
            zap.Error(err),
            zap.Int("status_code", statusCode),
        )
        http.Error(w, "WebSocket connection failed", statusCode)
        return
    }
    defer upstreamConn.Close()

    // Upgrade client connection
    clientConn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        logger.Error("failed to upgrade client connection", zap.Error(err))
        return
    }
    defer clientConn.Close()

    // Setup connection parameters
    upstreamConn.SetReadLimit(maxMessageSize)
    clientConn.SetReadLimit(maxMessageSize)

    // Create wait group for goroutines
    var wg sync.WaitGroup
    wg.Add(2)

    // Create channels for error handling
    errorChan := make(chan error, 2)

    // Forward messages from client to upstream
    go func() {
        defer wg.Done()
        err := p.pumpMessages(clientConn, upstreamConn, "client→upstream", logger)
        if err != nil {
            errorChan <- err
        }
    }()

    // Forward messages from upstream to client
    go func() {
        defer wg.Done()
        err := p.pumpMessages(upstreamConn, clientConn, "upstream→client", logger)
        if err != nil {
            errorChan <- err
        }
    }()

    // Wait for completion or error
    go func() {
        wg.Wait()
        close(errorChan)
    }()

    // Handle errors
    for err := range errorChan {
        if err != nil {
            logger.Error("websocket error",
                zap.Error(err),
            )
        }
    }

    logger.Info("websocket connection closed")
}

func (p *Proxy) pumpMessages(src, dst *websocket.Conn, direction string, logger *zap.Logger) error {
    for {
        messageType, message, err := src.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                logger.Error("unexpected websocket close",
                    zap.String("direction", direction),
                    zap.Error(err),
                )
            }
            return err
        }

        logger.Debug("websocket message",
            zap.String("direction", direction),
            zap.Int("message_type", messageType),
            zap.Int("message_size", len(message)),
        )

        err = dst.WriteMessage(messageType, message)
        if err != nil {
            logger.Error("failed to write websocket message",
                zap.String("direction", direction),
                zap.Error(err),
            )
            return err
        }
    }
}
