package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
)

type Compilation struct {
	// XMLName       xml.Name `xml:"system.web>compilation"`
	TempDirectory string `xml:"tempDirectory,attr"`
	Assemblies    []struct {
		XMLName xml.Name `xml:"assemblies"`
		Add     struct {
			Assembly string `xml:assembly,attr"`
		} `xml:"add"`
	} `xml:"assemblies"`
	// BuildProviders     []BuildProviders
	// ExpressionBuilders []ExpressionBuilders
}

// type SystemWeb struct {
// 	XMLName     xml.Name `xml:"system.web"`
// 	Compilation Compilation
// }

type Configuration struct {
	XMLName     xml.Name    `xml:"configuration"`
	Compilation Compilation `xml:"system.web>compilation"`
	AllXML      string      `xml:",innerxml"`
}

func main() {
	file, _ := os.Open("web.config")
	data, _ := ioutil.ReadAll(file)
	l := Configuration{}
	err := xml.Unmarshal(data, &l)
	fmt.Println("ERR:", err)

	l.Compilation.TempDirectory = "C:\\tmp"

	webXMLFile, err := os.OpenFile("web2.config", os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}
	defer webXMLFile.Close()

	xmlstring, err := xml.MarshalIndent(l, "", "    ")
	if err != nil {
		panic(err)
	}
	xmlstring = []byte(xml.Header + string(xmlstring))
	webXMLFile.Write(xmlstring)

	// enc := xml.NewEncoder(webXMLFile)
	// enc.Indent("  ", "    ")
	// if err := enc.Encode(l); err != nil {
	// 	fmt.Printf("error: %v\n", err)
	// }
}
