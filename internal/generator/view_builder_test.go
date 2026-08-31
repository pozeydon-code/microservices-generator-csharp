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

func TestBuildSolutionViewProjectsRelationshipViews(t *testing.T) {
	view, err := buildSolutionView(relationshipTestConfig())
	if err != nil {
		t.Fatalf("build solution view: %v", err)
	}

	service := view.Services[0]
	entities := map[string]EntityView{}
	for _, entity := range service.Entities {
		entities[entity.Name] = entity
	}

	order := entities["Order"]
	orderItem := entities["OrderItem"]
	customer := entities["Customer"]

	if !reflect.DeepEqual(orderItem.RelationshipScalarFields, []RelationshipScalarFieldView{{Name: "OrderId", CamelName: "orderId", Type: "Guid", ContractType: "Guid", ValueAccess: "OrderId", Required: true}}) {
		t.Fatalf("unexpected required FK scalar view: %#v", orderItem.RelationshipScalarFields)
	}
	if !reflect.DeepEqual(orderItem.ReferenceNavigations, []ReferenceNavigationView{{Name: "Order", TargetEntity: "Order", Nullable: false, Initializer: " = null!;"}}) {
		t.Fatalf("unexpected required reference navigation view: %#v", orderItem.ReferenceNavigations)
	}
	if !reflect.DeepEqual(order.CollectionNavigations, []CollectionNavigationView{{Name: "Items", TargetEntity: "OrderItem"}}) {
		t.Fatalf("unexpected principal collection view: %#v", order.CollectionNavigations)
	}
	if !reflect.DeepEqual(order.RelationshipScalarFields, []RelationshipScalarFieldView{{Name: "CustomerId", CamelName: "customerId", Type: "Guid?", ContractType: "Guid?", ValueAccess: "CustomerId", Required: false}}) {
		t.Fatalf("unexpected optional FK scalar view: %#v", order.RelationshipScalarFields)
	}
	if !reflect.DeepEqual(order.ReferenceNavigations, []ReferenceNavigationView{{Name: "Customer", TargetEntity: "Customer", Nullable: true, Initializer: ""}}) {
		t.Fatalf("unexpected optional reference navigation view: %#v", order.ReferenceNavigations)
	}
	if !reflect.DeepEqual(customer.CollectionNavigations, []CollectionNavigationView{{Name: "Orders", TargetEntity: "Order"}}) {
		t.Fatalf("unexpected optional principal collection view: %#v", customer.CollectionNavigations)
	}
	if !reflect.DeepEqual(orderItem.EFRelationships, []EFRelationshipView{{PrincipalEntity: "Order", DependentEntity: "OrderItem", ForeignKeyName: "OrderId", PrincipalNavigation: "Items", DependentNavigation: "Order", Required: true, IsRequiredCall: ".IsRequired()"}}) {
		t.Fatalf("unexpected required EF relationship view: %#v", orderItem.EFRelationships)
	}
	if !reflect.DeepEqual(order.EFRelationships, []EFRelationshipView{{PrincipalEntity: "Customer", DependentEntity: "Order", ForeignKeyName: "CustomerId", PrincipalNavigation: "Orders", DependentNavigation: "Customer", Required: false, IsRequiredCall: ".IsRequired(false)"}}) {
		t.Fatalf("unexpected optional EF relationship view: %#v", order.EFRelationships)
	}
}

func TestBuildSolutionViewUsesExplicitForeignKeyFieldAsRelationshipScalar(t *testing.T) {
	cfg := relationshipTestConfig()
	for serviceIndex := range cfg.Services {
		for entityIndex := range cfg.Services[serviceIndex].Entities {
			if cfg.Services[serviceIndex].Entities[entityIndex].Name == "OrderItem" {
				cfg.Services[serviceIndex].Entities[entityIndex].Fields = append(cfg.Services[serviceIndex].Entities[entityIndex].Fields, spec.Field{Name: "OrderId", Type: "Guid"})
			}
		}
	}

	view, err := buildSolutionView(cfg)
	if err != nil {
		t.Fatalf("build solution view: %v", err)
	}

	service := view.Services[0]
	entities := map[string]EntityView{}
	for _, entity := range service.Entities {
		entities[entity.Name] = entity
	}
	orderItem := entities["OrderItem"]

	if !reflect.DeepEqual(orderItem.RelationshipScalarFields, []RelationshipScalarFieldView{{Name: "OrderId", CamelName: "orderId", Type: "Guid", ContractType: "Guid", ValueAccess: "OrderId", Required: true}}) {
		t.Fatalf("unexpected FK scalar metadata for explicit field: %#v", orderItem.RelationshipScalarFields)
	}
	if got := countFieldViewsNamed(orderItem.Fields, "OrderId"); got != 0 {
		t.Fatalf("expected explicit FK field to be emitted only as relationship scalar metadata, got %d entity fields", got)
	}
	if got := countFieldViewsNamed(orderItem.NonIDFields, "OrderId"); got != 0 {
		t.Fatalf("expected explicit FK field to be omitted from non-ID fields, got %d non-ID fields", got)
	}
	if !reflect.DeepEqual(orderItem.EFRelationships, []EFRelationshipView{{PrincipalEntity: "Order", DependentEntity: "OrderItem", ForeignKeyName: "OrderId", PrincipalNavigation: "Items", DependentNavigation: "Order", Required: true, IsRequiredCall: ".IsRequired()"}}) {
		t.Fatalf("unexpected EF relationship metadata for explicit field: %#v", orderItem.EFRelationships)
	}
}

func countFieldViewsNamed(fields []FieldView, name string) int {
	count := 0
	for _, field := range fields {
		if field.Name == name {
			count++
		}
	}
	return count
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

func relationshipTestConfig() spec.Config {
	optional := false
	return spec.Config{
		Solution: spec.Solution{Name: "SalesPlatform", Description: "Relationship generation regression."},
		Services: []spec.Service{{
			Name: "OrderingService",
			Entities: []spec.Entity{
				{Name: "Customer", Fields: []spec.Field{{Name: "Id", Type: "Guid"}, {Name: "Name", Type: "string"}}},
				{Name: "Order", Fields: []spec.Field{{Name: "Id", Type: "Guid"}, {Name: "Number", Type: "string"}}},
				{Name: "OrderItem", Fields: []spec.Field{{Name: "Id", Type: "Guid"}, {Name: "Sku", Type: "string"}}},
			},
			Relationships: []spec.Relationship{
				{Multiplicity: "one-to-many", PrincipalEntity: "Order", DependentEntity: "OrderItem", ForeignKeyName: "OrderId", PrincipalNavigation: "Items", DependentNavigation: "Order"},
				{Multiplicity: "many-to-one", PrincipalEntity: "Customer", DependentEntity: "Order", ForeignKeyName: "CustomerId", Required: &optional, PrincipalNavigation: "Orders", DependentNavigation: "Customer"},
			},
		}},
	}
}
