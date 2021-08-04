package message

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewMessageSuccess(t *testing.T) {
	msgType := "testMsg"
	msgContent := struct{}{}
	msg, err := NewMessage(msgType, msgContent)
	require.NoError(t, err)
	require.Equal(t, msgType, msg.Type)
	require.Equal(t, map[string]interface{}{}, msg.Content)
}

func TestNewMessageFailure(t *testing.T) {
	msgType := "testMsg"
	msgContent := make(chan int)
	msg, err := NewMessage(msgType, msgContent)
	require.Error(t, err)
	require.Nil(t, msg)
}
