/*
Copyright © 2023 Danny Roes
*/
package cmd

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dannyroes/pinger/data"
	"github.com/dannyroes/pinger/output"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pinger host",
	Short: "Ping a host a generate a downtime report",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		port, err := cmd.Flags().GetInt("port")
		if err != nil {
			panic(err.Error())
		}

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			state := []data.Status{{
				Start: time.Now().Add(-2 * time.Hour),
				// End:   time.Now().Add(-5 * time.Minute),
				State: data.StatusUp,
			},
				{
					Start: time.Now().Add(-3 * time.Hour),
					End:   time.Now().Add(-2 * time.Hour),
					State: data.StatusDown,
				},
			}

			err := output.GeneratePage(w, state)
			if err != nil {
				fmt.Println(err)
			}
		})

		fmt.Println("Listening for requests")
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().IntP("port", "p", 8080, "local port to listen for web requests")
}
