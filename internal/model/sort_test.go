package model

import "testing"

func TestSortPinnedAndMove(t *testing.T) {
	accs := []MFAAccount{
		{AccountName: "a"},
		{AccountName: "b", Pinned: true},
		{AccountName: "c"},
		{AccountName: "d", Pinned: true},
	}
	SortPinned(accs)
	if accs[0].AccountName != "b" || accs[1].AccountName != "d" {
		t.Fatalf("pinned order: %+v", accs)
	}

	accs = MoveAccount(accs, 0, 1)
	if accs[0].AccountName != "d" || accs[1].AccountName != "b" {
		t.Fatalf("move within pinned: %+v", accs)
	}

	// 跨组不应移动
	before := accs[1].AccountName
	accs = MoveAccount(accs, 1, 1)
	if accs[1].AccountName != before {
		t.Fatalf("should not cross pin group: %+v", accs)
	}
}

func TestFindAccount(t *testing.T) {
	accs := []MFAAccount{{AccountName: "a", Secret: "s"}}
	if FindAccount(accs, "a", "s") != 0 {
		t.Fatal("should find")
	}
	if FindAccount(accs, "a", "x") != -1 {
		t.Fatal("should miss")
	}
}
