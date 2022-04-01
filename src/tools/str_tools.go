package tools

import "fmt"
import md5hash "crypto/md5"

func Md5(input string) string {
	return fmt.Sprintf("%x", md5hash.Sum([]byte(input)))
}
