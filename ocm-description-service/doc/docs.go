/*
Copyright 2023-2024 Bull SAS

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package doc

// "github.com/alecthomas/template"

// var doc = `{ TODO }`

// type swaggerInfo struct {
// 	Version     string
// 	Host        string
// 	BasePath    string
// 	Schemes     []string
// 	Title       string
// 	Description string
// }

// // SwaggerInfo holds exported Swagger Info so clients can modify it
// var SwaggerInfo = swaggerInfo{
// 	Version:     "1.0",
// 	Host:        "localhost:8083",
// 	BasePath:    "/api/v1",
// 	Schemes:     []string{},
// 	Title:       "Micro Service API Document",
// 	Description: "List of APIs for Micro Service",
// }

// type s struct{}

// func (s *s) ReadDoc() string {
// 	sInfo := SwaggerInfo
// 	sInfo.Description = strings.Replace(sInfo.Description, "\n", "\\n", -1)

// 	t, err := template.New("swagger_info").Funcs(template.FuncMap{
// 		"marshal": func(v interface{}) string {
// 			a, _ := json.Marshal(v)
// 			return string(a)
// 		},
// 	}).Parse(doc)
// 	if err != nil {
// 		return doc
// 	}

// 	var tpl bytes.Buffer
// 	if err := t.Execute(&tpl, sInfo); err != nil {
// 		return doc
// 	}

// 	return tpl.String()
// }

// func init() {
// 	swag.Register(swag.Name, &s{})
// }
