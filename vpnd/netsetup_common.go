package vpnd

import (
	"fmt"
	"strconv"
	"strings"
)

func itoa(n int) string {
	return strconv.Itoa(n)
}

func wrapCmdError(name string, args []string, err error, out []byte) error {
	return fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
}
