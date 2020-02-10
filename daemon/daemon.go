package daemon

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/fatih/color"
	"github.com/kr/mitm"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	log = logrus.StandardLogger()
)

// StartServer and listen for requests
func StartServer(cmd *cobra.Command, args []string) {
	path, _ := cmd.Flags().GetString("database")
	maxValueSize, _ := cmd.Flags().GetString("database-max-size")
	replay, _ := cmd.Flags().GetBool("replay")
	server := NewServer(path, maxValueSize, replay)

	ca, err := genCA()
	if err != nil {
		log.Fatal(err)
	}

	proxy := &mitm.Proxy{
		CA:   &ca,
		Wrap: server.Handler,
	}

	addr, _ := cmd.Flags().GetString("listen")
	http.ListenAndServe(addr, httpHandler(server, proxy))
}

func httpHandler(api, proxy http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Hostname() == "" {
			api.ServeHTTP(w, r)
		} else {
			proxy.ServeHTTP(w, r)
		}
	})
}

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
