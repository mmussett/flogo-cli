package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/TIBCOSoftware/flogo/cli"
	"github.com/TIBCOSoftware/flogo/util"
)

var optAdd = &cli.OptionInfo{
	Name:      "add",
	UsageLine: "add <activity|model|trigger> <path>",
	Short:     "add an activity, model or trigger to a flogo project",
	Long: `Add an activity, model or trigger to a flogo project

Options:
    -src   copy contents to source (only when using local/file)
`,
}

var validItemTypes = []string{itActivity, itTrigger, itModel}

func init() {
	commandRegistry.RegisterCommand(&cmdAdd{option: optAdd})
}

type cmdAdd struct {
	option   *cli.OptionInfo
	addToSrc bool
}

func (c *cmdAdd) OptionInfo() *cli.OptionInfo {
	return c.option
}

func (c *cmdAdd) AddFlags(fs *flag.FlagSet) {
	fs.BoolVar(&(c.addToSrc), "src", false, "copy contents to source (only when using local/file)")
}

func (c *cmdAdd) Exec(args []string) error {

	projectDescriptor := loadProjectDescriptor()

	if len(args) == 0 {
		fmt.Fprint(os.Stderr, "Error: item type not specified\n\n")
		cmdUsage(c)
	}

	itemType := strings.ToLower(args[0])

	if !fgutil.IsStringInList(itemType, validItemTypes) {
		fmt.Fprintf(os.Stderr, "Error: invalid item type '%s'\n\n", itemType)
		cmdUsage(c)
	}

	if len(args) == 1 {
		fmt.Fprintf(os.Stderr, "Error: %s path not specified\n\n", fgutil.Capitalize(itemType))
		cmdUsage(c)
	}

	itemPath := args[1]

	if len(args) > 2 {
		fmt.Fprintf(os.Stderr, "Error: Too many arguments given\n\n")
		cmdUsage(c)
	}

	installItem(projectDescriptor, itemType, itemPath, c.addToSrc)

	return nil
}

func installItem(projectDescriptor *FlogoProjectDescriptor, itemType string, itemPath string, addToSrc bool) {

	gb := fgutil.NewGb(projectDescriptor.Name)

	var itemConfigPath string
	var itemConfig *ItemDescriptor

	switch itemType {
	case itActivity:
		itemConfig, itemConfigPath = AddFlogoItem(gb, itActivity, itemPath, projectDescriptor.Activities, addToSrc)
		projectDescriptor.Activities = append(projectDescriptor.Activities, itemConfig)
	case itModel:
		itemConfig, itemConfigPath = AddFlogoItem(gb, itModel, itemPath, projectDescriptor.Models, addToSrc)
		projectDescriptor.Models = append(projectDescriptor.Models, itemConfig)
	case itTrigger:
		itemConfig, itemConfigPath = AddFlogoItem(gb, itTrigger, itemPath, projectDescriptor.Triggers, addToSrc)
		projectDescriptor.Triggers = append(projectDescriptor.Triggers, itemConfig)

		//read trigger.json
		triggerConfigFile, err := os.Open(itemConfigPath)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Unable to find '%s'\n\n", itemConfigPath)
			os.Exit(2)
		}

		triggerProjectDescriptor := &TriggerProjectDescriptor{}
		jsonParser := json.NewDecoder(triggerConfigFile)

		if err = jsonParser.Decode(triggerProjectDescriptor); err != nil {
			fmt.Fprint(os.Stderr, "Error: Unable to parse trigger.json, file may be corrupted.\n\n")
			os.Exit(2)
		}

		triggerConfigFile.Close()

		//read engine config.json
		engineConfigPath := gb.NewBinFilePath(fileEngineConfig)
		engineConfigFile, err := os.Open(engineConfigPath)

		engineConfig := &EngineConfig{}
		jsonParser = json.NewDecoder(engineConfigFile)

		if err = jsonParser.Decode(engineConfig); err != nil {
			fmt.Fprint(os.Stderr, "Error: Unable to parse application config.json, file may be corrupted.\n\n")
			os.Exit(2)
		}

		engineConfigFile.Close()

		if engineConfig.Triggers == nil {
			engineConfig.Triggers = make([]*TriggerConfig, 0)
		}

		if !ContainsTriggerConfig(engineConfig.Triggers, itemConfig.Name) {

			triggerConfig := &TriggerConfig{Name: itemConfig.Name, Settings: make(map[string]string)}

			for _, v := range triggerProjectDescriptor.Settings {

				triggerConfig.Settings[v.Name] = v.Value
			}

			engineConfig.Triggers = append(engineConfig.Triggers, triggerConfig)

			fgutil.WriteJSONtoFile(engineConfigPath, engineConfig)
		}

	// add config
	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown item type '%s'\n\n", itemType)
		os.Exit(2)
	}

	updateProjectDescriptorFiles(gb, projectDescriptor)
}

// ContainsTriggerConfig determines if the list of TriggerConfigs contains the specified one
func ContainsTriggerConfig(list []*TriggerConfig, triggerName string) bool {
	for _, v := range list {
		if v.Name == triggerName {
			return true
		}
	}
	return false
}
