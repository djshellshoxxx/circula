package messages

import (
	"bytes"
	"html"
	"html/template"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type roleListPage struct {
	D             any
	Slug, NetName string
}

func (roleListPage) Val(string) string { return "" }

// Render actual table content; the shared shell and pagination are separate contracts.
func TestMemberRolesAreSemanticLists(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	src, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "templates", "admin_members.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		roles []string
	}{
		{"none", nil}, {"one", []string{Text("admin.members.role.admin")}},
		{"two", []string{Text("admin.members.role.admin"), Text("admin.members.role.reviewer")}},
		{"literal unknown", []string{`future 100% %! <img src=x> & role`, Text("admin.members.role.admin")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmpl, err := template.New("members").Funcs(template.FuncMap{"msg": Text, "dict": func(...any) any { return nil }}).Parse(string(src) + `{{define "pagenav"}}{{end}}`)
			if err != nil {
				t.Fatal(err)
			}
			row := map[string]any{"ID": "member-id", "DisplayName": "A", "StatusLabel": "active", "Roles": tc.roles}
			p := roleListPage{D: map[string]any{"Rows": []any{row}, "Pages": map[string]any{"Total": 1}}, Slug: "town", NetName: "Town"}
			var out bytes.Buffer
			if err := tmpl.ExecuteTemplate(&out, "content", p); err != nil {
				t.Fatal(err)
			}
			body := out.String()
			start := strings.Index(body, `<td data-label="Roles">`)
			if start < 0 {
				t.Fatal("missing roles cell")
			}
			cell := strings.SplitN(body[start:], "</td>", 2)[0]
			if len(tc.roles) == 0 {
				if !strings.Contains(cell, ">member") || strings.Contains(cell, "<ul") {
					t.Fatalf("member fallback changed: %s", cell)
				}
				return
			}
			want := `<ul class="role-list">`
			for _, role := range tc.roles {
				want += "<li>" + html.EscapeString(role) + "</li>"
			}
			want += "</ul>"
			if !strings.Contains(cell, want) {
				t.Fatalf("roles need separate escaped list items in source order: got %s; want %s", cell, want)
			}
			if strings.Contains(cell, "<img") {
				t.Fatal("role data became markup")
			}
		})
	}
}
