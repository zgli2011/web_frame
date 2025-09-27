package main

import (
	"flag"
	"fmt"
	"os"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"
	"web_frame/proto/common"
)

const (
	pluginName    = "protoc-gen-web-registry"
	pluginVersion = "v1.0.0"
)

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Printf("%s %s\n", pluginName, pluginVersion)
		os.Exit(0)
	}

	var flags flag.FlagSet
	protogen.Options{
		ParamFunc: flags.Set,
	}.Run(func(gen *protogen.Plugin) error {
		gen.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)

		for _, f := range gen.Files {
			if !f.Generate {
				continue
			}
			generateFile(gen, f)
		}
		return nil
	})
}

func generateFile(gen *protogen.Plugin, file *protogen.File) {
	config := &common.GeneratorConfig{
		Plugin: common.PluginInfo{
			Name:        pluginName,
			Version:     pluginVersion,
			Description: "Web API Registry Generator",
		},
		PackageName: string(file.GoPackageName),
		Imports: []string{
			`"web_frame/check"`,
		},
	}

	codeGen := common.NewCodeGenerator(config, gen)

	if !codeGen.ShouldGenerateForFile(file) {
		return
	}

	filename := codeGen.GetOutputFilename(file, "_registry.pb.go")
	g := gen.NewGeneratedFile(filename, file.GoImportPath)

	codeGen.GenerateHeader(g, file)
	generateRegistryCode(g, file, codeGen)
}


func generateRegistryCode(g *protogen.GeneratedFile, file *protogen.File, codeGen *common.CodeGenerator) {
	// Generate registration function for each service
	for _, service := range file.Services {
		serviceInfo := codeGen.ExtractServiceInfo(service)
		generateServiceRegistry(g, serviceInfo)
	}

	// Generate init function to register all APIs
	g.P("func init() {")
	for _, service := range file.Services {
		g.P("\tRegister", service.GoName, "APIs()")
	}
	g.P("}")
}

func generateServiceRegistry(g *protogen.GeneratedFile, serviceInfo *common.ServiceInfo) {
	serviceName := serviceInfo.GoName

	g.P("// Register", serviceName, "APIs registers all API endpoints for ", serviceName, " service")
	g.P("func Register", serviceName, "APIs() {")

	for _, method := range serviceInfo.Methods {
		generateMethodRegistry(g, serviceInfo, method)
	}

	g.P("}")
	g.P()
}

func generateMethodRegistry(g *protogen.GeneratedFile, serviceInfo *common.ServiceInfo, method *common.MethodInfo) {
	serviceName := serviceInfo.GoName
	methodName := method.GoName
	httpMethod := method.HTTPMethod
	httpPath := method.HTTPPath
	tag := method.Tag
	summary := common.SanitizeString(method.Summary)
	description := common.SanitizeString(method.Description)
	deprecated := method.Deprecated

	g.P("\tcheck.RegisterAPI(check.APIInfo{")
	g.P("\t\tPath:        \"", httpPath, "\",")
	g.P("\t\tMethod:      \"", httpMethod, "\",")
	g.P("\t\tTag:         \"", tag, "\",")
	g.P("\t\tDescription: \"", description, "\",")
	g.P("\t\tService:     \"", serviceName, "\",")
	g.P("\t\tHandler:     \"", methodName, "\",")
	g.P("\t\tSummary:     \"", summary, "\",")
	g.P("\t\tDeprecated:  ", deprecated, ",")
	g.P("\t})")
}

