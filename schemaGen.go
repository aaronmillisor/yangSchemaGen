package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	yang "github.com/openconfig/goyang/pkg/yang"
)

// struct templates to marshal a json schema file for yaml
type Schema struct {
	Id          string                 `json:"$id"`
	Schema      string                 `json:"$schema"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Properties  map[string]interface{} `json:"properties"`
}

//json schema struct for properties

type Property struct {
	Type        string                 `json:"type,omitempty"`
	Description string                 `json:"description"`
	Format      string                 `json:"format,omitempty"` // may be used later
	Enum        interface{}            `json:"enum,omitempty"`
	Items       *Property              `json:"items,omitempty"` // used for paths with keys and leaf lists
	Properties  map[string]interface{} `json:"properties,omitempty"`
}

// global state bad blah blah blah
var EntriesByPath map[string]*yang.Entry // does not contain namespaces
var ConfigEntries map[string]*yang.Entry // does not contain state nodes, does contain namespaces

func CollectPaths(firstRun bool, path string, entries map[string]*yang.Entry) {
	for k, v := range entries {
		//fmt.Println("Testing...", v.Parent.Prefix.Parent.NName(), " ", v.Prefix.Parent.NName())

		if v.Parent.Prefix.Parent.NName() != v.Prefix.Parent.NName() || firstRun == true {

			showPrefix := false
			if v.Parent.Augmented != nil {
				for _, vv := range v.Parent.Augmented {
					if vv.Prefix.Name == v.Prefix.Name {
						showPrefix = true
					}
				}
			}
			if showPrefix || firstRun {
				currentPrefix := v.Prefix.Parent.NName()
				k = currentPrefix + ":" + k
			}
		}

		//fmt.Println(v.Prefix.Parent.NName())
		EntriesByPath[path+k] = v
		//fmt.Println(path + k)
		if !v.ReadOnly() {
			ConfigEntries[path+k] = v
		}
		CollectPaths(false, path+k+"/", v.Dir)
	}
}

func resolveLeafRef(entry *yang.Entry) *yang.Entry {
	if entry.Type.Name != "leafref" {
		return entry
	}

	path := entry.Type.Path
	// remove all text between square brackets in the path using the
	// regexp module, this allows us to get the path to the leafref
	re := regexp.MustCompile(`\[.*\]`)
	path = re.ReplaceAllString(path, "")
	e := entry.Find(path)
	if e != nil {
		return resolveLeafRef(e)
	}
	return nil
}

func resolveIdentities(entry *yang.Entry) []string {
	//fmt.Println("resolution name: ", entry.Type.Name, entry.Name)
	if entry.Type.IdentityBase == nil {
		return nil
	}
	/*	if entry.Type.Name != "identityref" {
			fmt.Println("This is not of type identityref: ", entry.Type.Name)
			return nil
		}
	*/
	identities := make([]string, 0)

	if len(entry.Type.IdentityBase.Values) == 0 {
		identities = append(identities, "n/a")
	}
	for _, v := range entry.Type.IdentityBase.Values {
		strResult := fmt.Sprintf("%s:%s", v.ParentNode().NName(), v.Name)
		identities = append(identities, strResult)
	}
	if len(identities) == 0 {
		return nil
	}
	return identities
}

func resolveEnums(entry *yang.Entry) []string {
	if entry.Type.Kind.String() != "enumeration" {
		log.Fatalf("Shouldn't happen")
		return nil
	}

	enums := make([]string, 0)
	if len(entry.Type.Enum.ToString) == 0 {
		log.Fatal("zero")
	}

	for _, v := range entry.Type.Enum.ToString {
		enums = append(enums, v)
	}

	return enums
}

func printModules(path string, entries map[string]*yang.Entry) {
	for k, v := range entries {
		// skip state nodes
		if v.ReadOnly() {
			continue
		}

		/*
			if strings.Contains(path, "/state/") || strings.HasSuffix(path, "/state") {
				continue
			}
		*/

		fmt.Println(path+k, " ", v.Kind, ", key: ", v.Key)
		if v.Type != nil {
			fmt.Println(v.Type.Name)
			if v.Type.Name == "leafref" {
				fmt.Println(v.Type.Path)

				//remove all text between square brackets in the path using the
				//regexp module, this allows us to get the path to the leafref
				re := regexp.MustCompile(`\[.*\]`)
				path := re.ReplaceAllString(v.Type.Path, "")

				fmt.Println(path)

				fmt.Println(v.Type.Base.Name)
				e := v.Find(path)

				if e != nil {
					fmt.Println(e.Type.Name)
					// sometimes a leafref points to another leafref, deal with
					// that situation.
					if e.Type.Name == "leafref" {
						path := re.ReplaceAllString(e.Type.Path, "")
						fmt.Println(path)
						ee := e.Find(path)
						if ee != nil {
							fmt.Println(ee.Type.Name)
							//fmt.Println(ee.Node.NName())
						}
					}
				}
			}

			if v.Type.Name == "enumeration" {
				fmt.Println("Enum: ", v.Type.Enum)
				for _, val := range v.Type.Enum.ToString {
					fmt.Println(" ", val)
				}
			}
			if v.Type.Name == "identityref" {
				for _, v := range v.Type.IdentityBase.Values {
					//fmt.Println(v.Name, " ", v.ParentNode().NName())
					fmt.Println(v.Name, " ", v.ParentNode().NName())
				}
				//	fmt.Println("Identity: ", v.Type.IdentityBase.Values)
			}
		}
		fmt.Println("--")
		printModules(path+k+"/", v.Dir)
	}
}

// initial commit
func main() {

	var (
		outputSchema = flag.String("outfile", "schema.json", "output json file that contains the schema from the input yang")
		skipModules  = flag.String("skipmodules", "", "comma separated set of modules to skip, e.g. 'ietf-interfaces'")
	)

	flag.Parse()

	//fmt.Println(inputFiles)
	EntriesByPath = make(map[string]*yang.Entry)
	ConfigEntries = make(map[string]*yang.Entry)

	modules := yang.NewModules()

	for _, filename := range flag.Args() {

		err := modules.Read(filename)

		if err != nil {
			log.Fatalln("Error loading yang module:", err)

		}
	}

	errs := modules.Process()
	if errs != nil {
		fmt.Println(errs)
	}

	thing1, _ := modules.FindModuleByNamespace("openconfig-network-instance")
	//if err != nil {
	//	log.Fatal(err)
	//}
	fmt.Println(thing1)

	for k, _ := range modules.Modules {
		//fmt.Println(k)

		skip := false
		toSkip := strings.Split(*skipModules, ",")

		for _, skipMod := range toSkip {
			if strings.Contains(k, skipMod) {
				//fmt.Println("skipping: ", k)
				skip = true
			}
		}

		if !strings.Contains(k, "@") && skip == false {
			entry, err := modules.GetModule(k)
			if err != nil {
				log.Fatal("Problem finding module ", err)
			}
			CollectPaths(true, "/", entry.Dir)
		}
	}

	// - schema is top level object
	//   - properties are a list of things that contain a unique set of objects, e.g. /interfaces
	//   - an item with a key is an array, e.g. /interfaces/interface
	//   - an identityref is an 'enum' single object with namespace
	//   - an enumeration is an 'enum' single object without namespace
	//   - a leafref must be reoslved
	//   - a type

	// write out the schema to disk
	schema := Schema{
		Id:          "https://example.com/person.schema.json",
		Schema:      "http://json-schema.org/draft-04/schema#",
		Title:       "openconfig",
		Description: "ng schema gen",
		Type:        "object",
		Properties:  map[string]interface{}{},
	}

	for k, _ := range ConfigEntries {
		pathArray := strings.Split(k, "/")[1:] //skip the first element which is empty
		var currentProperty Property
		currentPath := ""

		for pathidx, pathElem := range pathArray {
			currentPath = currentPath + "/" + pathElem
			currentEntry := EntriesByPath[currentPath]

			/*
				if pathElem == "loopback-mode" {
					fmt.Println("yes")
				}

				fmt.Println(k)
				fmt.Println("current path: ", currentPath)
				fmt.Println("container: ", currentEntry.IsContainer())
				fmt.Println("leaf: ", currentEntry.IsLeaf())
				fmt.Println("list: ", currentEntry.IsList())
				fmt.Println("key: ", currentEntry.Key)
				fmt.Println("type: ", currentEntry.Type)
				if currentEntry.Type != nil {
					fmt.Println("type: ", currentEntry.Type.Kind)
				}

			*/
			if pathidx == 0 {
				//making an assumption here that the outermost 'thing' is a container not a list.
				if schema.Properties[pathElem] == nil {
					//create the first element if it's not there yet
					currentProperty = Property{
						Type:        "object",
						Description: currentEntry.Description,
						Properties:  map[string]interface{}{},
					}
					schema.Properties[pathElem] = currentProperty
				} else {
					currentProperty = schema.Properties[pathElem].(Property)
				}
				continue
			}

			if currentEntry.IsList() {

				if currentProperty.Properties[pathElem] == nil {
					currentProperty.Properties[pathElem] = Property{
						Type:        "array",
						Description: currentEntry.Description,
						Items: &Property{
							Type: "object",
							//Description: currentEntry.Description,
							Properties: map[string]interface{}{},
						},
					}
					currentProperty = *currentProperty.Properties[pathElem].(Property).Items
				} else {
					currentProperty = *currentProperty.Properties[pathElem].(Property).Items
				}
			}

			if currentEntry.IsContainer() {
				if currentProperty.Properties[pathElem] == nil {
					currentProperty.Properties[pathElem] = Property{
						Type:        "object",
						Description: currentEntry.Description,
						Properties:  map[string]interface{}{},
					}
					currentProperty = currentProperty.Properties[pathElem].(Property)
				} else {
					currentProperty = currentProperty.Properties[pathElem].(Property)
				}

			}

			if currentEntry.IsLeaf() || currentEntry.IsLeafList() {

				leafProperty := Property{
					Description: currentEntry.Description,
				}

				leafType := currentEntry.Type.Kind.String()

				if leafType == "leafref" {
					currentEntry = resolveLeafRef(currentEntry)
					//fmt.Println(currentPath)
					//fmt.Println("result of leafref resolution: ", currentEntry.Type.Kind.String())
					leafType = currentEntry.Type.Kind.String()
				}

				if strings.Contains(leafType, "float") {
					leafProperty.Type = "number"

				}
				if strings.Contains(leafType, "int") {
					leafProperty.Type = "integer"
				}
				if leafType == "string" {
					leafProperty.Type = "string"
				}
				if leafType == "boolean" {
					leafProperty.Type = "boolean"
				}
				if leafType == "identityref" {
					result := resolveIdentities(currentEntry)
					leafProperty.Enum = result
				}
				if leafType == "enumeration" {
					result := resolveEnums(currentEntry)
					if result == nil {
						log.Fatal(("Got to nil"))
						continue
					}
					leafProperty.Enum = result
				}

				if leafType == "union" {
					// need to figure out a better way to handle unions in the
					// schema, using string for now.
					leafProperty.Type = "string"
				}

				if currentEntry.IsLeafList() {
					currentProperty.Properties[pathElem] = Property{
						Type:        "array",
						Description: currentEntry.Description,
						Items:       &leafProperty,
					}
				}

				if currentEntry.IsLeaf() {
					currentProperty.Properties[pathElem] = leafProperty
				}

			}

		}
	}

	/* below is an example of how the schema looks filled in
	testSchema := Schema{
		Id:          "https://example.com/person.schema.json",
		Schema:      "http://json-schema.org/draft-04/schema#",
		Title:       "openconfig",
		Description: "ng schema gen",
		Type:        "object",
		Properties: map[string]interface{}{"interfaces": Property{
			Type:        "object",
			Description: "",
			Format:      "",
			Properties: map[string]interface{}{
				"interface": Property{
					Type:        "array",
					Description: "",
					Items: &Property{
						Type: "object",
						Properties: map[string]interface{}{
							"config": &Property{
								Type:        "object",
								Description: "",
								Properties: map[string]interface{}{
									"name": &Property{
										Type:        "string",
										Description: "Interface name",
									},
									"type": &Property{
										Description: "Interface type",
										Enum:        []interface{}{"iana-if-type:ethernetCsmacd"},
									},
								},
							},
							"name": &Property{
								Type:        "string",
								Description: "Interface name",
							},
						},
					},
				},
			},
		},
		},
	}
	*/

	// marshal schema
	schemaBytes, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	// write schema to file
	schemaFile, err := os.Create(*outputSchema)
	if err != nil {
		log.Fatal(err)
	}
	defer schemaFile.Close()
	schemaFile.Write(schemaBytes)

}
