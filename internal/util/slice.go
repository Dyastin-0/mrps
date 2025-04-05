package util

func ReverseSlice(slice []string) []string {
	reversed := make([]string, len(slice))
	for i, v := range slice {
		reversed[len(slice)-1-i] = v
	}
	return reversed
}
