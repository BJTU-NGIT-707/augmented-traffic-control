package main

import (
	"runtime"
	"testing"

	"github.com/facebook/augmented-traffic-control/atc/atc_thrift/atc_thrift"
)

func TestAtcdCreatesGroup(_t *testing.T) {
	t := Setup(_t, true)
	defer t.Cleanup()
	grp := t.CreateGroup("1.2.3.4")

	t.checkMembers(grp, "1.2.3.4")

	if grp.Shaping != nil {
		t.Errorf("Created group had shaping: %+v", grp.Shaping)
	}
}

func TestAtcdJoinGroup(_t *testing.T) {
	t := Setup(_t, true)
	defer t.Cleanup()
	grp := t.CreateGroup("1.2.3.4")
	token := t.token(grp)

	t.JoinGroup(grp.Id, "4.3.2.1", token)

	grp = t.GetGroup(grp.Id)
	t.checkMembers(grp, "1.2.3.4", "4.3.2.1")
}

func TestAtcdLeaveGroup(_t *testing.T) {
	t := Setup(_t, true)
	defer t.Cleanup()
	grp := t.CreateGroup("1.2.3.4")
	token := t.token(grp)
	t.JoinGroup(grp.Id, "4.3.2.1", token) // Empty groups are deleted

	t.LeaveGroup(grp.Id, "1.2.3.4", token)

	grp = t.GetGroup(grp.Id)
	t.checkMembers(grp, "4.3.2.1")
}

func TestAtcdCleansEmptyGroups(_t *testing.T) {
	t := Setup(_t, true)
	defer t.Cleanup()
	grp := t.CreateGroup("1.2.3.4")
	token := t.token(grp)
	t.LeaveGroup(grp.Id, "1.2.3.4", token)

	// Allow other goroutines to run
	runtime.Gosched()

	grp, err := t.atcd.GetGroup(grp.Id)
	// specific err
	if err == nil {
		t.Fatalf("Group should have been deleted: %+v", grp)
	}
}

func TestAtcdBadTokenJoin(_t *testing.T) {
	t := Setup(_t, true)
	defer t.Cleanup()
	grp := t.CreateGroup("1.2.3.4")
	token := t.token(grp)

	err := t.atcd.JoinGroup(grp.Id, "4.3.2.1", token+"invalid")
	if err == nil {
		t.Errorf("Joined group with bad token")
	}

	grp = t.GetGroup(grp.Id)
	t.checkMembers(grp, "1.2.3.4")
}

func TestAtcdBadTokenLeave(_t *testing.T) {
	t := Setup(_t, true)
	defer t.Cleanup()
	grp := t.CreateGroup("1.2.3.4")
	token := t.token(grp)

	err := t.atcd.LeaveGroup(grp.Id, "1.2.3.4", token+"invalid")
	if err == nil {
		t.Errorf("Left group with bad token")
	}

	grp = t.GetGroup(grp.Id)
	t.checkMembers(grp, "1.2.3.4")
}

func TestAtcdGetGroupWith(_t *testing.T) {
	t := Setup(_t, true)
	defer t.Cleanup()
	grp := t.CreateGroup("1.2.3.4")

	grp2 := t.GetGroupWith("1.2.3.4")

	if grp2.Id != grp.Id {
		t.Errorf("Group IDs don't match: %d != %d", grp.Id, grp2.Id)
	}
	t.checkMembers(grp2, "1.2.3.4")
}

/**
*** Test utils below this point
**/

type testAtcd struct {
	*testing.T
	atcd *Atcd
}

func Setup(t *testing.T, secure bool) *testAtcd {
	db, err := NewDbRunner("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	return &testAtcd{
		T:    t,
		atcd: NewAtcd(db, FakeShaper{}, secure).(*Atcd),
	}
}

func (t *testAtcd) Cleanup() {
	t.atcd.db.Close()
}

func (t *testAtcd) check(err error) {
	if err != nil {
		t.Fatal(err)
	}
}

func (t *testAtcd) CreateGroup(member string) *atc_thrift.ShapingGroup {
	grp, err := t.atcd.CreateGroup(member)
	t.check(err)
	return grp
}

func (t *testAtcd) GetGroup(id int64) *atc_thrift.ShapingGroup {
	grp, err := t.atcd.GetGroup(id)
	t.check(err)
	return grp
}

func (t *testAtcd) GetGroupWith(member string) *atc_thrift.ShapingGroup {
	grp, err := t.atcd.GetGroupWith(member)
	t.check(err)
	return grp
}

func (t *testAtcd) JoinGroup(id int64, member, token string) {
	t.check(t.atcd.JoinGroup(id, member, token))
}

func (t *testAtcd) LeaveGroup(id int64, member, token string) {
	t.check(t.atcd.LeaveGroup(id, member, token))
}

func (t *testAtcd) token(grp *atc_thrift.ShapingGroup) string {
	token, err := t.atcd.GetGroupToken(grp.Id)
	t.check(err)
	return token
}

func (t *testAtcd) checkMembers(grp *atc_thrift.ShapingGroup, members ...string) {
	if len(grp.Members) != len(members) {
		t.Errorf("Wrong number of members: %d != %d", len(members), len(grp.Members))
	}
	// list is not ordered in any specific way so we have to be a little clever
	x := make(map[string]struct{})
	for _, addr := range grp.Members {
		x[addr] = struct{}{}
	}
	for _, addr := range members {
		if _, ok := x[addr]; !ok {
			t.Errorf(`Group doesn't contain: %q`, addr)
		}
	}
}
