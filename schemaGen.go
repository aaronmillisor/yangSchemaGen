package main

import (
	"fmt"

	"log"

	yang "github.com/openconfig/goyang/pkg/yang"
)

// struct templates to marshal a json schema file for yaml
type Schema struct {
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Properties  map[string]interface{} `json:"properties"`
}

type Property struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Format      string                 `json:"format,omitempty"`
	Enum        []interface{}          `json:"enum,omitempty"`
	Items       *Property              `json:"items,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
}

type EnumProperty struct {
	Type        string        `json:"type"`
	Description string        `json:"description"`
	Enum        []interface{} `json:"enum"`
}

type ArrayProperty struct {
	Type  string    `json:"type"`
	Items *Property `json:"items"`
}

type ObjectProperty struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
}

type StringProperty struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Format      string `json:"format,omitempty"`
}

type NumberProperty struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Format      string `json:"format,omitempty"`
}

type IntegerProperty struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Format      string `json:"format,omitempty"`
}

type BooleanProperty struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

func printModules(path string, entries map[string]*yang.Entry) {

	for k, v := range entries {
		fmt.Println(path+k, " ", v.Kind, ", key: ", v.Key)
		if v.Type != nil {
			fmt.Println(v.Type.Name)
			if v.Type.Name == "leafref" {
				fmt.Println(v.Type.Path)
				/*
						relativePath := strings.Split(v.Type.Path())
						fmt.Println(v.Type, relativePath)
						entry := v.Parent
						for _, elem := range relativePath[1:] {
							fmt.Println("the elem: ", elem)
							entry = entry.Dir[elem]
						}
					fmt.Println("leafref type: ", entry.Type)
				*/
			}
			if v.Type.Name == "enumeration" {
				fmt.Println("Enum: ", v.Type.Enum)
			}
			if v.Type.Name == "identityref" {
				for _, v := range v.Type.IdentityBase.Values {
					fmt.Println(v.Name, " ", v.ParentNode().NName())
				}
				//	fmt.Println("Identity: ", v.Type.IdentityBase.Values)
			}
		}
		fmt.Println("--")
		printModules(path+k+"/", v.Dir)
	}
}

// function that reads in set of yang files using goyang
func readYangFiles(yangDir string) {

	filenames := make(map[string]bool)
	filenames["yang/openconfig-interfaces.yang"] = true
	filenames["yang/openconfig-network-instance.yang"] = true
	filenames["yang/openconfig-if-ethernet.yang"] = true
	filenames["yang/openconfig-if-ip.yang"] = true

	// Get a list of all yang files in the directory
	/*files, err := ioutil.ReadDir(yangDir)
	if err != nil {
		fmt.Println("Error reading yang files:", err)
		return
	}
	*/

	modules := yang.NewModules()

	// Loop through each yang file and process it
	for filename, _ := range filenames {
		// Check if the file is a yang file
		/*if strings.HasSuffix(file.Name(), ".yang") {

		// Read the contents of the yang file
		filePath := filepath.Join(yangDir, file.Name())
		*/

		err := modules.Read(filename)

		if err != nil {
			log.Fatalln("Error loading yang module:", err)

		}

		fmt.Println("Processinhg yang file:", filename)
	}

	errs := modules.Process()
	if errs != nil {
		fmt.Println(errs)
	}

	// get yang entry from module
	entry, err := modules.GetModule("openconfig-network-instance")
	if err != nil {
		log.Fatal("Problem finding module ", err)
	}

	printModules("/", entry.Dir)

	//fmt.Println("Successfully processed yang file:", filename)

	//}
}

// initial commit
func main() {

	readYangFiles("./yang")

	fmt.Println("Hello world")
}
