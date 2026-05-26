package connection

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"spored/internal/message"
	"strings"
)

type Node interface {
	Id() string
	Route(msg message.Message) error
	Disconnected()
	WitnessIncoming(msg string, node string, index int64)
	WitnessOutgoing(msg string, node string, index int64)
	WitnessSpore(msg string)
	WitnessNode(msg string)
}

type Connection struct {
	node Node

	conn net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
	j int64
	pid int
}

func (c *Connection) Open(conn net.Conn) error {
	c.conn = conn
	c.reader = bufio.NewReader(conn)
	c.writer = bufio.NewWriter(conn)
	c.j = 0
	c.pid = peerPID(conn)
	return nil
}

func (c *Connection) Run(node Node) {
	slog.Info("Connecting", "node", node.Id())
	c.node = node

	var index int64 = 0
	for {
		raw, err := c.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF || errors.Is(err, net.ErrClosed) {
				slog.Info("Disconnecting", "node", node.Id())
				break
			} else {
				slog.Warn("Unable to read message", "node", node.Id(), "error", err)
				continue
			}
		}
		raw = strings.TrimSpace(raw)
		i := index
		index++
		slog.Info("Receiving message from node", "i", i, "node", node.Id())
		slog.Debug("Message", "i", i, "raw", raw)

		// Intercept node-emitted witness messages before routing.
		// A node sends: witness <body>
		// The hub appends cast= and spore_time= then dispatches to witnesses.
		if strings.HasPrefix(raw, "witness ") {
			node.WitnessNode(strings.TrimPrefix(raw, "witness "))
			continue
		}

		node.WitnessIncoming(raw, node.Id(), i)

		msg, err := message.Parse(raw, node.Id())
		if err != nil {
			slog.Warn("Unable to parse message", "i", i, "node", node.Id(), "error", err)
			if handle := message.ExtractHandle(raw); handle != "" {
				// Build the error string without capture= so NodeError.Parse
				// will inject it correctly (with the right node ID).
				errStr := fmt.Sprintf(`~%s:SPORE.unknown error code=MessageMalformed what="%s"`, handle, err.Error())
				// Route the error through the node so the router can remove
				// replies[handle] if one was registered for this transaction.
				// If routing fails (no pending reply, e.g. the malformed message
				// was the originating cast itself), fall back to writing directly
				// on this connection so the sender still gets the error.
				errMsg, parseErr := message.Parse(errStr, "SPORE.hub")
				if parseErr == nil {
					if routeErr := node.Route(errMsg); routeErr != nil {
						_ = c.SendRaw(errStr + ` capture=SPORE.hub`)
					}
				} else {
					_ = c.SendRaw(errStr + ` capture=SPORE.hub`)
				}
			} else {
				node.WitnessSpore(message.SporeEvent("warn", "Message parse failed", fmt.Sprintf(`error="%s"`, err.Error())))
			}
			continue
		}
		if msg == nil {
			slog.Warn("Parse returned nil without error", "i", i, "node", node.Id(), "raw", raw)
			node.WitnessSpore(message.SporeEvent("warn", "Parse returned nil without error"))
			continue
		}
		msg.SetMessageId(i)

		err = node.Route(msg)
		if err != nil {
			slog.Warn("Unable to route message", "i", i, "node", node.Id(), "error", err)
			node.WitnessSpore(message.SporeEvent("warn", "Routing error", fmt.Sprintf(`error="%s"`, err.Error())))
			continue
		}

		slog.Info("Message received", "i", i)
	}
	node.Disconnected()
}

func (c *Connection) Close() {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

func (c *Connection) Send(msg message.Message) error {
	slog.Info("Sending message to node", "i", msg.MessageId(), "node", msg.Destination())
	slog.Debug("Message", "i", msg.MessageId(), "raw", msg.ToString())
	if c.node != nil {
		c.node.WitnessOutgoing(msg.ToString(), msg.Destination(), msg.MessageId())
	}
	return c.SendRaw(msg.ToString())
}

// used to send JSONs
// used to send witness messages
func (c *Connection) SendRaw(s string) error {
	if c.conn == nil {
		return errors.New("node not connected")	
	}
	_, err := c.conn.Write([]byte(s + "\n"))
	if err != nil {
		slog.Warn("Unable to send message", "raw", s)
		return err
	}
	return nil
}

func (c *Connection) IsConnected() bool {
	return c.conn != nil
}

func (c *Connection) GetPID() int {
	return c.pid
}
