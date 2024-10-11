package tools

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func ToIntOr(str string, defaultValue int) int {
	if value, err := strconv.Atoi(str); err == nil {
		return value
	}
	return defaultValue
}

func ToFloat32Or(str string, defaultValue float32) float32 {
	if value, err := strconv.ParseFloat(str, 32); err == nil {
		return float32(value)
	}
	return defaultValue
}

func RegexMatch(str, pattern string) (bool, []string) {
	re := regexp.MustCompile(pattern)
	success := re.MatchString(str)
	groups := re.FindAllStringSubmatch(str, -1)
	if len(groups) > 0 {
		return success, groups[0]
	}
	return success, []string{}
}

func RegexReplace(str, pattern, replacement string) string {
	re := regexp.MustCompile(pattern)
	return re.ReplaceAllString(str, replacement)
}

func RegexSplit(str, splitPattern string, n int) []string {
	re := regexp.MustCompile(splitPattern)
	return re.Split(str, n)
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}

func ConcatPathParts(a string, b string) string {
	a = strings.TrimRight(a, "/")
	b = strings.TrimLeft(b, "/")
	return fmt.Sprintf("%s/%s", a, b)
}
