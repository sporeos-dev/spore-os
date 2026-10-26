package file

import "testing"

func TestExecutablePathFor(t *testing.T) {
	cases := []struct{ goos, in, want string }{
		{"windows", `C:\n\foundry-alias`, `C:\n\foundry-alias.exe`},
		{"windows", `C:\n\foundry-alias.exe`, `C:\n\foundry-alias.exe`},
		{"windows", "", ""},
		{"linux", "/n/foundry-alias", "/n/foundry-alias"},
		{"darwin", "/n/foundry-alias", "/n/foundry-alias"},
	}
	for _, c := range cases {
		if got := executablePathFor(c.goos, c.in); got != c.want {
			t.Errorf("executablePathFor(%q, %q) = %q, want %q", c.goos, c.in, got, c.want)
		}
	}
}
