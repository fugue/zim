package format

import (
	"testing"
)

type item struct {
	Name           string
	Age            int
	Married        bool
	FavoriteNumber int
}

func TestFormatTable(t *testing.T) {

	items := []interface{}{
		item{"hank", 32, true, 42},
		item{"peggy", 31, true, 43},
		item{"bobby", 1, false, 44},
	}

	rows, err := Table(TableOpts{
		Rows:       items,
		Columns:    []string{"Name", "Age", "FavoriteNumber"},
		ShowHeader: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"=============================",
		"NAME  | AGE | FAVORITE_NUMBER",
		"=============================",
		"hank  | 32  | 42             ",
		"peggy | 31  | 43             ",
		"bobby | 1   | 44             ",
	}

	for i, row := range rows {
		if row != expected[i] {
			t.Errorf("Got: '%s' Expected: '%s'", row, expected[i])
		}
	}
}

func TestFormatTableNoHeader(t *testing.T) {

	items := []interface{}{
		item{"a", 0, true, 0},
		item{"abcd", 31, true, 0},
		item{"abcdef", 1, false, 0},
	}

	rows, err := Table(TableOpts{
		Rows:      items,
		Columns:   []string{"Name", "Married"},
		Separator: " . ",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"a      . true ",
		"abcd   . true ",
		"abcdef . false",
	}

	for i, row := range rows {
		if row != expected[i] {
			t.Errorf("Got: '%s' Expected: '%s'", row, expected[i])
		}
	}
}
