/*
Copyright 2018 The Kubernetes Authors.

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

package get

import (
	"strings"

	"github.com/spf13/cobra"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kubernetes/pkg/kubectl/cmd/util/openapi"
	"k8s.io/kubernetes/pkg/printers"
)

// PrintFlags composes common printer flag structs
// used in the Get command.
type PrintFlags struct {
	JSONYamlPrintFlags *printers.JSONYamlPrintFlags
	NamePrintFlags     *printers.NamePrintFlags
	TemplateFlags      *printers.KubeTemplatePrintFlags
	CustomColumnsFlags *printers.CustomColumnsPrintFlags
	HumanReadableFlags *HumanPrintFlags

	NoHeaders    *bool
	OutputFormat *string
}

// EnsureWithNamespace ensures that humanreadable flags return
// a printer capable of printing with a "namespace" column.
func (f *PrintFlags) EnsureWithNamespace() error {
	return f.HumanReadableFlags.EnsureWithNamespace()
}

// EnsureWithKind ensures that humanreadable flags return
// a printer capable of including resource kinds.
func (f *PrintFlags) EnsureWithKind(kind schema.GroupKind) error {
	return f.HumanReadableFlags.EnsureWithKind(kind)
}

// Copy returns a copy of PrintFlags for mutation
func (f *PrintFlags) Copy() PrintFlags {
	printFlags := *f
	return printFlags
}

// UseOpenAPIColumns modifies the output format, as well as the
// "allowMissingKeys" option for template printers, to values
// defined in the OpenAPI schema of a resource.
func (f *PrintFlags) UseOpenAPIColumns(api openapi.Resources, mapping *meta.RESTMapping) error {
	// Found openapi metadata for this resource
	schema := api.LookupResource(mapping.GroupVersionKind)
	if schema == nil {
		// Schema not found, return empty columns
		return nil
	}

	columns, found := openapi.GetPrintColumns(schema.GetExtensions())
	if !found {
		// Extension not found, return empty columns
		return nil
	}

	parts := strings.SplitN(columns, "=", 2)
	if len(parts) < 2 {
		return nil
	}

	allowMissingKeys := true
	f.OutputFormat = &parts[0]
	f.TemplateFlags.TemplateArgument = &parts[1]
	f.TemplateFlags.AllowMissingKeys = &allowMissingKeys
	return nil
}

// ToPrinter attempts to find a composed set of PrintFlags suitable for
// returning a printer based on current flag values.
func (f *PrintFlags) ToPrinter() (printers.ResourcePrinter, error) {
	outputFormat := ""
	if f.OutputFormat != nil {
		outputFormat = *f.OutputFormat
	}

	noHeaders := false
	if f.NoHeaders != nil {
		noHeaders = *f.NoHeaders
	}
	f.HumanReadableFlags.NoHeaders = noHeaders
	f.CustomColumnsFlags.NoHeaders = noHeaders

	if f.TemplateFlags.TemplateArgument != nil {
		f.CustomColumnsFlags.TemplateArgument = *f.TemplateFlags.TemplateArgument
	}

	if p, err := f.JSONYamlPrintFlags.ToPrinter(outputFormat); !printers.IsNoCompatiblePrinterError(err) {
		return p, err
	}

	if p, err := f.HumanReadableFlags.ToPrinter(outputFormat); !printers.IsNoCompatiblePrinterError(err) {
		return p, err
	}

	if p, err := f.TemplateFlags.ToPrinter(outputFormat); !printers.IsNoCompatiblePrinterError(err) {
		return p, err
	}

	if p, err := f.CustomColumnsFlags.ToPrinter(outputFormat); !printers.IsNoCompatiblePrinterError(err) {
		return p, err
	}

	if p, err := f.NamePrintFlags.ToPrinter(outputFormat); !printers.IsNoCompatiblePrinterError(err) {
		return p, err
	}

	return nil, printers.NoCompatiblePrinterError{Options: f}
}

// AddFlags receives a *cobra.Command reference and binds
// flags related to humanreadable and template printing.
func (f *PrintFlags) AddFlags(cmd *cobra.Command) {
	f.JSONYamlPrintFlags.AddFlags(cmd)
	f.NamePrintFlags.AddFlags(cmd)
	f.TemplateFlags.AddFlags(cmd)
	f.HumanReadableFlags.AddFlags(cmd)
	f.CustomColumnsFlags.AddFlags(cmd)

	if f.OutputFormat != nil {
		cmd.Flags().StringVarP(f.OutputFormat, "output", "o", *f.OutputFormat, "Output format. One of: json|yaml|wide|name|custom-columns=...|custom-columns-file=...|go-template=...|go-template-file=...|jsonpath=...|jsonpath-file=... See custom columns [http://kubernetes.io/docs/user-guide/kubectl-overview/#custom-columns], golang template [http://golang.org/pkg/text/template/#pkg-overview] and jsonpath template [http://kubernetes.io/docs/user-guide/jsonpath].")
	}
	if f.NoHeaders != nil {
		cmd.Flags().BoolVar(f.NoHeaders, "no-headers", *f.NoHeaders, "When using the default or custom-column output format, don't print headers (default print headers).")
	}

	// TODO(juanvallejo): This is deprecated - remove
	cmd.Flags().BoolP("show-all", "a", true, "When printing, show all resources (default show all pods including terminated one.)")
	cmd.Flags().MarkDeprecated("show-all", "will be removed in an upcoming release")
}

// NewGetPrintFlags returns flags associated with humanreadable,
// template, and "name" printing, with default values set.
func NewGetPrintFlags(scheme runtime.ObjectConvertor) *PrintFlags {
	outputFormat := ""
	noHeaders := false

	return &PrintFlags{
		OutputFormat: &outputFormat,
		NoHeaders:    &noHeaders,

		JSONYamlPrintFlags: printers.NewJSONYamlPrintFlags(scheme),
		NamePrintFlags:     printers.NewNamePrintFlags("", scheme),
		TemplateFlags:      printers.NewKubeTemplatePrintFlags(),
		HumanReadableFlags: NewHumanPrintFlags(),
		CustomColumnsFlags: printers.NewCustomColumnsPrintFlags(),
	}
}
