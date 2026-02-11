package writers

func safe(s string) string {
	if s == "" {
		return "(无)"
	}
	return s
}
