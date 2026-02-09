package gateway

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

// Message header is currently uint64 zero value.
const wsMsgHeader = uint64(0)

// decodeWSMessage decodes a websocket message as bytes into a proto.Message.
func decodeWSMessage(bytes []byte, msg proto.Message) error {
	// Message header is optional until the end of grace period that ends Feb 1, 2023.
	// Check if the header is present.
	if len(bytes) > 0 && bytes[0] == 0 {
		// New message format. The Protobuf message is preceded by a zero byte header.
		// Decode the header.
		header, n := binary.Uvarint(bytes)
		if header != wsMsgHeader {
			return errors.New("unexpected non-zero header")
		}
		// Skip the header. It really is just a single zero byte for now.
		bytes = bytes[n:]
	}
	// If no header was present (the "if" check above), then this is the old
	// message format. No header is present.

	// Decode WebSocket message as a Protobuf message.
	err := proto.Unmarshal(bytes, msg)
	if err != nil {
		return err
	}
	return nil
}

// encodeWSMessage encodes a proto.Message into bytes suitable for websocket transmission.
func encodeWSMessage(msg proto.Message) ([]byte, error) {
	// Encode the message to protobuf
	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal proto: %w", err)
	}

	// Prepend the header (single zero byte)
	header := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(header, wsMsgHeader)

	result := make([]byte, n+len(data))
	copy(result, header[:n])
	copy(result[n:], data)

	return result, nil
}

func writeWSMessage(con *websocket.Conn, msg []byte) error {
	writer, err := con.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return fmt.Errorf("next writer: %w", err)
	}

	// Write the encoded data.
	_, err = writer.Write(msg)
	if err != nil {
		return errors.Join(fmt.Errorf("write data: %w", err), writer.Close())
	}

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("close writer: %w", err)
	}
	return nil
}

type agentID struct {
	raw    []byte
	parsed string
}

// newAgentID parses the raw byte representation of the agent ID according to the provided format.
func newAgentID(raw []byte) (agentID, error) {
	parsedString := ""
	switch len(raw) {
	case 26:
		// agentID is a pre-formatted legacy ULID
		parsedString = string(raw)
	case 16:
		u := uuid.UUID(raw)
		parsedString = u.String()

	default:
		return agentID{}, fmt.Errorf("expected 16 or 26 bytes, got %d", len(raw))
	}

	return agentID{
		raw:    raw,
		parsed: parsedString,
	}, nil
}

func (a agentID) String() string {
	return a.parsed
}

func parseAgentID(instanceUID []byte) (string, error) {
	id, err := newAgentID(instanceUID)
	if err != nil {
		return "", err
	}
	return id.String(), nil
}
