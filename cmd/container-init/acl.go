// Copyright  observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"sort"
)

// POSIX ACL xattr wire format, see include/uapi/linux/posix_acl_xattr.h.
// A value is a little-endian u32 version followed by 8 byte entries of
// {u16 tag, u16 perm, u32 id}, ordered by tag then id.
const (
	aclXattrAccess  = "system.posix_acl_access"
	aclXattrDefault = "system.posix_acl_default"

	aclVersion     uint32 = 2
	aclUndefinedID uint32 = 0xFFFFFFFF
	aclHeaderSize         = 4
	aclEntrySize          = 8

	tagUserObj  uint16 = 0x01
	tagUser     uint16 = 0x02
	tagGroupObj uint16 = 0x04
	tagGroup    uint16 = 0x08
	tagMask     uint16 = 0x10
	tagOther    uint16 = 0x20

	permExec  uint16 = 0x01
	permWrite uint16 = 0x02
	permRead  uint16 = 0x04
)

var errInvalidACL = errors.New("invalid posix acl")

type aclEntry struct {
	tag  uint16
	perm uint16
	id   uint32
}

// acl is a POSIX ACL as stored in the system.posix_acl_* xattrs.
type acl []aclEntry

// decodeACL parses an xattr value into its entries.
func decodeACL(b []byte) (acl, error) {
	if len(b) < aclHeaderSize || (len(b)-aclHeaderSize)%aclEntrySize != 0 {
		return nil, fmt.Errorf("%w: length %d", errInvalidACL, len(b))
	}
	if v := binary.LittleEndian.Uint32(b); v != aclVersion {
		return nil, fmt.Errorf("%w: version %d", errInvalidACL, v)
	}
	n := (len(b) - aclHeaderSize) / aclEntrySize
	out := make(acl, 0, n)
	for i := 0; i < n; i++ {
		off := aclHeaderSize + i*aclEntrySize
		out = append(out, aclEntry{
			tag:  binary.LittleEndian.Uint16(b[off:]),
			perm: binary.LittleEndian.Uint16(b[off+2:]),
			id:   binary.LittleEndian.Uint32(b[off+4:]),
		})
	}
	return out, nil
}

// encode serializes the ACL in the canonical order the kernel requires.
func (a acl) encode() []byte {
	s := a.sorted()
	b := make([]byte, aclHeaderSize+len(s)*aclEntrySize)
	binary.LittleEndian.PutUint32(b, aclVersion)
	for i, e := range s {
		off := aclHeaderSize + i*aclEntrySize
		binary.LittleEndian.PutUint16(b[off:], e.tag)
		binary.LittleEndian.PutUint16(b[off+2:], e.perm)
		binary.LittleEndian.PutUint32(b[off+4:], e.id)
	}
	return b
}

func (a acl) sorted() acl {
	out := append(acl(nil), a...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].tag != out[j].tag {
			return out[i].tag < out[j].tag
		}
		return out[i].id < out[j].id
	})
	return out
}

func (a acl) index(tag uint16, id uint32) int {
	for i, e := range a {
		if e.tag == tag && e.id == id {
			return i
		}
	}
	return -1
}

// aclFromMode builds the minimal ACL equivalent to plain mode bits, which is
// what the kernel uses for an inode without an ACL xattr.
func aclFromMode(mode fs.FileMode) acl {
	perm := uint16(mode.Perm())
	return acl{
		{tag: tagUserObj, perm: (perm >> 6) & 7, id: aclUndefinedID},
		{tag: tagGroupObj, perm: (perm >> 3) & 7, id: aclUndefinedID},
		{tag: tagOther, perm: perm & 7, id: aclUndefinedID},
	}
}

// base returns the owner, owning group and other entries. They seed the
// default ACL of a directory that has none, like setfacl does.
func (a acl) base() acl {
	var out acl
	for _, e := range a {
		if e.tag == tagUserObj || e.tag == tagGroupObj || e.tag == tagOther {
			out = append(out, e)
		}
	}
	return out
}

// grantGroup returns a copy of the ACL with perm added for the named group.
// The mask is widened just enough for the grant to be effective: an existing
// mask gains perm, a missing one is computed from the owning group and the
// named entries, which is setfacl's default mask calculation.
func (a acl) grantGroup(gid uint32, perm uint16) acl {
	out := append(acl(nil), a...)
	if i := out.index(tagGroup, gid); i >= 0 {
		out[i].perm |= perm
	} else {
		out = append(out, aclEntry{tag: tagGroup, perm: perm, id: gid})
	}
	if i := out.index(tagMask, aclUndefinedID); i >= 0 {
		out[i].perm |= perm
	} else {
		var mask uint16
		for _, e := range out {
			if e.tag == tagGroupObj || e.tag == tagUser || e.tag == tagGroup {
				mask |= e.perm
			}
		}
		out = append(out, aclEntry{tag: tagMask, perm: mask, id: aclUndefinedID})
	}
	return out.sorted()
}

// validate mirrors the kernel's posix_acl_valid so a malformed ACL is
// reported with context instead of a bare EINVAL from setxattr.
func (a acl) validate() error {
	var userObj, groupObj, other, mask, named int
	for _, e := range a {
		switch e.tag {
		case tagUserObj:
			userObj++
		case tagGroupObj:
			groupObj++
		case tagOther:
			other++
		case tagMask:
			mask++
		case tagUser, tagGroup:
			if e.id == aclUndefinedID {
				return fmt.Errorf("%w: named entry without id", errInvalidACL)
			}
			named++
		default:
			return fmt.Errorf("%w: unknown tag %#x", errInvalidACL, e.tag)
		}
		if e.perm&^(permRead|permWrite|permExec) != 0 {
			return fmt.Errorf("%w: perm %#x", errInvalidACL, e.perm)
		}
	}
	if userObj != 1 || groupObj != 1 || other != 1 {
		return fmt.Errorf("%w: owner, owning group and other entries are required exactly once", errInvalidACL)
	}
	if mask > 1 || (named > 0 && mask == 0) {
		return fmt.Errorf("%w: exactly one mask entry is required with named entries", errInvalidACL)
	}
	return nil
}
