package generator

import (
	"reflect"
	"testing"

	"github.com/pozeydon-code/microservices-generator-csharp/internal/spec"
)

func TestServiceRoutePrefixUsesDeterministicKebabCase(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		want        string
	}{
		{name: "pascal case service suffix is preserved", serviceName: "ProductService", want: "product-service"},
		{name: "multi word service suffix is preserved", serviceName: "OrderFulfillmentService", want: "order-fulfillment-service"},
		{name: "camel case is split", serviceName: "productService", want: "product-service"},
		{name: "acronym boundary is stable", serviceName: "HTTPGatewayService", want: "http-gateway-service"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := serviceRoutePrefix(tt.serviceName); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestBuildSolutionViewGatewayRoutesAreDeterministicForWebApis(t *testing.T) {
	cfg := gatewayTestConfig()
	cfg.Generation.Gateway.Enabled = true

	view, err := buildSolutionView(cfg)
	if err != nil {
		t.Fatalf("build solution view: %v", err)
	}

	wantProject := ProjectView{
		Name:         "ShopPlatform.Gateway",
		Directory:    "Gateway",
		FileName:     "ShopPlatform.Gateway.csproj",
		Path:         "Gateway/ShopPlatform.Gateway.csproj",
		SolutionPath: "Gateway/ShopPlatform.Gateway.csproj",
		GUID:         deterministicGUID("ShopPlatform.Gateway"),
	}
	if !reflect.DeepEqual(view.Gateway.Project, wantProject) {
		t.Fatalf("unexpected gateway project\nexpected: %#v\nactual:   %#v", wantProject, view.Gateway.Project)
	}
	wantRoutes := []GatewayRouteView{
		{ServiceName: "OrderFulfillmentService", RouteID: "order-fulfillment-service-route", ClusterID: "order-fulfillment-service-cluster", Path: "/order-fulfillment-service/{**catch-all}", DestinationAddress: "http://localhost:5100/", LocalPort: 5100},
		{ServiceName: "ProductService", RouteID: "product-service-route", ClusterID: "product-service-cluster", Path: "/product-service/{**catch-all}", DestinationAddress: "http://localhost:5101/", LocalPort: 5101},
	}
	if !reflect.DeepEqual(view.Gateway.Routes, wantRoutes) {
		t.Fatalf("unexpected gateway routes\nexpected: %#v\nactual:   %#v", wantRoutes, view.Gateway.Routes)
	}
}

func TestBuildSolutionViewGatewayDisabledHasNoModelFootprint(t *testing.T) {
	view, err := buildSolutionView(gatewayTestConfig())
	if err != nil {
		t.Fatalf("build solution view: %v", err)
	}

	if view.Gateway.Enabled {
		t.Fatal("expected omitted gateway config to leave gateway disabled")
	}
	if view.Gateway.Project != (ProjectView{}) {
		t.Fatalf("expected no disabled gateway project, got %#v", view.Gateway.Project)
	}
	if len(view.Gateway.Routes) != 0 {
		t.Fatalf("expected no disabled gateway routes, got %#v", view.Gateway.Routes)
	}
	for _, project := range view.Projects {
		if project.Directory == "Gateway" || project.Path == "Gateway/ShopPlatform.Gateway.csproj" {
			t.Fatalf("expected no disabled gateway project in root projects, got %#v", project)
		}
	}
}

func TestBuildSolutionViewGatewayDoesNotEnterServiceLayers(t *testing.T) {
	cfg := gatewayTestConfig()
	cfg.Generation.Gateway.Enabled = true

	view, err := buildSolutionView(cfg)
	if err != nil {
		t.Fatalf("build solution view: %v", err)
	}

	for _, service := range view.Services {
		for _, project := range []ProjectView{service.DomainProject, service.ApplicationProject, service.InfrastructureProject} {
			if project.Name == view.Gateway.Project.Name || project.Path == view.Gateway.Project.Path {
				t.Fatalf("gateway leaked into service layer project for %s: %#v", service.Name, project)
			}
		}
	}
}

func gatewayTestConfig() spec.Config {
	return spec.Config{
		Solution: spec.Solution{Name: "ShopPlatform", Description: "Shop platform."},
		Services: []spec.Service{
			{Name: "ProductService", Entities: []spec.Entity{{Name: "Product", Fields: []spec.Field{{Name: "Id", Type: "Guid"}}}}},
			{Name: "OrderFulfillmentService", Entities: []spec.Entity{{Name: "Order", Fields: []spec.Field{{Name: "Id", Type: "Guid"}}}}},
		},
	}
}
