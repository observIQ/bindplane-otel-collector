//go:build windows

package etw

import (
	"sync"

	"golang.org/x/sys/windows"
)

type Provider struct {
	Name        string
	GUID        string
	EnableLevel uint8
}

func baseNewProvider() *Provider {
	return &Provider{
		EnableLevel: 0xff,
	}
}

func (p *Provider) Enable(session *Session) error {
	// err := advapi32.StartTrace(session.handle, p.GUID, session.properties)
	// if err != nil {
	// 	return err
	// }
	return nil
}

// ParseProvider attempts to find a provider by name or GUID
func ParseProvider(nameOrGuid string) (*Provider, error) {
	// First, try to parse as a GUID
	_, err := windows.GUIDFromString(nameOrGuid)
	if err == nil {
		// Valid GUID format
		return &Provider{
			Name: nameOrGuid,
			GUID: nameOrGuid,
		}, nil
	}

	// If not a GUID, look it up in the provider map
	if provider, ok := getProviderByName(nameOrGuid); ok {
		return provider, nil
	}

	// Return a generic provider with the name
	// In a real implementation, we would try to look up the GUID
	return &Provider{
		Name: nameOrGuid,
		GUID: "", // This would need to be populated in a real implementation
	}, nil
}

var (
	// Use a different variable name to avoid conflict with session.go
	providerNameMapOnce sync.Once
	providerNameMap     map[string]*Provider
)

func getProviderByName(name string) (*Provider, bool) {
	providerNameMapOnce.Do(func() {
		// Initialize the provider map
		providerNameMap = map[string]*Provider{
			// Some common providers for Windows ETW
			"Microsoft-Windows-Kernel-Process": &Provider{
				Name:        "Microsoft-Windows-Kernel-Process",
				GUID:        "{22FB2CD6-0E7B-422B-A0C7-2FAD1FD0E716}",
				EnableLevel: 0xff,
			},
			"Microsoft-Windows-DNS-Client": &Provider{
				Name:        "Microsoft-Windows-DNS-Client",
				GUID:        "{1C95126E-7EEA-49A9-A3FE-A378B03DDB4D}",
				EnableLevel: 0xff,
			},
			"Microsoft-Windows-Security-Auditing": &Provider{
				Name:        "Microsoft-Windows-Security-Auditing",
				GUID:        "{54849625-5478-4994-A5BA-3E3B0328C30D}",
				EnableLevel: 0xff,
			},
		}
	})

	provider, exists := providerNameMap[name]
	return provider, exists
}
