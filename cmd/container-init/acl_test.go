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
	"bytes"
	"errors"
	"testing"
)

// u::rw-,g::r--,o::--- as getfacl would print it, encoded by hand.
var mode0640ACL = []byte{
	0x02, 0x00, 0x00, 0x00, // version 2
	0x01, 0x00, 0x06, 0x00, 0xff, 0xff, 0xff, 0xff, // USER_OBJ rw
	0x04, 0x00, 0x04, 0x00, 0xff, 0xff, 0xff, 0xff, // GROUP_OBJ r
	0x20, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, // OTHER -
}

func TestDecodeEncodeRoundTrip(t *testing.T) {
	a, err := decodeACL(mode0640ACL)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	want := acl{
		{tag: tagUserObj, perm: permRead | permWrite, id: aclUndefinedID},
		{tag: tagGroupObj, perm: permRead, id: aclUndefinedID},
		{tag: tagOther, perm: 0, id: aclUndefinedID},
	}
	requireACL(t, want, a)
	if got := a.encode(); !bytes.Equal(got, mode0640ACL) {
		t.Fatalf("encode: got %x, want %x", got, mode0640ACL)
	}
}

func TestDecodeRejectsMalformed(t *testing.T) {
	cases := map[string][]byte{
		"short":         {0x02, 0x00},
		"bad version":   {0x01, 0x00, 0x00, 0x00},
		"partial entry": append(append([]byte{}, mode0640ACL...), 0x01, 0x00),
	}
	for name, b := range cases {
		if _, err := decodeACL(b); !errors.Is(err, errInvalidACL) {
			t.Errorf("%s: got %v, want errInvalidACL", name, err)
		}
	}
}

func TestACLFromMode(t *testing.T) {
	requireACL(t, acl{
		{tag: tagUserObj, perm: 7, id: aclUndefinedID},
		{tag: tagGroupObj, perm: 5, id: aclUndefinedID},
		{tag: tagOther, perm: 5, id: aclUndefinedID},
	}, aclFromMode(0755))

	a, _ := decodeACL(mode0640ACL)
	requireACL(t, a, aclFromMode(0640))
}

func TestGrantGroupCreatesEntryAndMask(t *testing.T) {
	got := aclFromMode(0640).grantGroup(4242, permRead)
	requireACL(t, acl{
		{tag: tagUserObj, perm: permRead | permWrite, id: aclUndefinedID},
		{tag: tagGroupObj, perm: permRead, id: aclUndefinedID},
		{tag: tagGroup, perm: permRead, id: 4242},
		{tag: tagMask, perm: permRead, id: aclUndefinedID},
		{tag: tagOther, perm: 0, id: aclUndefinedID},
	}, got)
	if err := got.validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	// A 0600 file has an empty owning group; the mask must still admit the grant.
	got = aclFromMode(0600).grantGroup(4242, permRead)
	if i := got.index(tagMask, aclUndefinedID); i < 0 || got[i].perm != permRead {
		t.Fatalf("mask on 0600 file: %+v", got)
	}
}

func TestGrantGroupIsIdempotent(t *testing.T) {
	once := aclFromMode(0755).grantGroup(4242, permRead|permExec)
	twice := once.grantGroup(4242, permRead|permExec)
	if !bytes.Equal(once.encode(), twice.encode()) {
		t.Fatalf("second grant changed the acl: %+v vs %+v", once, twice)
	}
}

