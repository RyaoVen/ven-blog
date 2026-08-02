package moderator

import (
	"strings"
	"testing"

	"ven_hybird/build/application/moderationapp"
)

// fakeInvalidator 记录失效调用。
type fakeInvalidator struct {
	pages    []string
	patterns []string
}

func (f *fakeInvalidator) InvalidatePage(path string) {
	f.pages = append(f.pages, path)
}

func (f *fakeInvalidator) DataChange(pattern string, params ...string) error {
	f.patterns = append(f.patterns, pattern+" "+strings.Join(params, " "))
	return nil
}

func authorNameFn() string { return "ryao" }

func TestApplyInvalidations(t *testing.T) {
	cases := []struct {
		name  string
		r     *moderationapp.Result
		pages []string
		data  []string
	}{
		{
			name: "post comment approved invalidates posts and detail",
			r: &moderationapp.Result{ApprovedItems: []moderationapp.Item{
				{Kind: moderationapp.KindComment, PostID: 5},
			}},
			pages: []string{"/posts", "/"},
			data:  []string{"/posts/:id 5"},
		},
		{
			name: "moment comment approved invalidates moments",
			r: &moderationapp.Result{ApprovedItems: []moderationapp.Item{
				{Kind: moderationapp.KindComment, MomentID: 8},
			}},
			data: []string{"/moments "},
		},
		{
			name: "guestbook approved invalidates author page",
			r: &moderationapp.Result{ApprovedItems: []moderationapp.Item{
				{Kind: moderationapp.KindGuestbook, ID: 1},
			}},
			pages: []string{"/author/ryao"},
		},
		{
			name: "guestbook rejected invalidates author page",
			r: &moderationapp.Result{RejectedItems: []moderationapp.Item{
				{Kind: moderationapp.KindGuestbook, ID: 2},
			}},
			pages: []string{"/author/ryao"},
		},
		{
			name: "comment rejected invalidates nothing",
			r: &moderationapp.Result{RejectedItems: []moderationapp.Item{
				{Kind: moderationapp.KindComment, ID: 3, PostID: 5},
			}},
		},
		{
			name: "nil result invalidates nothing",
			r:    nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inv := &fakeInvalidator{}
			applyInvalidations(inv, authorNameFn, tc.r)
			if !equalStrings(inv.pages, tc.pages) {
				t.Fatalf("pages = %v, want %v", inv.pages, tc.pages)
			}
			if !equalStrings(inv.patterns, tc.data) {
				t.Fatalf("data changes = %v, want %v", inv.patterns, tc.data)
			}
		})
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
