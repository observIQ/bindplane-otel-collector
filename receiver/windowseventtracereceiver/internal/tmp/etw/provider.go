//go:build windows

package etw

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
