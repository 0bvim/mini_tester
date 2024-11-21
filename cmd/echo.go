/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// echoCmd represents the echo command
var echoCmd = &cobra.Command{
	Use:   "echo",
	Short: "Run just echo tests",
	Long:  `Run echo with quotes, redirect, pipe and so forth`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("echo called")
	},
}

func init() {
	rootCmd.AddCommand(echoCmd)

	echoCmd.PersistentFlags().String("n", "", "only tests for '-n'")
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// echoCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// echoCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
