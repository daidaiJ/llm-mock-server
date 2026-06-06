package chat

import "math/rand"

func prompt2Response(prompt string) string {
	runes := []rune(prompt)
	n := len(runes)
	if n == 0 {
		return ""
	}
	length := n / 10
	if length < 1 {
		length = 1
	}
	if length > n {
		length = n
	}
	start := rand.Intn(n - length + 1)
	return string(runes[start : start+length])
}

func ptr[T any](v T) *T {
	return &v
}
