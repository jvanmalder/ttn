// Copyright © 2017 The Things Network
// Use of this source code is governed by the MIT license that can be found in the LICENSE file.

package types

import (
	"testing"

	s "github.com/smartystreets/assertions"
)

func TestAccessKeysRights(t *testing.T) {
	a := s.New(t)
	c := AccessKey{
		Name: "name",
		Key:  "123456",
		Rights: []Right{
			Right("right"),
		},
	}

	a.So(c.HasRight(Right("right")), s.ShouldBeTrue)
	a.So(c.HasRight(Right("foo")), s.ShouldBeFalse)
}

func TestRightString(t *testing.T) {
	a := s.New(t)

	right := Right("foo")
	a.So(right.String(), s.ShouldEqual, "foo")
}
