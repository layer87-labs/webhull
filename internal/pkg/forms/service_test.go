package forms

import "testing"

func TestRenderAutoReply_RemovesEveryPlaceholderButContactID(t *testing.T) {
	got := renderAutoReply("Hi {{.Name}} {{FIRST-NAME}} {{MESSAGE}}, ref {{CONTACT_ID}}", "a1b2c3d4")
	want := "Hi   , ref a1b2c3d4"
	if got != want {
		t.Fatalf("renderAutoReply = %q, want %q", got, want)
	}
}
