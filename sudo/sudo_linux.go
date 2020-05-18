package sudo

import "os/exec"

var queries = []string{
	"gksu",
	"gksudo",
	"kdesu",
	"kdesudo",
}

func WithSudo(args []string) []string {
	var sudo = "sudo"
	var dash bool
	for _, name := range queries {
		if _, err := exec.LookPath(name); err == nil {
			sudo = name
			dash = true
			break
		}
	}
	var ans []string
	ans = append(ans, sudo)
	if dash {
		ans = append(ans, "--")
	}
	ans = append(ans, args...)
	return ans
}
