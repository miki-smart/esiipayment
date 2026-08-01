package expr

import "testing"

// These are regression tests for two edge cases found while expanding
// vectors/expressions/extraction.json and vectors/money/conversions.json
// with int64-boundary vectors: both are cases a hand-reasoned vector
// caught that no existing test exercised.

func TestExtractBareDollarIsInvalid(t *testing.T) {
	doc := map[string]interface{}{"status": "success"}
	if _, ok := Extract(doc, "$"); ok {
		t.Fatal("Extract(doc, \"$\") should not resolve: a bare $ with no .field or [index] segment is not a valid path in this grammar")
	}
}

func TestApplyTransformAmountMajorInt64Boundary(t *testing.T) {
	cases := []struct {
		name      string
		minorUnits int64
		exponent  int
		want      string
	}{
		{"int64 max, exponent 2", 9223372036854775807, 2, "92233720368547758.07"},
		{"int64 min, exponent 2", -9223372036854775808, 2, "-92233720368547758.08"},
		{"int64 max, exponent 0", 9223372036854775807, 0, "9223372036854775807"},
	}
	for _, c := range cases {
		got, err := ApplyTransform("amount_major", c.minorUnits, c.exponent)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", c.name, err)
		}
		if got != c.want {
			t.Fatalf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}
