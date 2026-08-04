package secregtest

import "testing"

func TestCheckFields(t *testing.T) {
	body := []byte(`{"data":{"name":"x","cost":null,"items":[{"qty":1,"cost":5000},{"qty":2}]}}`)

	cases := []struct {
		name     string
		fields   []Field
		wantErrs int
	}{
		{"visible present non-null", []Field{Visible("data.name")}, 0},
		{"visible null fails", []Field{Visible("data.cost")}, 1},
		{"visible missing fails", []Field{Visible("data.missing")}, 1},
		{"null present", []Field{Null("data.cost")}, 0},
		{"null on non-null fails", []Field{Null("data.items.0.cost")}, 1},
		{"null missing fails", []Field{Null("data.missing")}, 1},
		{"absent missing", []Field{Absent("data.missing")}, 0},
		{"absent present fails", []Field{Absent("data.name")}, 1},
		{"absent null value is still present", []Field{Absent("data.cost")}, 1},
		{"array index path", []Field{Visible("data.items.1.qty")}, 0},
		{"array index out of range fails", []Field{Visible("data.items.2.qty")}, 1},
		{"nested via array", []Field{Visible("data.items.0.cost")}, 0},
		{"array index absent in element", []Field{Absent("data.items.1.cost")}, 0},
		{"multiple violations counted", []Field{Visible("data.missing"), Null("data.name")}, 2},
		{"invalid json body", []Field{Visible("data.name")}, 1},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var bodyForCase = body
			if c.name == "invalid json body" {
				bodyForCase = []byte(`{not-json`)
			}
			errs := checkErr(bodyForCase, c.fields...)
			if len(errs) != c.wantErrs {
				t.Fatalf("got %d errors, want %d: %v", len(errs), c.wantErrs, errs)
			}
		})
	}
}

func TestCheck_ReportsViolations(t *testing.T) {
	t.Run("all satisfied passes", func(t *testing.T) {
		Check(t, []byte(`{"data":{"name":"x"}}`), Visible("data.name"), Absent("data.cost"))
	})

	t.Run("missing path does not panic", func(t *testing.T) {
		Check(t, []byte(`{"data":{}}`), Absent("data.cost"), Absent("data.items.0.name"), Absent("data.items.1.cost"))
	})
}
