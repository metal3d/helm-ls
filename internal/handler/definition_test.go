package handler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/mrjosh/helm-ls/internal/charts"
	lsplocal "github.com/mrjosh/helm-ls/internal/lsp"
	gotemplate "github.com/mrjosh/helm-ls/internal/tree-sitter/gotemplate"
	sitter "github.com/smacker/go-tree-sitter"
	lsp "go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	yamlv3 "gopkg.in/yaml.v3"
)

var testFileContent = `
{{ $variable := "text" }} # line 1
{{ $variable }}           # line 2

{{ $someOther := "text" }}# line 4
{{ $variable }}           # line 5

{{ range $index, $element := pipeline }}{{ $index }}{{ $element }}{{ end }} # line 7
{{ .Values.foo }} # line 8
{{ .Values.something.nested }} # line 9

{{ range .Values.list }}
{{ . }} # line 12
{{ end }}
`

var (
	testDocumentTemplateURI = uri.URI("file:///test.yaml")
	testValuesURI           = uri.URI("file:///values.yaml")
	testOtherValuesURI      = uri.URI("file:///values.other.yaml")
	valuesContent           = `
foo: bar
something: 
  nested: false
list:
  - test
`
)

func genericDefinitionTest(t *testing.T, position lsp.Position, expectedLocations []lsp.Location, expectedError error) {
	var node yamlv3.Node
	err := yamlv3.Unmarshal([]byte(valuesContent), &node)
	if err != nil {
		t.Fatal(err)
	}
	handler := &langHandler{
		linterName: "helm-lint",
		connPool:   nil,
		documents:  nil,
	}

	parser := sitter.NewParser()
	parser.SetLanguage(gotemplate.GetLanguage())
	tree, _ := parser.ParseCtx(context.Background(), nil, []byte(testFileContent))
	doc := &lsplocal.Document{
		Content: testFileContent,
		URI:     testDocumentTemplateURI,
		Ast:     tree,
	}

	location, err := handler.definitionAstParsing(&charts.Chart{
		ChartMetadata: &charts.ChartMetadata{},
		ValuesFiles: &charts.ValuesFiles{
			MainValuesFile: &charts.ValuesFile{
				Values:    make(map[string]interface{}),
				ValueNode: node,
				URI:       testValuesURI,
			},
			AdditionalValuesFiles: []*charts.ValuesFile{},
		},
		RootURI: "",
	}, doc, position)

	if err != nil && err.Error() != expectedError.Error() {
		t.Errorf("expected %v, got %v", expectedError, err)
	}

	if reflect.DeepEqual(location, expectedLocations) == false {
		t.Errorf("expected %v, got %v", expectedLocations, location)
	}
}

// Input:
// {{ $variable }}           # line 2
// -----|                    # this line incides the coursor position for the test
func TestDefinitionVariable(t *testing.T) {
	genericDefinitionTest(t, lsp.Position{Line: 2, Character: 8}, []lsp.Location{
		{
			URI: testDocumentTemplateURI,
			Range: lsp.Range{
				Start: lsp.Position{
					Line:      1,
					Character: 3,
				},
			},
		},
	}, nil)
}

func TestDefinitionNotImplemented(t *testing.T) {
	genericDefinitionTest(t, lsp.Position{Line: 1, Character: 1}, []lsp.Location{},
		fmt.Errorf("Definition not implemented for node type %s", "{{"))
}

// Input:
// {{ range $index, $element := pipeline }}{{ $index }}{{ $element }}{{ end }} # line 7
// -----------------------------------------------------------|
// Expected:
// {{ range $index, $element := pipeline }}{{ $index }}{{ $element }}{{ end }} # line 7
// -----------------|
func TestDefinitionRange(t *testing.T) {
	genericDefinitionTest(t, lsp.Position{Line: 7, Character: 60}, []lsp.Location{
		{
			URI: testDocumentTemplateURI,
			Range: lsp.Range{
				Start: lsp.Position{
					Line:      7,
					Character: 17,
				},
			},
		},
	}, nil)
}

