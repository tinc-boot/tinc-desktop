package sudo

import (
	"strconv"
	"strings"
)

func WithSudo(args []string) []string {
	var escaped []string
	for _, arg := range args {
		escaped = append(escaped, strconv.Quote(arg))
	}

	cli := "do shell script " + strconv.Quote(strings.Join(escaped, " ")) + " with administrator privileges"

	return []string{"osascript", "-e", cli}
}
