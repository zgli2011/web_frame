package common

import (
	"strings"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	gw_options "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2/options"
)

// RESTAnnotation represents REST API annotations from proto comments
type RESTAnnotation struct {
	Method      string // HTTP method (GET, POST, PUT, DELETE)
	Path        string // URL path
	Tag         string // Swagger tag
	Summary     string // API summary
	Description string // API description
	Deprecated  bool   // Whether the API is deprecated
}

// OpenAPITag represents service-level tag information
type OpenAPITag struct {
	Name        string
	Description string
}

// ExtractCompleteAnnotations extracts all REST and OpenAPI annotations from protobuf method
func ExtractCompleteAnnotations(method *protogen.Method) *RESTAnnotation {
	annotation := &RESTAnnotation{
		Method:      "",
		Path:        "",
		Tag:         "",
		Summary:     "",
		Description: "",
		Deprecated:  false,
	}

	// 1. Extract from Google API HTTP options (highest priority)
	extractGoogleAPIHTTP(method, annotation)

	// 2. Extract from grpc-gateway OpenAPI v2 options
	extractGrpcGatewayOpenAPI(method, annotation)

	// 3. Extract from service-level tag information
	extractServiceLevelTags(method, annotation)

	// 4. Set intelligent defaults if not specified
	setIntelligentDefaults(method, annotation)

	// 5. Validate and sanitize
	ValidateCompleteAnnotation(annotation)

	return annotation
}

// extractGoogleAPIHTTP extracts Google API HTTP annotations
func extractGoogleAPIHTTP(method *protogen.Method, annotation *RESTAnnotation) {
	opts := method.Desc.Options().(*descriptorpb.MethodOptions)
	if opts == nil {
		return
	}

	// Check for Google API HTTP extension
	if proto.HasExtension(opts, annotations.E_Http) {
		httpRule := proto.GetExtension(opts, annotations.E_Http).(*annotations.HttpRule)
		if httpRule != nil {
			// Extract HTTP method and path
			switch pattern := httpRule.GetPattern().(type) {
			case *annotations.HttpRule_Get:
				annotation.Method = "GET"
				annotation.Path = pattern.Get
			case *annotations.HttpRule_Put:
				annotation.Method = "PUT"
				annotation.Path = pattern.Put
			case *annotations.HttpRule_Post:
				annotation.Method = "POST"
				annotation.Path = pattern.Post
			case *annotations.HttpRule_Delete:
				annotation.Method = "DELETE"
				annotation.Path = pattern.Delete
			case *annotations.HttpRule_Patch:
				annotation.Method = "PATCH"
				annotation.Path = pattern.Patch
			case *annotations.HttpRule_Custom:
				annotation.Method = strings.ToUpper(pattern.Custom.Kind)
				annotation.Path = pattern.Custom.Path
			}
		}
	}

	// Check for deprecated flag
	if opts.GetDeprecated() {
		annotation.Deprecated = true
	}
}

// extractGrpcGatewayOpenAPI extracts grpc-gateway OpenAPI v2 annotations
func extractGrpcGatewayOpenAPI(method *protogen.Method, annotation *RESTAnnotation) {
	opts := method.Desc.Options().(*descriptorpb.MethodOptions)
	if opts == nil {
		return
	}

	// Check for OpenAPI v2 operation extension
	if proto.HasExtension(opts, gw_options.E_Openapiv2Operation) {
		operation := proto.GetExtension(opts, gw_options.E_Openapiv2Operation).(*gw_options.Operation)
		if operation != nil {
			// Extract tags (use first tag if multiple)
			if len(operation.Tags) > 0 && annotation.Tag == "" {
				annotation.Tag = operation.Tags[0]
			}

			// Extract summary
			if operation.Summary != "" && annotation.Summary == "" {
				annotation.Summary = operation.Summary
			}

			// Extract description
			if operation.Description != "" && annotation.Description == "" {
				annotation.Description = operation.Description
			}

			// Extract deprecated flag
			if operation.Deprecated && !annotation.Deprecated {
				annotation.Deprecated = true
			}
		}
	}
}

