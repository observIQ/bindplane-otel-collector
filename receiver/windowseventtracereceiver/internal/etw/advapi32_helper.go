//go:build windows
// +build windows

package etw

import (
	"unsafe"
)

func GetAccessString(guid string) (s string, err error) {

	g := MustParseGUIDFromString(guid)
	bSize := uint32(0)
	// retrieves size
	EventAccessQuery(g, nil, &bSize)
	buffer := make([]byte, bSize)
	sd := (*SecurityDescriptor)(unsafe.Pointer(&buffer[0]))
	// we get the security descriptor
	EventAccessQuery(g, sd, &bSize)

	if s, err = ConvertSecurityDescriptorToStringSecurityDescriptorW(
		sd,
		SDDL_REVISION_1,
		DACL_SECURITY_INFORMATION); err != nil {
		return
	}

	return
}

/*func AddProviderAccess(guid, sid string, rights uint32) (err error) {
	var psid *SID

	if psid, err = ConvertStringSidToSidW(sid); err != nil {
		log.Errorf("Failed to convert string to sid: %s", err)
		return
	}

	g := MustGUIDFromString(guid)

	return EventAccessControl(g,
		uint32(EVENT_SECURITY_ADD_DACL),
		psid,
		rights,
		true,
	)
}*/
