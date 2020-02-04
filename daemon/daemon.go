package daemon

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
)

var (
	log = logrus.StandardLogger()
)

func colorFromMethod(method, format string, a ...interface{}) string {
	switch strings.ToUpper(method) {
	case "GET":
		return fmt.Sprintf(format, a...)
	case "POST":
		return color.CyanString(format, a...)
	case "PUT":
		return color.YellowString(format, a...)
	case "DELETE":
		return color.RedString(format, a...)
	case "PATCH":
		return color.GreenString(format, a...)
	case "HEAD":
		return color.MagentaString(format, a...)
	case "OPTIONS":
		return color.WhiteString(format, a...)
	default:
		return fmt.Sprintf(format, a...)
	}
}

func colorFromStatus(status int, format string, a ...interface{}) string {
	switch {
	case status >= 200 && status < 300:
		return fmt.Sprintf(format, a...)
	case status >= 300 && status < 400:
		return color.CyanString(format, a...)
	case status >= 400 && status < 500:
		return color.YellowString(format, a...)
	default:
		return color.RedString(format, a...)
	}
}
