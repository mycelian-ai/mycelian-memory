package types

import "testing"

func TestValidateUserID(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in string
		ok bool
	}{
		{"a", true}, {"user_1", true}, {"this_is_20_chars____", true}, {"", false}, {"UPPER", false}, {"has-dash", false},
	}
	for _, c := range cases {
		err := ValidateUserID(c.in)
		if c.ok && err != nil {
			t.Fatalf("expected ok for %q, got %v", c.in, err)
		}
		if !c.ok && err == nil {
			t.Fatalf("expected error for %q", c.in)
		}
	}
}