// extractServiceLevelTags extracts service-level tag information
func extractServiceLevelTags(method *protogen.Method, annotation *RESTAnnotation) {
	service := method.Parent
	serviceOpts := service.Desc.Options().(*descriptorpb.ServiceOptions)
	if serviceOpts == nil {
		return
	}

	// Check for service-level OpenAPI tag extension
	if proto.HasExtension(serviceOpts, gw_options.E_Openapiv2Tag) {
		serviceTag := proto.GetExtension(serviceOpts, gw_options.E_Openapiv2Tag).(*gw_options.Tag)
		if serviceTag != nil && annotation.Tag == "" {
			annotation.Tag = serviceTag.Name
		}
	}
}

// setIntelligentDefaults sets intelligent defaults for missing annotations
func setIntelligentDefaults(method *protogen.Method, annotation *RESTAnnotation) {
	serviceName := method.Parent.GoName
	methodName := method.GoName

	// Set default HTTP method if not specified
	if annotation.Method == "" {
		annotation.Method = DefaultHTTPMethod(methodName)
	}

	// Set default path if not specified
	if annotation.Path == "" {
		annotation.Path = GenerateRESTPath(serviceName, methodName)
	}

	// Set default tag if not specified (prefer service name in snake_case)
	if annotation.Tag == "" {
		annotation.Tag = ToSnakeCase(serviceName)
	}

	// Extract description from comments if not set
	if annotation.Description == "" {
		annotation.Description = ExtractCommentDescription(string(method.Comments.Leading))
	}

	// Set summary from description if not set
	if annotation.Summary == "" && annotation.Description != "" {
		// Use first sentence as summary
		sentences := strings.Split(annotation.Description, "。")
		if len(sentences) > 0 && sentences[0] != "" {
			annotation.Summary = strings.TrimSpace(sentences[0])
		} else {
			sentences = strings.Split(annotation.Description, ".")
			if len(sentences) > 0 && sentences[0] != "" {
				annotation.Summary = strings.TrimSpace(sentences[0])
			} else {
				annotation.Summary = annotation.Description
			}
		}
	}

	// Fallback summary and description
	if annotation.Summary == "" {
		annotation.Summary = methodName + " method"
	}
	if annotation.Description == "" {
		annotation.Description = annotation.Summary
	}
}

// ExtractAllServiceTags extracts all service tags for documentation
func ExtractAllServiceTags(services []*protogen.Service) []*OpenAPITag {
	var tags []*OpenAPITag

	for _, service := range services {
		tag := extractServiceTag(service)
		if tag != nil {
			tags = append(tags, tag)
		}
	}

	return tags
}

// extractServiceTag extracts service-level tag information
func extractServiceTag(service *protogen.Service) *OpenAPITag {
	tag := &OpenAPITag{
		Name:        ToSnakeCase(service.GoName),
		Description: ExtractCommentDescription(string(service.Comments.Leading)),
	}

	serviceOpts := service.Desc.Options().(*descriptorpb.ServiceOptions)
	if serviceOpts != nil {
		// Extract from grpc-gateway service tag extension
		if proto.HasExtension(serviceOpts, gw_options.E_Openapiv2Tag) {
			serviceTag := proto.GetExtension(serviceOpts, gw_options.E_Openapiv2Tag).(*gw_options.Tag)
			if serviceTag != nil {
				if serviceTag.Name != "" {
					tag.Name = serviceTag.Name
				}
				if serviceTag.Description != "" {
					tag.Description = serviceTag.Description
				}
			}
		}
	}

	// Set default description if empty
	if tag.Description == "" {
		tag.Description = service.GoName + " service operations"
	}

	return tag
}

// ValidateCompleteAnnotation validates the complete annotation
func ValidateCompleteAnnotation(annotation *RESTAnnotation) error {
	// Validate HTTP method
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "DELETE": true,
		"PATCH": true, "HEAD": true, "OPTIONS": true,
	}

	if annotation.Method != "" && !validMethods[strings.ToUpper(annotation.Method)] {
		annotation.Method = "POST" // Default fallback
	}

	// Ensure path starts with /
	if annotation.Path != "" && !strings.HasPrefix(annotation.Path, "/") {
		annotation.Path = "/" + annotation.Path
	}

	// Sanitize strings
	annotation.Tag = SanitizeString(annotation.Tag)
	annotation.Summary = SanitizeString(annotation.Summary)
	annotation.Description = SanitizeString(annotation.Description)

	return nil
}
