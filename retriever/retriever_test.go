package retriever

import "testing"

type containsTest struct {
	stringArray []string
	token       string
	expected    bool
}

type concatSpaceTest struct {
	s         string
	connector string
	expected  string
}

var containsTests = []containsTest{
	{
		[]string{"This", "is", "a", "test", "string"},
		"test",
		true,
	},
	{
		[]string{"日本語", "でも", "おｋ"},
		"日本語",
		true,
	},
	{
		[]string{"This", "string", "array", "may", "fail"},
		"that",
		false,
	},
	{
		[]string{"中国語", "では", "ない"},
		"ある",
		false,
	},
}

var concatSpaceTests = []concatSpaceTest{
	{
		"this is a test",
		"_",
		"this_is_a_test",
	},
	{
		"that\tis\talso\ttest",
		"-",
		"that-is-also-test",
	},
	{
		"    これは　テスト　です　　　 ",
		"_",
		"これは_テスト_です",
	},
}

func TestContains(t *testing.T) {
	for _, ct := range containsTests {
		actual := Contains(ct.stringArray, ct.token)
		if ct.expected != actual {
			t.Errorf("%q token %s: want %t got %t", ct.stringArray, ct.token, ct.expected, actual)
			continue
		}
	}
}

func TestConcatSpace(t *testing.T) {
	for _, cst := range concatSpaceTests {
		actual := ConcatSpace(cst.s, cst.connector)
		if cst.expected != actual {
			t.Errorf("%s connector %s: want %s got %s", cst.s, cst.connector, cst.expected, actual)
			continue
		}
	}
}
