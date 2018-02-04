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
	"fmt"

	"github.com/spf13/cobra"
	"github.com/renstrom/dedent"
	"github.com/wso2/product-apim-tooling/import-export-cli/utils"
	"net/http"
	"crypto/tls"
	"time"
	"strings"
	"path/filepath"
	"bytes"
	"mime/multipart"
	"io"
	"os"
)

var importAppFile string
var importAppEnvironment string
var importAppCmdUsername string
var importAppCmdPassword string
var preserveOwner bool
var addSubscriptions bool

// ImportApp command related usage info
const importAppCmdLiteral = "import-app"
const importAppCmdShortDesc = "Import App"

var importAppCmdLongDesc = "Import an Application to an environment"

var importAppCmdExamples = dedent.Dedent(`
		Examples:
		` + utils.ProjectName + ` ` + importAppCmdLiteral + ` -f qa/sampleApp.zip -e dev
		` + utils.ProjectName + ` ` + importAppCmdShortDesc + ` -f staging/sampleApp.zip -e prod -u admin -p admin
		` + utils.ProjectName + ` ` + importAppCmdLiteral + ` -f qa/sampleApp.zip --preserveOwner --addSubscriptions -e prod
	`)

// importAppCmd represents the importApp command
var ImportAppCmd = &cobra.Command{
	Use: importAppCmdLiteral + " (--file <app-zip-file> --environment " +
		"<environment-to-which-the-app-should-be-imported>)",
	Short: importAppCmdShortDesc,
	Long:  importAppCmdLongDesc + importAppCmdExamples,
	Run: func(cmd *cobra.Command, args []string) {
		utils.Logln(utils.LogPrefixInfo + importAppCmdLiteral + " called")
		executeImportAppCmd(utils.MainConfigFilePath, utils.EnvKeysAllFilePath, utils.ExportDirectory)
	},
}

func executeImportAppCmd(mainConfigFilePath, envKeysAllFilePath, exportDirectory string) {
	accessToken, preCommandErr :=
		utils.ExecutePreCommandWithOAuth(importAppEnvironment, importAppCmdUsername, importAppCmdPassword,
			mainConfigFilePath, envKeysAllFilePath)

	if preCommandErr == nil {
		storeEndpiont := utils.GetStoreEndpointOfEnv(importAppEnvironment, mainConfigFilePath)
		resp, err := ImportApplication(importAppFile, storeEndpiont, accessToken, exportDirectory)
		if err != nil {
			utils.HandleErrorAndExit("Error importing Application", err)
		}

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
			// 200 OK or 201 Created
			utils.Logln(utils.LogPrefixInfo+"Header:", resp.Header)
			fmt.Println("Succesfully imported Application!")
		} else {
			fmt.Println("Error importing Application")
			utils.Logln(utils.LogPrefixError + resp.Status)
		}
	} else {
		// env_endpoints file is not configured properly by the user
		fmt.Println("Error:", preCommandErr)
		utils.Logln(utils.LogPrefixError + preCommandErr.Error())
	}
}

// ImportApplication function is used with import-app command
// @param name: name of the Application (zipped file) to be imported
// @param apiManagerEndpoint: API Manager endpoint for the environment
// @param accessToken: OAuth2.0 access token for the resource being accessed
func ImportApplication(query, storeEndpiont, accessToken, exportDirectory string) (*http.Response, error) {
	storeEndpiont = utils.AppendSlashToString(storeEndpiont)

	applicationImportEndpoint := storeEndpiont + "import/applications"
	//url := applicationImportEndpoint+"addSubscriptions"
	utils.Logln(utils.LogPrefixInfo + "Import URL: " + applicationImportEndpoint)

	sourceEnv := strings.Split(query, "/")[0] // environment from which the Application was exported
	utils.Logln(utils.LogPrefixInfo + "Source Environment: " + sourceEnv)

	fileName := query // ex:- fileName = dev/sampleApp.zip //TODO change the ex here appropriately

	zipFilePath := filepath.Join(exportDirectory, fileName)
	fmt.Println("ZipFilePath:", zipFilePath)

	// check if '.zip' exists in the input 'fileName'
	//hasZipExtension, _ := regexp.MatchString(`^\S+\.zip$`, fileName)

	//if hasZipExtension {
	//	// import the zip file directly
	//	//fmt.Println("hasZipExtension: ", true)
	//
	//} else {
	//	//fmt.Println("hasZipExtension: ", false)
	//	// search for a directory with the given fileName
	//	destination := filepath.Join(exportDirectory, fileName+".zip")
	//	err := utils.ZipDir(zipFilePath, destination)
	//	if err != nil {
	//		utils.HandleErrorAndExit("Error creating zip archive", err)
	//	}
	//	zipFilePath += ".zip"
	//}

	extraParams := map[string]string{}
	// TODO:: Add extraParams as necessary

	req, err := NewAppFileUploadRequest(applicationImportEndpoint, extraParams, "file", zipFilePath, accessToken)
	if err != nil {
		utils.HandleErrorAndExit("Error creating request.", err)
	}

	var tr *http.Transport
	if utils.Insecure {
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	} else {
		tr = &http.Transport{}
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   time.Duration(utils.HttpRequestTimeout) * time.Second,
	}

	resp, err := client.Do(req)

	if err != nil {
		utils.Logln(utils.LogPrefixError, err)
	} else {
		//var bodyContent []byte

		if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
			// 201 Created or 200 OK
			fmt.Println("Successfully imported Appplication '" + fileName + "'")
		} else {
			fmt.Println("Error importing Application.")
			fmt.Println("Status: " + resp.Status)
		}

		//fmt.Println(resp.Header)
		//resp.Body.Read(bodyContent)
		//resp.Body.Close()
		//fmt.Println(bodyContent)
	}

	return resp, err
}

// NewFileUploadRequest form an HTTP Put request
// Helper function for forming multi-part form data
// Returns the formed http request and errors
func NewAppFileUploadRequest(uri string, params map[string]string, paramName, path,
accessToken string) (*http.Request, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(paramName, filepath.Base(path))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest(http.MethodPost, uri, body)
	request.Header.Add(utils.HeaderAuthorization, utils.HeaderValueAuthBearerPrefix+" "+accessToken)
	request.Header.Add(utils.HeaderContentType, writer.FormDataContentType())
	request.Header.Add(utils.HeaderAccept, "*/*")
	request.Header.Add(utils.HeaderConnection, utils.HeaderValueKeepAlive)

	return request, err
}

func init() {
	RootCmd.AddCommand(ImportAppCmd)
	ImportAppCmd.Flags().StringVarP(&importAppFile, "file", "f", "",
		"Name of the Application to be imported")
	ImportAppCmd.Flags().StringVarP(&importAppEnvironment, "environment", "e",
		utils.DefaultEnvironmentName, "Environment from the which the Application should be imported")
	ImportAppCmd.Flags().BoolVarP(&preserveOwner, "perserveOwner", "o", false,
		"Preserves app owner")
	ImportAppCmd.Flags().BoolVarP(&addSubscriptions, "addSubscriptions", "s", false,
		"Adds subscriptions of the Application" )
	ImportAppCmd.Flags().StringVarP(&importAppCmdUsername, "username", "u", "", "Username")
	ImportAppCmd.Flags().StringVarP(&importAppCmdPassword, "password", "p", "", "Password")
}

