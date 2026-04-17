package utils

import (
    "fmt"
    "time"
)

func StringToMicrosecondsSinceEpoch(s string) (int64, error) {
    formats := []string{
        "2006-01-02 15:04:05.999999999 -0700",
        "2006-01-02 15:04:05.999999999 +0000",
        "2006-01-02 15:04:05.999999999",
        time.RFC3339Nano,
    }

    for _, f := range formats {
        if t, err := time.Parse(f, s); err == nil {
            return t.UnixMicro(), nil
        }
    }
    return 0, fmt.Errorf("could not parse timestamp: %s", s)
}

