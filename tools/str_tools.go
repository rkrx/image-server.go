package tools

import "fmt"
import md5hash "crypto/md5"

func Contains[T string](haystack []T, needle T) bool {
	for _, a := range haystack {
		if a == needle {
			return true
		}
	}
	return false
}

func Md5(input string) string {
	return fmt.Sprintf("%x", md5hash.Sum([]byte(input)))
}
