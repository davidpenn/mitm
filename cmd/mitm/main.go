package main

import (
	"github.com/davidpenn/mitm/daemon"
	"github.com/spf13/cobra"
)

func main() {
	root := cobra.Command{
		Use: "mitm",
		Run: daemon.StartServer,
	}
	root.Flags().String("listen", ":3128", "address the cache server will listen on")
	root.Flags().String("database", ".cached", "path to the cache database")
	root.Flags().String("database-max-size", "100mb", "max value size a cached object will save")
	root.Flags().Bool("replay", false, "replay requests from cache")
	root.Execute()
}
