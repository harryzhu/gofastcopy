package cmd

import (
	"errors"
	"fmt"
	"log"
	"strings"
)

func FatalError(prefix string, err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func NewError(args ...any) error {
	var s []string
	for _, arg := range args {
		s = append(s, fmt.Sprintf("%v", arg))
	}
	return errors.New(strings.Join(s, ""))
}

func DebugInfo(prefix string, args ...any) {
	if IsDebug {
		var info []string
		for _, arg := range args {
			info = append(info, fmt.Sprintf("%v", arg))
		}
		log.Printf("DEBUG: %v: %v\n", prefix, strings.Join(info, ""))
	}
}

func DebugWarn(prefix string, args ...any) {
	if IsDebug {
		var info []string
		for _, arg := range args {
			info = append(info, fmt.Sprintf("%v", arg))
		}
		log.Println(Yellow("WARN:"), Yellow(prefix+":"), Yellow(strings.Join(info, "")))
	}
}

func PrintError(prefix string, err error) {
	if err != nil {
		log.Println(Red("ERROR:"), Red(prefix), err)
	}
}

func PrintlnInfo(color, prefix string, args ...any) {
	var info []string
	for _, arg := range args {
		info = append(info, fmt.Sprintf("%v", arg))
	}
	line := strings.Join(info, "")

	switch strings.ToLower(color) {
	case "green":
		log.Printf("INFO: %v: %v\n", Green(prefix), line)
	case "black":
		log.Printf("INFO: %v: %v\n", Black(prefix), line)
	case "red":
		log.Printf("INFO: %v: %v\n", Red(prefix), line)
	case "yellow":
		log.Printf("INFO: %v: %v\n", Yellow(prefix), line)
	case "blue":
		log.Printf("INFO: %v: %v\n", Blue(prefix), line)
	case "purple":
		log.Printf("INFO: %v: %v\n", Purple(prefix), line)
	case "cyan":
		log.Printf("INFO: %v: %v\n", Cyan(prefix), line)
	case "white":
		log.Printf("INFO: %v: %v\n", White(prefix), line)
	default:
		log.Printf("INFO: %v: %v\n", prefix, line)
	}
}

// -----color----
const (
	textBlack = iota + 30
	textRed
	textGreen
	textYellow
	textBlue
	textPurple
	textCyan
	textWhite
)

func Black(str string) string {
	return textColor(textBlack, str)
}

func Red(str string) string {
	return textColor(textRed, str)
}
func Yellow(str string) string {
	return textColor(textYellow, str)
}
func Green(str string) string {
	return textColor(textGreen, str)
}
func Cyan(str string) string {
	return textColor(textCyan, str)
}
func Blue(str string) string {
	return textColor(textBlue, str)
}
func Purple(str string) string {
	return textColor(textPurple, str)
}
func White(str string) string {
	return textColor(textWhite, str)
}

func textColor(color int, str string) string {
	return fmt.Sprintf("\x1b[0;%dm%s\x1b[0m", color, str)
}

func PrintSpinner(s string) {
	fmt.Printf(" ... %20.10s\r", s)
}

func PrintSpinner2(s string, suffix string) {
	fmt.Printf(" ... %5.10s    %s\r", s, suffix)
}
