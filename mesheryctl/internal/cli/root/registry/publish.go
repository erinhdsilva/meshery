// # Copyright Meshery Authors
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

package registry

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/meshery/schemas/models/v1beta1/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/meshery/meshery/mesheryctl/pkg/utils"
	meshkitRegistryUtils "github.com/meshery/meshkit/registry"
	meshkitUtils "github.com/meshery/meshkit/utils"
)

var (
	system                string
	googleSheetCredential string
	sheetID               string
	modelsOutputPath      string
	imgsOutputPath        string
	models                = []meshkitRegistryUtils.ModelCSV{}
	components            = map[string]map[string][]meshkitRegistryUtils.ComponentCSV{}
	relationships         = []meshkitRegistryUtils.RelationshipCSV{}
	outputFormat          string
)

// Example publishing to meshery docs
// cd docs;
// mesheryctl registry publish website $CRED 1DZHnzxYWOlJ69Oguz4LkRVTFM79kC2tuvdwizOJmeMw docs/pages/integrations docs/assets/img/integrations -o md

// Example publishing to mesheryio docs
// mesheryctl registry publish website $CRED 1DZHnzxYWOlJ69Oguz4LkRVTFM79kC2tuvdwizOJmeMw meshery.io/integrations meshery.io/assets/images/integration -o js

// Example publishing to remove provider docs
// mesheryctl registry publish website $CRED 1DZHnzxYWOlJ69Oguz4LkRVTFM79kC2tuvdwizOJmeMw /src/collections/integrations /src/collections/integrations -o mdx

