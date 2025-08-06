package encoreapp

import (
    "log"
    "os"
    "time"
    "fmt"
)

// Global logger instance
var logger = log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)

// LogEvent logs a structured event with optional user info
func LogEvent(event string, userID string, details map[string]interface{}) {
    msg := "[" + event + "]"
    if userID != "" {
        msg += " userID=" + userID
    }
    for k, v := range details {
        msg += " " + k + "=" + formatLogValue(v)
    }
    logger.Println(msg)
}

func formatLogValue(v interface{}) string {
    switch t := v.(type) {
    case string:
        return t
    case int, int64, float64:
        return fmt.Sprintf("%v", t)
    case time.Time:
        return t.Format(time.RFC3339)
    default:
        return fmt.Sprintf("%v", t)
    }
}