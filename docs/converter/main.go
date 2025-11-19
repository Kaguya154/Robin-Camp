package main

import (
	"flag"
	"os"

	"github.com/cloudwego/hertz/pkg/common/json"
	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"gopkg.in/yaml.v2"
)

func main() {

	inFlag := flag.String("i", "", "Input JSON file")
	outFlag := flag.String("o", "", "Output JSON file")
	helpFlag := flag.Bool("help", false, "Show help")
	flag.Parse()

	if *inFlag == "" || *outFlag == "" || *helpFlag {
		flag.Usage()
		return
	}

	input, err := os.ReadFile(*inFlag)
	if err != nil {
		panic(err)
	}

	var doc openapi2.T
	if err = json.Unmarshal(input, &doc); err != nil {
		panic(err)
	}

	// 转换为 OpenAPI3
	openapi3Doc, err := openapi2conv.ToV3(&doc)
	if err != nil {
		panic(err)
	}

	// 输出 YAML
	out, err := openapi3Doc.MarshalYAML()
	if err != nil {
		panic(err)
	}

	marshal, err := yaml.Marshal(out)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(*outFlag, marshal, 0644)
	if err != nil {
		panic(err)
	}
}