// publishCmd represents the publish command to publish Meshery Models to Websites, Remote Provider, Meshery
var publishCmd = &cobra.Command{
	Use:   "publish [system] [google-sheet-credential] [sheet-id] [models-output-path] [imgs-output-path]",
	Short: "Publish Meshery Models to Websites, Remote Provider, Meshery Server",
	Long: `Publishes metadata about Meshery Models to Websites, Remote Provider, or Meshery Server, including model and component icons by reading from a Google Spreadsheet and outputing to markdown or json format.
Documentation for components can be found at https://docs.meshery.io/reference/mesheryctl/registry/publish`,
	Example: `
// Publish To System
mesheryctl registry publish [system] [google-sheet-credential] [sheet-id] [models-output-path] [imgs-output-path] -o [output-format]

// Publish To Meshery
mesheryctl registry publish meshery GoogleCredential GoogleSheetID [repo]/server/meshmodel

// Publish To Remote Provider
mesheryctl registry publish remote-provider GoogleCredential GoogleSheetID [repo]/meshmodels/models [repo]/ui/public/img/meshmodels

// Publish To Website
mesheryctl registry publish website GoogleCredential GoogleSheetID [repo]/integrations [repo]/ui/public/img/meshmodels

// Publishing to meshery docs
cd docs;
mesheryctl registry publish website "$CRED" 1DZHnzxYWOlJ69Oguz4LkRVTFM79kC2tuvdwizOJmeMw docs/pages/integrations docs/assets/img/integrations -o md

// Publishing to mesheryio site
mesheryctl registry publish website "$CRED" 1DZHnzxYWOlJ69Oguz4LkRVTFM79kC2tuvdwizOJmeMw meshery.io/integrations meshery.io/assets/images/integration -o js

// Publishing to any website
mesheryctl registry publish website "$CRED" 1DZHnzxYWOlJ69Oguz4LkRVTFM79kC2tuvdwizOJmeMw path/to/models path/to/icons -o mdx
	`,
	PreRunE: func(cmd *cobra.Command, args []string) error {

		if len(args) != 5 {
			return errors.New(utils.RegistryError("[ system, google sheet credential, sheet-id, models output path, imgs output path] are required\n\nUsage: \nmesheryctl registry publish [system] [google-sheet-credential] [sheet-id] [models-output-path] [imgs-output-path]\nmesheryctl registry publish [system] [google-sheet-credential] [sheet-id] [models-output-path] [imgs-output-path] -o [output-format]\nRun 'mesheryctl registry publish --help'", "publish"))
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {

		system = args[0]
		googleSheetCredential = args[1]
		sheetID = args[2]
		modelsOutputPath = args[3]
		imgsOutputPath = args[4]

		srv, err := meshkitUtils.NewSheetSRV(googleSheetCredential)
		if err != nil {
			return errors.New(utils.RegistryError("Invalid JWT Token: Ensure the provided token is a base64-encoded, valid Google Spreadsheets API token.", "publish"))
		}
		resp, err := srv.Spreadsheets.Get(sheetID).Fields().Do()
		if err != nil || resp.HTTPStatusCode != 200 {
			errMsg := fmt.Sprintf("Request to Google Spreadsheet did not succeed.\n\nReturned error: %s", err.Error())
			return errors.New(utils.RegistryError(errMsg, "publish"))
		}

		modelCSVHelper := &meshkitRegistryUtils.ModelCSVHelper{}
		componentCSVHelper := &meshkitRegistryUtils.ComponentCSVHelper{}
		relationshipCSVHelper := &meshkitRegistryUtils.RelationshipCSVHelper{}
		GoogleSpreadSheetURL += sheetID

		for _, v := range resp.Sheets {
			switch v.Properties.Title {
			case "Models":
				modelCSVHelper, err = meshkitRegistryUtils.NewModelCSVHelper(GoogleSpreadSheetURL, v.Properties.Title, v.Properties.SheetId, modelCSVFilePath)
				if err != nil {
					utils.Log.Error(err)
					return nil
				}
				err := modelCSVHelper.ParseModelsSheet(true, modelName)
				if err != nil {
					utils.Log.Error(err)
					return nil
				}
			case "Components":
				componentCSVHelper, err = meshkitRegistryUtils.NewComponentCSVHelper(GoogleSpreadSheetURL, v.Properties.Title, v.Properties.SheetId, componentCSVFilePath)
				if err != nil {
					utils.Log.Error(err)
					return nil
				}
				err := componentCSVHelper.ParseComponentsSheet(modelName)
				if err != nil {
					utils.Log.Error(err)
					return nil
				}
			case "Relationships":
				relationshipCSVHelper, err = meshkitRegistryUtils.NewRelationshipCSVHelper(GoogleSpreadSheetURL, v.Properties.Title, v.Properties.SheetId, relationshipCSVFilePath)
				if err != nil {
					utils.Log.Error(err)
					return nil
				}
				err = relationshipCSVHelper.ParseRelationshipsSheet(modelName)
				if err != nil {
					utils.Log.Error(err)
					return nil
				}
			}
		}

		models = modelCSVHelper.Models
		components = componentCSVHelper.Components
		relationships = relationshipCSVHelper.Relationships

		switch system {
		case "meshery":
			err = mesherySystem()
		case "remote-provider":
			err = remoteProviderSystem()
		case "website":
			if outputFormat != "md" && outputFormat != "mdx" && outputFormat != "js" {
				return errors.New(utils.RegistryError(fmt.Sprintf("invalid output format: %s", outputFormat), "publish"))
			}
			err = websiteSystem()
		default:
			err = fmt.Errorf("invalid system: %s", system) // update to meshkit
		}

		if err != nil {
			utils.Log.Error(err)
			return nil
		}

		err = modelCSVHelper.Cleanup()
		if err != nil {
			utils.Log.Error(err)
			return nil
		}

		err = componentCSVHelper.Cleanup()
		if err != nil {
			utils.Log.Error(err)
			return nil
		}

		return nil
	},
}

// TODO
func mesherySystem() error {
	return nil
}

// Create models definitions to remote provider path
// and add models icons to image output path
func remoteProviderSystem() error {
	// Construct absolute path to store models
	outputPath, _ := filepath.Abs(filepath.Join("../", modelsOutputPath))
	modelDir := filepath.Join(outputPath)
	totalModelsPublished := 0
	for _, model := range models {
		comps, ok := components[model.Registrant][model.Model]
		if !ok {
			utils.Log.Debug("no components found for ", model.Model)
			comps = []meshkitRegistryUtils.ComponentCSV{}
		}

		err := utils.GenerateIcons(model, comps, imgsOutputPath)
		if err != nil {
			utils.Log.Debug(utils.ErrGeneratingIcons(err, imgsOutputPath))
			log.Fatalln(fmt.Printf("Error generating icons for model %s: %v\n", model.Model, err.Error()))
		}

		_, _, err = WriteModelDefToFileSystem(&model, "", modelDir)
		if err != nil {
			return ErrGenerateModel(err, model.Model)
		}
		totalModelsPublished++
	}
	utils.Log.Info("Total model published: ", totalModelsPublished)
	return nil
}

func websiteSystem() error {
	var err error

	relationshipMap := make(map[string][]meshkitRegistryUtils.RelationshipCSV)
	for _, rel := range relationships {
		relationshipMap[rel.Model] = append(relationshipMap[rel.Model], rel)
	}
	docsJSON := "const data = ["
	for _, model := range models {
		comps, ok := components[model.Registrant][model.Model]
		if !ok {
			utils.Log.Debug("no components found for ", model.Model)
			comps = []meshkitRegistryUtils.ComponentCSV{}
		}

		relnships, ok := relationshipMap[model.Model]
		if !ok || len(relnships) == 0 {
			utils.Log.Debug("no relationships found for ", model.Model)
			relnships = []meshkitRegistryUtils.RelationshipCSV{}
		}
		switch outputFormat {
		case "mdx":
			err := utils.GenerateMDXStyleDocs(model, comps, modelsOutputPath, imgsOutputPath) // creates mdx file
			if err != nil {
				log.Fatalln(fmt.Printf("Error generating remote provider docs for model %s: %v\n", model.Model, err.Error()))
			}
		case "md":
			err := utils.GenerateMDStyleDocs(model, comps, relnships, modelsOutputPath, imgsOutputPath) // creates md file
			if err != nil {
				log.Fatalln(fmt.Printf("Error generating meshery docs for model %s: %v\n", model.Model, err.Error()))
			}
		case "js":
			docsJSON, err = utils.GenerateJSStyleDocs(model, docsJSON, comps, relnships, modelsOutputPath, imgsOutputPath) // json file
			if err != nil {
				log.Fatalln(fmt.Printf("Error generating mesheryio docs for model %s: %v\n", model.Model, err.Error()))
			}
		}

	}

	if outputFormat == "js" {
		docsJSON = strings.TrimSuffix(docsJSON, ",")
		docsJSON += "]; export default data"
		mOut, _ := filepath.Abs(filepath.Join(modelsOutputPath, "data.js"))
		if err := meshkitUtils.WriteToFile(mOut, docsJSON); err != nil {
			utils.Log.Error(err)
			return nil
		}
	}

	return nil
}

func init() {
	// these flags are making the command too long. So currently using args instead of flags @theBeginner86

	// publishCmd.Flags().StringVarP(&system, "system", "s", "", "system to publish to")
	// publishCmd.Flags().StringVarP(&googleSheetCredential, "google-sheet-credential", "g", "", "google sheet credential")
	// publishCmd.Flags().StringVarP(&sheetID, "sheet-id", "i", "", "sheet id")
	// publishCmd.Flags().StringVarP(&modelsOutputPath, "models-output-path", "m", "", "models output path")
	// publishCmd.Flags().StringVarP(&imgsOutputPath, "imgs-output-path", "p", "", "images output path")

	publishCmd.Flags().StringVarP(&outputFormat, "output-format", "o", "", "output format [md | mdx | js]")

	// publishCmd.MarkFlagRequired("system")
	// publishCmd.MarkFlagRequired("google-sheet-credential")
	// publishCmd.MarkFlagRequired("sheet-id")
	// publishCmd.MarkFlagRequired("models-output-path")
	// publishCmd.MarkFlagRequired("imgs-output-path")
}

func WriteModelDefToFileSystem(model *meshkitRegistryUtils.ModelCSV, version string, location string) (string, *model.ModelDefinition, error) {
	modelDef := model.CreateModelDefinition(version, defVersion)
	modelDefPath := filepath.Join(location, modelDef.Name)
	err := modelDef.WriteModelDefinition(modelDefPath+"/model.json", "json")
	if err != nil {
		return "", nil, err
	}

	return modelDefPath, &modelDef, nil
}
