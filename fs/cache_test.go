package fs

import (
	"bytes"
	"testing"
)

type input struct {
	offset int64
	data   string
}

func TestCacheWrites(t *testing.T) {
	tests := []struct {
		writes []input
		expect []string
	}{
		{[]input{{0, "test"}}, []string{"test"}},
		{[]input{
			{0, "test"},
			{0, "test"},
		}, []string{
			"test",
		}},
		{[]input{
			{0, "test"},
			{4, "test"},
		}, []string{
			"testtest",
		}},
		{[]input{
			{0, "test"},
			{5, "test"},
		}, []string{
			"test",
			"test",
		}},
		{[]input{
			{0, "test"},
			{1, "--"},
		}, []string{
			"t--t",
		}},
		{[]input{
			{0, "test"},
			{5, "test"},
			{10, "test"},
			{4, "------"},
		}, []string{
			"test------test",
		}},
		{[]input{
			{0, "test"},
			{5, "test"},
			{10, "test"},
			{4, "-"},
			{9, "-"},
		}, []string{
			"test-test-test",
		}},
	}

	for n, test := range tests {
		c := &CacheFile{
			stat:    nil,
			deleted: false,
			slices:  nil,
		}
		for _, w := range test.writes {
			c.Write([]byte(w.data), w.offset)
		}
		for i, s := range c.slices {
			if !bytes.Equal(s.data, []byte(test.expect[i])) {
				t.Fatalf("Test %v: Expexted slice %v to be %v, got %v instead\n",
					n, i, test.expect[i], string(s.data))
			}
		}
	}
}
