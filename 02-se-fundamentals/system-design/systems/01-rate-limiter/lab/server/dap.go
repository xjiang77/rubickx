package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

type dapClient struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
	seq    int
	events []map[string]any
}

func newDAPClient(conn net.Conn) *dapClient {
	return &dapClient{conn: conn, reader: bufio.NewReader(conn), writer: bufio.NewWriter(conn)}
}

func (c *dapClient) close() error {
	return c.conn.Close()
}

func (c *dapClient) request(ctx context.Context, command string, arguments any) (map[string]any, error) {
	c.seq++
	sequence := c.seq
	message := map[string]any{"seq": sequence, "type": "request", "command": command}
	if arguments != nil {
		message["arguments"] = arguments
	}
	if err := c.write(ctx, message); err != nil {
		return nil, err
	}
	for {
		incoming, err := c.read(ctx)
		if err != nil {
			return nil, err
		}
		switch stringValue(incoming["type"]) {
		case "event":
			c.events = append(c.events, incoming)
		case "response":
			if intValue(incoming["request_seq"]) != sequence {
				continue
			}
			if success, ok := incoming["success"].(bool); ok && !success {
				return nil, fmt.Errorf("DAP %s failed: %s", command, stringValue(incoming["message"]))
			}
			body, _ := incoming["body"].(map[string]any)
			return body, nil
		}
	}
}

func (c *dapClient) waitEvent(ctx context.Context, name string) (map[string]any, error) {
	if event, ok := c.popEvent(name); ok {
		return event, nil
	}
	for {
		incoming, err := c.read(ctx)
		if err != nil {
			return nil, err
		}
		if stringValue(incoming["type"]) == "event" {
			if stringValue(incoming["event"]) == name {
				return incoming, nil
			}
			c.events = append(c.events, incoming)
		}
	}
}

func (c *dapClient) popEvent(name string) (map[string]any, bool) {
	for index, event := range c.events {
		if stringValue(event["event"]) == name {
			c.events = append(c.events[:index], c.events[index+1:]...)
			return event, true
		}
	}
	return nil, false
}

func (c *dapClient) write(ctx context.Context, message map[string]any) error {
	if deadline, ok := ctx.Deadline(); ok {
		_ = c.conn.SetWriteDeadline(deadline)
	}
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.writer, "Content-Length: %d\r\n\r\n", len(payload)); err != nil {
		return err
	}
	if _, err := c.writer.Write(payload); err != nil {
		return err
	}
	return c.writer.Flush()
}

func (c *dapClient) read(ctx context.Context) (map[string]any, error) {
	if deadline, ok := ctx.Deadline(); ok {
		_ = c.conn.SetReadDeadline(deadline)
	}
	contentLength := -1
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		if line == "" {
			break
		}
		name, value, found := strings.Cut(line, ":")
		if found && strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			contentLength, err = strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return nil, err
			}
		}
	}
	if contentLength < 0 || contentLength > 8<<20 {
		return nil, fmt.Errorf("invalid DAP Content-Length %d", contentLength)
	}
	payload := make([]byte, contentLength)
	if _, err := io.ReadFull(c.reader, payload); err != nil {
		return nil, err
	}
	var message map[string]any
	if err := json.Unmarshal(payload, &message); err != nil {
		return nil, err
	}
	return message, nil
}

func stringValue(value any) string {
	stringValue, _ := value.(string)
	return stringValue
}

func intValue(value any) int {
	switch number := value.(type) {
	case float64:
		return int(number)
	case int:
		return number
	case json.Number:
		parsed, _ := number.Int64()
		return int(parsed)
	default:
		return 0
	}
}