// Input:
// {{ .Values.foo }} # line 8
// ------------|
func TestDefinitionValue(t *testing.T) {
	genericDefinitionTest(t, lsp.Position{Line: 8, Character: 13}, []lsp.Location{
		{
			URI: testValuesURI,
			Range: lsp.Range{
				Start: lsp.Position{
					Line:      1,
					Character: 0,
				},
				End: lsp.Position{
					Line:      1,
					Character: 0,
				},
			},
		},
	}, nil)
}

// Input:
// {{ range .Values.list }}
// {{ . }} # line 12
// ---|
func TestDefinitionValueInList(t *testing.T) {
	genericDefinitionTest(t, lsp.Position{Line: 12, Character: 3}, []lsp.Location{
		{
			URI: testValuesURI,
			Range: lsp.Range{
				Start: lsp.Position{
					Line:      4,
					Character: 0,
				},
				End: lsp.Position{
					Line:      4,
					Character: 0,
				},
			},
		},
	}, nil)
}

// Input:
// {{ . }} # line 9
// ----------------------|
func TestDefinitionValueNested(t *testing.T) {
	genericDefinitionTest(t, lsp.Position{Line: 9, Character: 26}, []lsp.Location{
		{
			URI: testValuesURI,
			Range: lsp.Range{
				Start: lsp.Position{
					Line:      3,
					Character: 2,
				},
				End: lsp.Position{
					Line:      3,
					Character: 2,
				},
			},
		},
	}, nil)
}

// {{ .Values.foo }} # line 8
// ------|
func TestDefinitionValueFile(t *testing.T) {
	genericDefinitionTest(t, lsp.Position{Line: 8, Character: 7}, []lsp.Location{
		{
			URI: testValuesURI,
			Range: lsp.Range{
				Start: lsp.Position{
					Line:      1,
					Character: 0,
				},
				End: lsp.Position{
					Line:      1,
					Character: 0,
				},
			},
		},
	}, nil)
}

func genericDefinitionTestMultipleValuesFiles(t *testing.T, position lsp.Position, expectedLocations []lsp.Location, expectedError error) {
	var node yamlv3.Node
	err := yamlv3.Unmarshal([]byte(valuesContent), &node)
	if err != nil {
		t.Fatal(err)
	}
	handler := &langHandler{
		linterName: "helm-lint",
		connPool:   nil,
		documents:  nil,
	}

	parser := sitter.NewParser()
	parser.SetLanguage(gotemplate.GetLanguage())
	tree, _ := parser.ParseCtx(context.Background(), nil, []byte(testFileContent))
	doc := &lsplocal.Document{
		Content: testFileContent,
		URI:     testDocumentTemplateURI,
		Ast:     tree,
	}

	location, err := handler.definitionAstParsing(&charts.Chart{
		ValuesFiles: &charts.ValuesFiles{
			MainValuesFile: &charts.ValuesFile{
				Values:    make(map[string]interface{}),
				ValueNode: node,
				URI:       testValuesURI,
			},
			AdditionalValuesFiles: []*charts.ValuesFile{
				{
					Values:    make(map[string]interface{}),
					ValueNode: node,
					URI:       testOtherValuesURI,
				},
			},
		},
		RootURI: "",
	}, doc, position)

	if err != nil && err.Error() != expectedError.Error() {
		t.Errorf("expected %v, got %v", expectedError, err)
	}

	if reflect.DeepEqual(location, expectedLocations) == false {
		t.Errorf("expected %v, got %v", expectedLocations, location)
	}
}

// {{ .Values.foo }} # line 8
// ------|
func TestDefinitionValueFileMulitpleValues(t *testing.T) {
	genericDefinitionTestMultipleValuesFiles(t, lsp.Position{Line: 8, Character: 7}, []lsp.Location{
		{
			URI: testValuesURI,
			Range: lsp.Range{
				Start: lsp.Position{
					Line:      1,
					Character: 0,
				},
				End: lsp.Position{
					Line:      1,
					Character: 0,
				},
			},
		}, {
			URI: testOtherValuesURI,
			Range: lsp.Range{
				Start: lsp.Position{
					Line:      1,
					Character: 0,
				},
				End: lsp.Position{
					Line:      1,
					Character: 0,
				},
			},
		},
	}, nil)
}
