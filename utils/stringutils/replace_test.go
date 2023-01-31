package stringutils

import "testing"

func Test_RemoveMultipleSpaces(t *testing.T) {
	source := "a b  c    d"
	expected := "a b c d"

	result := RemoveMultipleSpaces(source)

	if result != expected {
		t.Errorf("%s: expected but got %s", expected, result)
	}

	source = "a b c d"
	result = RemoveMultipleSpaces(source)
	if result != source {
		t.Errorf("%s: expected but got %s", source, result)
	}

	source = "abc d"
	result = RemoveMultipleSpaces(source)
	if result != source {
		t.Errorf("%s: expected but got %s", source, result)
	}

}
