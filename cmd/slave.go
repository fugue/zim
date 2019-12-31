package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs"

	"github.com/LuminalHQ/zim/slave"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NOTE: NOT USED RIGHT NOW

// NewSlaveCommand returns a slave builder command
func NewSlaveCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "slave",
		Short: "Run a slave that accepts build jobs over a message queue",
		Run: func(cmd *cobra.Command, args []string) {

			opts := getZimOptions()
			fmt.Printf("Zim options: %+v\n", opts)

			timeout := time.Second * time.Duration(viper.GetInt("timeout"))
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			sess, objStore := awsInit(opts)

			res, err := listTaskDefinitions(sess)
			if err != nil {
				fatal(err)
			}
			for _, def := range res {
				fmt.Printf("%+v\n", def)
			}

			builder := slave.New(slave.Opts{
				SQS:   sqs.New(sess),
				Store: objStore,
				// Queue: msgQueue,
				Kind: "",
			})
			fmt.Println("Slave ready")
			fmt.Println("Timeout:", timeout)

			doneChan := make(chan bool)
			if err := builder.Run(ctx, doneChan); err != nil {
				fmt.Println("Slave run error:", err)
			} else {
				fmt.Println("Slave done")
			}
		},
	}

	cmd.Flags().String("name", "", "Name for this execution")
	cmd.Flags().String("source", "", "Source key within S3 bucket")
	cmd.Flags().String("commit", "", "Commit hash")
	cmd.Flags().Int("timeout", 600, "Slave timeout (sec)")

	viper.BindPFlag("name", cmd.Flags().Lookup("name"))
	viper.BindPFlag("source", cmd.Flags().Lookup("source"))
	viper.BindPFlag("commit", cmd.Flags().Lookup("commit"))
	viper.BindPFlag("timeout", cmd.Flags().Lookup("timeout"))

	return cmd
}

// func init() {
// 	rootCmd.AddCommand(NewSlaveCommand())
// }
