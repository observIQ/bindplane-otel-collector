package observiq

import (
	"testing"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/require"
)

func TestCreateDefaultConfig(t *testing.T) {
	config := createDefaultConfig()
	require.Equal(t, typeStr, config.ID().String())

	observiqConfig, ok := config.(*Config)
	require.True(t, ok)
	require.Equal(t, endpoint, observiqConfig.Endpoint)
	require.Equal(t, statusInterval, observiqConfig.StatusInterval)
	require.Equal(t, reconnectInterval, observiqConfig.ReconnectInterval)
}

func TestConfigUnmarshal(t *testing.T) {
	configMap := map[string]interface{}{
		"endpoint":           "endpoint-value",
		"agent_name":         "name-value",
		"agent_id":           "id-value",
		"status_interval":    time.Second * 5,
		"reconnect_interval": time.Minute * 30,
		"template_id":        "template-value",
	}

	config := &Config{}
	err := mapstructure.Decode(configMap, config)
	require.NoError(t, err)

	require.Equal(t, "endpoint-value", config.Endpoint)
	require.Equal(t, "name-value", config.AgentName)
	require.Equal(t, "id-value", config.AgentID)
	require.Equal(t, time.Second*5, config.StatusInterval)
	require.Equal(t, time.Minute*30, config.ReconnectInterval)
	require.Equal(t, "template-value", config.TemplateID)
}
