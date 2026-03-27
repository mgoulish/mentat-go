package debug

import "fmt"

var (
    showInfo  bool
    showDebug bool
)

func SetInfo(b bool)  { showInfo = b }
func SetDebug(b bool) { showDebug = b }

func Info(s string) {
    if showInfo {
        fmt.Printf("mentat info: %s\n", s)
    }
}

func Debug(s string) {
    if showDebug {
        fmt.Printf("mentat debug: %s\n", s)
    }
}