func TestGrantGroupWidensExistingMaskMinimally(t *testing.T) {
	// user:7:rwx exists but is masked down to r--; granting r-x must widen the
	// mask to r-x without restoring write for the other entry.
	existing := acl{
		{tag: tagUserObj, perm: 7, id: aclUndefinedID},
		{tag: tagUser, perm: 7, id: 7},
		{tag: tagGroupObj, perm: 5, id: aclUndefinedID},
		{tag: tagMask, perm: permRead, id: aclUndefinedID},
		{tag: tagOther, perm: 0, id: aclUndefinedID},
	}
	got := existing.grantGroup(4242, permRead|permExec)
	if i := got.index(tagMask, aclUndefinedID); got[i].perm != permRead|permExec {
		t.Fatalf("mask: got %#o, want r-x", got[i].perm)
	}
	if i := got.index(tagUser, 7); got[i].perm != 7 {
		t.Fatalf("existing user entry was modified: %+v", got[i])
	}
	if i := got.index(tagGroup, 4242); got[i].perm != permRead|permExec {
		t.Fatalf("group entry: %+v", got[i])
	}
}

func TestGrantGroupKeepsExistingGroupPerms(t *testing.T) {
	existing := aclFromMode(0750).grantGroup(4242, permRead|permWrite|permExec)
	got := existing.grantGroup(4242, permRead)
	if i := got.index(tagGroup, 4242); got[i].perm != 7 {
		t.Fatalf("group perms were reduced: %+v", got[i])
	}
}

func TestEncodeSortsEntries(t *testing.T) {
	unsorted := acl{
		{tag: tagOther, perm: 0, id: aclUndefinedID},
		{tag: tagGroup, perm: 4, id: 9},
		{tag: tagGroup, perm: 4, id: 3},
		{tag: tagMask, perm: 4, id: aclUndefinedID},
		{tag: tagGroupObj, perm: 4, id: aclUndefinedID},
		{tag: tagUserObj, perm: 6, id: aclUndefinedID},
	}
	got, err := decodeACL(unsorted.encode())
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	wantTags := []uint16{tagUserObj, tagGroupObj, tagGroup, tagGroup, tagMask, tagOther}
	for i, e := range got {
		if e.tag != wantTags[i] {
			t.Fatalf("entry %d: tag %#x, want %#x (%+v)", i, e.tag, wantTags[i], got)
		}
	}
	if got[2].id != 3 || got[3].id != 9 {
		t.Fatalf("named groups not sorted by id: %+v", got)
	}
}

func TestValidate(t *testing.T) {
	cases := map[string]struct {
		acl     acl
		wantErr bool
	}{
		"mode only":       {aclFromMode(0644), false},
		"named with mask": {aclFromMode(0644).grantGroup(1, permRead), false},
		"named without mask": {acl{
			{tag: tagUserObj, perm: 6, id: aclUndefinedID},
			{tag: tagGroupObj, perm: 4, id: aclUndefinedID},
			{tag: tagGroup, perm: 4, id: 1},
			{tag: tagOther, perm: 4, id: aclUndefinedID},
		}, true},
		"missing other": {acl{
			{tag: tagUserObj, perm: 6, id: aclUndefinedID},
			{tag: tagGroupObj, perm: 4, id: aclUndefinedID},
		}, true},
		"bad perm": {acl{
			{tag: tagUserObj, perm: 8, id: aclUndefinedID},
			{tag: tagGroupObj, perm: 4, id: aclUndefinedID},
			{tag: tagOther, perm: 4, id: aclUndefinedID},
		}, true},
		"unknown tag": {acl{
			{tag: 0x40, perm: 4, id: aclUndefinedID},
			{tag: tagUserObj, perm: 6, id: aclUndefinedID},
			{tag: tagGroupObj, perm: 4, id: aclUndefinedID},
			{tag: tagOther, perm: 4, id: aclUndefinedID},
		}, true},
	}
	for name, tc := range cases {
		err := tc.acl.validate()
		if (err != nil) != tc.wantErr {
			t.Errorf("%s: got %v, wantErr %v", name, err, tc.wantErr)
		}
	}
}

func requireACL(t *testing.T, want, got acl) {
	t.Helper()
	want, got = want.sorted(), got.sorted()
	if len(want) != len(got) {
		t.Fatalf("acl length: got %+v, want %+v", got, want)
	}
	for i := range want {
		if want[i] != got[i] {
			t.Fatalf("acl entry %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}
