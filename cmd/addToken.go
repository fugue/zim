// Copyright 2020 Fugue, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/fugue/zim/project"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewAddTokenCommand returns a command that adds a new cache token
func NewAddTokenCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "token",
		Short: "Add a cache token",
		Run: func(cmd *cobra.Command, args []string) {

			opts, err := getZimOptions(cmd, args)
			if err != nil {
				fatal(err)
			}

			name := viper.GetString("name")
			email := viper.GetString("email")

			if name == "" || !strings.Contains(email, "@") {
				fatal(errors.New("Must specify name and email"))
			}

			sess, err := getSession(opts.Region)
			if err != nil {
				fatal(err)
			}

			svc := dynamodb.New(sess)

			authToken := project.UUID()

			values := map[string]string{
				"Token": authToken,
				"Name":  name,
				"Email": email,
			}
			item, err := dynamodbattribute.MarshalMap(values)
			if err != nil {
				fatal(err)
			}

			_, err = svc.PutItem(&dynamodb.PutItemInput{
				Item:      item,
				TableName: aws.String("AuthTokens"),
			})
			if err != nil {
				fatal(err)
			}

			fmt.Println(authToken)
		},
	}

	cmd.Flags().String("name", "", "Username")
	cmd.Flags().String("email", "", "Email")
	viper.BindPFlag("name", cmd.Flags().Lookup("name"))
	viper.BindPFlag("email", cmd.Flags().Lookup("email"))

	return cmd
}

func init() {
	addCmd.AddCommand(NewAddTokenCommand())
}
