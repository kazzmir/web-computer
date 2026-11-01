package html

import "testing"

func TestUnescapeSinglePass(t *testing.T) {
	for input, want := range map[string]string{
		"&#38;amp;": "&amp;", "&amp;lt;": "&lt;", "&#x26;#65;": "&#65;",
		"&notin;": "∉", "&acE;": "∾̳", "&#xFDD0;": "", "&#x1FFFE;": "",
		"&#999999999999999999999999999999999;": "�", "&&amp;": "&&",
	} {
		if got := unescape(input); got != want {
			t.Errorf("unescape(%q)=%q,want%q", input, got, want)
		}
	}
}
