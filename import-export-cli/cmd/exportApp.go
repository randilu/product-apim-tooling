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
	"github.com/wso2/product-apim-tooling/import-export-cli/utils"
	"github.com/renstrom/dedent"
	"net/http"
	"fmt"
	"github.com/go-resty/resty"
	"path/filepath"
	"os"
	"io/ioutil"
)

var exportAppName string
var exportAppCmdUsername string
var exportAppCmdPassword string
//var flagExportAPICmdToken string
// ExportApp command related usage info
const exportAppCmdLiteral = "export-app"
const exportAppCmdShortDesc = "Export App"

var exportAppCmdLongDesc = "Export an Application from an environment"

var exportAppCmdExamples = dedent.Dedent(`
		Examples:
		` + utils.ProjectName + ` ` + exportAppCmdLiteral + ` -n SampleApp 9f6affe2-4c97-4817-bded-717f8b01eee8 -e dev
		` + utils.ProjectName + ` ` + exportAppCmdLiteral + ` -n SampleApp 7bc2b94e-c6d2-4d4f-beb1-cdccb08cd87f -e prod
		NOTE: Flag --uuid (-i) is mandatory
	`)
// exportAppCmd represents the exportApp command
var ExportAppCmd = &cobra.Command{
	Use: exportAppCmdLiteral + " (--uuid <uuid-of-the-application> --environment " +
		"<environment-from-which-the-app-should-be-exported>)",
	Short: exportAppCmdShortDesc,
	Long:  exportAppCmdLongDesc + exportAppCmdExamples,
	Run: func(cmd *cobra.Command, args []string) {
		utils.Logln(utils.LogPrefixInfo + exportAppCmdLiteral + " called")
		executeExportAppCmd(utils.MainConfigFilePath, utils.EnvKeysAllFilePath, utils.ExportDirectory)
	},
}

func executeExportAppCmd(mainConfigFilePath, envKeysAllFilePath, exportDirectory string) {
	accessToken, preCommandErr :=
		utils.ExecutePreCommandWithOAuth(exportEnvironment, exportAppCmdUsername, exportAppCmdPassword,
			mainConfigFilePath, envKeysAllFilePath)

	if preCommandErr == nil {
		storeEndpiont := utils.GetStoreEndpointOfEnv(exportEnvironment, mainConfigFilePath)
		resp := getExportAppResponse(exportAppName, storeEndpiont, accessToken)

		// Print info on response
		utils.Logf(utils.LogPrefixInfo+"ResponseStatus: %v\n", resp.Status())

		if resp.StatusCode() == http.StatusOK {
			WriteApplicationToZip(exportAppName, exportAppCmdUsername, exportEnvironment, exportDirectory, resp)
		} else if resp.StatusCode() == http.StatusInternalServerError {
			// 500 Internal Server Error
			fmt.Println("Incorrect password")
		} else {
			// neither 200 nor 500
			fmt.Println("Error exporting Application:", resp.Status())
		}
	} else {
		// error exporting Application
		fmt.Println("Error exporting Application:" + preCommandErr.Error())
	}
}

// WriteApplicationToZip
// @param exportAppName : Name of the Application to be exported
// @param resp : Response returned from making the HTTP request (only pass a 200 OK)
// Exported Application will be written to a zip file
func WriteApplicationToZip(exportAppName, exportAppCmdUsername, exportEnvironment, exportDirectory string,
	resp *resty.Response) {
	// Write to file
	directory := filepath.Join(exportDirectory, exportEnvironment)
	// create directory if it doesn't exist
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		os.Mkdir(directory, 0777)
		// permission 777 : Everyone can read, write, and execute
	}
	zipFilename := exportAppCmdUsername + "_" + exportAppName + ".zip" // admin_testApp.zip
	pFile := filepath.Join(directory, zipFilename)
	err := ioutil.WriteFile(pFile, resp.Body(), 0644)
	// permission 644 : Only the owner can read and write.. Everyone else can only read.
	if err != nil {
		utils.HandleErrorAndExit("Error creating zip archive", err)
	}
	fmt.Println("Succesfully exported Application!")
	fmt.Println("Find the exported Application at " + pFile)
}

// ExportApp
// @param name : Name of the Application to be exported
// @param apimEndpoint : API Manager Endpoint for the environment
// @param accessToken : Access Token for the resource
// @return response Response in the form of *resty.Response
func getExportAppResponse(name, storeEndpoint, accessToken string) *resty.Response {
	storeEndpoint = utils.AppendSlashToString(storeEndpoint)
	query := "export/applications?appId=" + name //TODO change the appId to appName

	url := storeEndpoint + query
	utils.Logln(utils.LogPrefixInfo+"ExportApp: URL:", url)
	headers := make(map[string]string)
	headers[utils.HeaderAuthorization] = utils.HeaderValueAuthBearerPrefix + " " + accessToken
	headers[utils.HeaderAccept] = utils.HeaderValueApplicationZip

	resp, err := utils.InvokeGETRequest(url, headers)

	if err != nil {
		utils.HandleErrorAndExit("Error exporting Application: "+name, err)
	}

	return resp
}

//init using Cobra
func init() {
	RootCmd.AddCommand(ExportAppCmd)
	ExportAppCmd.Flags().StringVarP(&exportAppName, "name", "n", "",
		"Name of the Application to be exported")
	ExportAppCmd.Flags().StringVarP(&exportEnvironment, "environment", "e",
		utils.DefaultEnvironmentName, "Environment to which the Application should be exported")

	ExportAppCmd.Flags().StringVarP(&exportAppCmdUsername, "username", "u", "", "Username")
	ExportAppCmd.Flags().StringVarP(&exportAppCmdPassword, "password", "p", "", "Password")
}
