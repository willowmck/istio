// Copyright Istio Authors
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

// +build ignore

package main

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"text/template"

	"github.com/ghodss/yaml"
)

const (
	codeRegex = `^IST\d\d\d\d$`
	nameRegex = `^[[:upper:]]\w*$`
)

// Utility for generating messages.gen.go. Called from gen.go
func main() {
	if len(os.Args) != 3 {
		fmt.Println("Invalid args:", os.Args)
		os.Exit(-1)
	}

	input := os.Args[1]
	output := os.Args[2]

	m, err := read(input)
	if err != nil {
		fmt.Println("Error reading metadata:", err)
		os.Exit(-2)
	}

	err = validate(m)
	if err != nil {
		fmt.Println("Error validating messages:", err)
		os.Exit(-3)
	}

	code, err := generate(m)
	if err != nil {
		fmt.Println("Error generating code:", err)
		os.Exit(-4)
	}

	if err = os.WriteFile(output, []byte(code), os.ModePerm); err != nil {
		fmt.Println("Error writing output file:", err)
		os.Exit(-5)
	}
}

func read(path string) (*messages, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read input file: %v", err)
	}

	m := &messages{}

	if err := yaml.Unmarshal(b, m); err != nil {
		return nil, err
	}

	return m, nil
}

// Enforce that names and codes follow expected regex and are unique
func validate(ms *messages) error {
	codes := make(map[string]bool)
	names := make(map[string]bool)

	for _, m := range ms.Messages {
		matched, err := regexp.MatchString(codeRegex, m.Code)
		if err != nil {
			return err
		}
		if !matched {
			return fmt.Errorf("Error code for message %q must follow the regex %s", m.Name, codeRegex)
		}

		if codes[m.Code] {
			return fmt.Errorf("Error codes must be unique, %q defined more than once", m.Code)
		}
		codes[m.Code] = true

		matched, err = regexp.MatchString(nameRegex, m.Name)
		if err != nil {
			return err
		}
		if !matched {
			return fmt.Errorf("Name for message %q must follow the regex %s", m.Name, nameRegex)
		}

		if names[m.Name] {
			return fmt.Errorf("Message names must be unique, %q defined more than once", m.Name)
		}
		names[m.Name] = true
	}
	return nil
}

var tmpl = `
// GENERATED FILE -- DO NOT EDIT
//

package msg

import (
	"istio.io/istio/galley/pkg/config/analysis/diag"
	"istio.io/istio/pkg/config/resource"
)

var (
	{{- range .Messages}}
	// {{.Name}} defines a diag.MessageType for message "{{.Name}}".
	// Description: {{.Description}}
	{{.Name}} = diag.NewMessageType(diag.{{.Level}}, "{{.Code}}", "{{.Template}}")
	{{end}}
)

// All returns a list of all known message types.
func All() []*diag.MessageType {
	return []*diag.MessageType{
		{{- range .Messages}}
			{{.Name}},
		{{- end}}
	}
}

{{range .Messages}}
// New{{.Name}} returns a new diag.Message based on {{.Name}}.
func New{{.Name}}(r *resource.Instance{{range .Args}}, {{.Name}} {{.Type}}{{end}}) diag.Message {
	return diag.NewMessage(
		{{.Name}},
		r,
		{{- range .Args}}
			{{.Name}},
		{{- end}}
	)
}
{{end}}
`

func generate(m *messages) (string, error) {
	t := template.Must(template.New("code").Parse(tmpl))

	var b bytes.Buffer
	if err := t.Execute(&b, m); err != nil {
		return "", err
	}
	return b.String(), nil
}

type messages struct {
	Messages []message `json:"messages"`
}

type message struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Level       string `json:"level"`
	Description string `json:"description"`
	Template    string `json:"template"`
	Url         string `json:"url"`
	Args        []arg  `json:"args"`
}

type arg struct {
	Name string `json:"name"`
	Type string `json:"type"`
}
