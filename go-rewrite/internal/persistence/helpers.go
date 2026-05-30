package persistence

import "time"

func timeNow() string {
	return time.Now().Format(time.RFC3339)
}
