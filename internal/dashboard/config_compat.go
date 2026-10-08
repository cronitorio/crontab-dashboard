package dashboard

import (
	"encoding/json"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"strings"
)

func clearManagedAuthMetadataIfReplacingKey(cmd *cobra.Command) {
	stored := readStoredAuthFromConfig()
	replacing := false
	if flag := cmd.Flags().Lookup("api-key"); flag != nil && flag.Changed {
		replacing = true
	}
	if envKey, exists := os.LookupEnv(varApiKey); exists {
		if envKey = strings.TrimSpace(envKey); envKey != "" && envKey != stored.APIKey {
			replacing = true
		}
	}
	if strings.TrimSpace(viper.GetString(varApiKey)) != stored.APIKey {
		replacing = true
	}
	if !replacing {
		return
	}
	viper.Set(varAuthManaged, false)
	viper.Set(varMachineCredentialName, "")
}

const defaultPingApiHost = "https://cronitor.link"
const fallbackPingApiHost = "https://cronitor.io"

// normalizePingApiHost turns CRONITOR_PING_API_HOST into a base URL. A bare
// host such as eu.cronitor.link gets https://; an empty value means the default.
func normalizePingApiHost(host string) string {
	host = strings.TrimRight(strings.TrimSpace(host), "/")
	if host == "" {
		return defaultPingApiHost
	}
	if !strings.Contains(host, "://") {
		host = "https://" + host
	}
	return host
}

type storedAuth struct {
	APIKey  string
	Managed bool
	Name    string
}

func readStoredAuthFromConfig() storedAuth {
	var out storedAuth
	data, err := os.ReadFile(configFilePath())
	if err != nil {
		return out
	}
	var raw map[string]interface{}
	if json.Unmarshal(data, &raw) != nil {
		return out
	}
	for k, v := range raw {
		switch {
		case strings.EqualFold(k, varApiKey):
			if s, ok := v.(string); ok {
				out.APIKey = strings.TrimSpace(s)
			}
		case strings.EqualFold(k, varAuthManaged):
			out.Managed, _ = v.(bool)
		case strings.EqualFold(k, varMachineCredentialName):
			if s, ok := v.(string); ok {
				out.Name = strings.TrimSpace(s)
			}
		}
	}
	return out
}
