/*
*  Copyright (c) WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
*
*  WSO2 Inc. licenses this file to you under the Apache License,
*  Version 2.0 (the "License"); you may not use this file except
*  in compliance with the License.
*  You may obtain a copy of the License at
*
*    http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing,
* software distributed under the License is distributed on an
* "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
* KIND, either express or implied.  See the License for the
* specific language governing permissions and limitations
* under the License.
 */

package cmd

import (
	"github.com/spf13/cobra"
	"github.com/renstrom/dedent"
	"github.com/wso2/product-apim-tooling/import-export-cli/utils"
	"fmt"
	"net/http"
	"encoding/json"
	"errors"
	"github.com/olekukonko/tablewriter"
	"os"
)

var listAppsCmdEnvironment string
var listAppsCmdUsername string
var listAppsCmdPassword string

// appsCmd related info
const appsCmdLiteral = "apps"
const appsCmdShortDesc = "Display a list of Applications in an environment specific to the user"

var appsCmdLongDesc = dedent.Dedent(`
		Display a list of Applications of the user in the environment specified by the flag --environment, -e
	`)

var appsCmdExamples = dedent.Dedent(`
	` + utils.ProjectName + ` ` + listCmdLiteral + ` ` + appsCmdLiteral + ` -e dev
	` + utils.ProjectName + ` ` + listCmdLiteral + ` ` + appsCmdLiteral + ` -e dev -q version:1.0.0
	` + utils.ProjectName + ` ` + listCmdLiteral + ` ` + appsCmdLiteral + ` -e prod -q provider:admin
	` + utils.ProjectName + ` ` + listCmdLiteral + ` ` + appsCmdLiteral + ` -e staging -u admin -p admin
	`)

// appsCmd represents the apps command
var appsCmd = &cobra.Command{
	Use:   appsCmdLiteral,
	Short: appsCmdShortDesc,
	Long:  appsCmdLongDesc + appsCmdExamples,
	Run: func(cmd *cobra.Command, args []string) {
		utils.Logln(utils.LogPrefixInfo + appsCmdLiteral + " called")
		executeAppsCmd(utils.MainConfigFilePath, utils.EnvKeysAllFilePath)
	},
}

func executeAppsCmd(mainConfigFilePath, envKeysAllFilePath string) {
	accessToken, preCommandErr :=
		utils.ExecutePreCommandWithOAuth(listAppsCmdEnvironment, listAppsCmdUsername, listAppsCmdPassword,
			mainConfigFilePath, envKeysAllFilePath)

	if preCommandErr == nil {
		applicationListEndpoint := utils.GetApplicationListEndpointOfEnv(listAppsCmdEnvironment, mainConfigFilePath)
		count, apps, err := GetApplicationList(accessToken, applicationListEndpoint)

		if err == nil {
			// Printing the list of available Applications
			fmt.Println("Environment:", listAppsCmdEnvironment)
			fmt.Println("No. of Applications:", count)
			if count > 0 {
				printApps(apps)
			}
		} else {
			utils.Logln(utils.LogPrefixError+"Getting List of Applications", err)
		}

	} else {
		utils.Logln(utils.LogPrefixError + "calling 'list' " + preCommandErr.Error())
		utils.HandleErrorAndExit("Error calling '"+appsCmdLiteral+"'", preCommandErr)
	}

}

//Get Application List
// @param accessToken : Access Token for the environment
// @param apiManagerEndpoint : API Manager Endpoint for the environment
// @return count (no. of Applications)
// @return array of Application objects
// @return error

func GetApplicationList(accessToken, applicationListEndpoint string) (count int32, apps []utils.Application,
	err error) {

	headers := make(map[string]string)
	headers[utils.HeaderAuthorization] = utils.HeaderValueAuthBearerPrefix + " " + accessToken

	resp, err := utils.InvokeGETRequest(applicationListEndpoint, headers)

	if err != nil {
		utils.HandleErrorAndExit("Unable to connect to "+applicationListEndpoint, err)
	}

	utils.Logln(utils.LogPrefixInfo+"Response:", resp.Status())

	if resp.StatusCode() == http.StatusOK {
		appListResponse := &utils.ApplicationListResponse{}
		unmarshalError := json.Unmarshal([]byte(resp.Body()), &appListResponse)

		if unmarshalError != nil {
			utils.HandleErrorAndExit(utils.LogPrefixError+"invalid JDON response", unmarshalError)
		}

		return appListResponse.Count, appListResponse.List, nil

	} else {
		return 0, nil, errors.New(resp.Status())
	}
}

func printApps (apps []utils.Application){
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Name", "Subscriber", "Tier", "Status"})

	var data [][]string

	for _, app := range apps {
		data = append (data, []string{app.ID, app.Name, app.Subscriber, app.Tier, app.Status})
	}

	for _, v := range data {
		table.Append(v)
	}

	table.Render()
}

func init() {
	ListCmd.AddCommand(appsCmd)

	appsCmd.Flags().StringVarP(&listAppsCmdEnvironment, "environment", "e",
		utils.DefaultEnvironmentName, "Environment to be searched")
	appsCmd.Flags().StringVarP(&listAppsCmdUsername, "username", "u", "", "Username")
	appsCmd.Flags().StringVarP(&listAppsCmdPassword, "password", "p", "", "Password")
}
