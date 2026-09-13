package template

import "testing"

func TestTitleFunc(t *testing.T) {
	fn, ok := GetFunctionMap()["title"].(func(string) string)
	if !ok {
		t.Fatal("title function missing or has unexpected signature")
	}
	cases := map[string]string{
		"hello world":   "Hello World",
		"order.created": "Order.Created",
		"ALREADY CAPS":  "ALREADY CAPS",
		"":              "",
	}
	for in, want := range cases {
		if got := fn(in); got != want {
			t.Errorf("title(%q) = %q, want %q", in, got, want)
		}
	}
}
