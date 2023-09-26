/*
Copyright © 2023 Danny Roes
*/
package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/dannyroes/pinger/data"
	"github.com/dannyroes/pinger/output"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pinger host",
	Args:  cobra.ExactArgs(1),
	Short: "Ping a host a generate a downtime report",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		port, err := cmd.Flags().GetInt("port")
		if err != nil {
			panic(err.Error())
		}

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			state := data.GetState()
			err := output.GeneratePage(w, state)
			if err != nil {
				fmt.Println(err)
			}
		})

		input, err := cmd.Flags().GetString("input")
		if err != nil {
			panic(err.Error())
		}

		if input != "" {
			err = data.InputState(input)
			if err != nil {
				fmt.Printf("Couldn't input state %v\n", err)
			}
		}

		output, err := cmd.Flags().GetString("output")
		if err != nil {
			panic(err.Error())
		}

		if output != "" {
			err = data.OutputState(ctx, output)
			if err != nil {
				fmt.Printf("Couldn't output state %v\n", err)
			}
		}

		fmt.Println("Running monitor")
		data.MonitorUptime(args[0])

		fmt.Println("Listening for requests")
		fmt.Println(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))

		cancel()
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
	rootCmd.Flags().StringP("output", "o", "", "output json file")
	rootCmd.Flags().StringP("input", "i", "", "input json file")
}
