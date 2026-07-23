package application

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/pozeydon-code/generator-microservices-go/internal/spec"
)

func TestServicePlanGenerationReturnsUIReadyFileListWithoutWriting(t *testing.T) {
	cfg := validConfig()
	loader := &fakeConfigLoader{cfg: cfg}
	validator := &fakeConfigValidator{}
	gen := &fakeGenerator{files: []GeneratedFile{
		{Path: "README.md", Content: []byte("readme")},
		{Path: "src/ProductService/Product.cs", Content: []byte("entity")},
	}}
	writer := &fakeOutputWriter{}
	service := NewService(Ports{
		ConfigLoader:    loader,
		ConfigValidator: validator,
		Generator:       gen,
		OutputWriter:    writer,
		OutputPlanner: fakeOutputPlanner{plan: OutputPlan{
			OutputDir:     "/abs/generated",
			Action:        "replace",
			ForceRequired: true,
			ForceUsed:     true,
			DeletedFiles:  []string{"src/OldEndpoint.cs"},
			Files: []OutputPlannedFile{
				{Path: "README.md", Action: "replace"},
				{Path: "src/ProductService/Product.cs", Action: "unchanged"},
			},
		}},
	})

	plan, err := service.PlanGeneration(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated", Force: true})

	if err != nil {
		t.Fatalf("expected plan, got %v", err)
	}
	if loader.path != "microgen.json" {
		t.Fatalf("expected config path to be loaded, got %q", loader.path)
	}
	if !reflect.DeepEqual(validator.cfg, cfg) {
		t.Fatalf("expected validator to receive loaded config")
	}
	if !reflect.DeepEqual(gen.cfg, cfg) {
		t.Fatalf("expected generator to receive loaded config")
	}
	if writer.called {
		t.Fatal("expected planning not to write output")
	}
	if plan.OutputDir != "/abs/generated" {
		t.Fatalf("expected resolved output dir, got %q", plan.OutputDir)
	}
	if plan.OutputAction != "replace" || !plan.ForceRequired || !plan.ForceUsed {
		t.Fatalf("expected mapped output status, got %#v", plan)
	}
	if plan.FileCount != 2 {
		t.Fatalf("expected file count 2, got %d", plan.FileCount)
	}
	if plan.ExtraFileCount != 1 || !reflect.DeepEqual(plan.DeletedFiles, []string{"src/OldEndpoint.cs"}) {
		t.Fatalf("expected deleted file metadata to be mapped, got count=%d files=%#v", plan.ExtraFileCount, plan.DeletedFiles)
	}
	expectedSummary := ConfigSummary{
		SolutionName:        "CommercePlatform",
		SolutionDescription: "Product management.",
		TargetFramework:     "net8.0",
		SolutionFormat:      "sln",
		ServiceCount:        2,
		EntityCount:         3,
		ValueObjectCount:    3,
		ServiceNames:        []string{"ProductService", "OrderService"},
		Services: []ServiceSummary{
			{Name: "ProductService", EntityNames: []string{"Product"}, ValueObjectNames: []string{"ProductName"}, ValueObjects: []ValueObjectSummary{{Name: "ProductName", Type: "string", RulesLabel: "no rules"}}, Entities: []EntitySummary{{Name: "Product", Fields: []FieldSummary{{Name: "Id", Type: "Guid"}, {Name: "Name", Type: "string"}}}}},
			{Name: "OrderService", EntityNames: []string{"Order", "OrderLine"}, ValueObjectNames: []string{"OrderNumber", "Money"}, ValueObjects: []ValueObjectSummary{{Name: "OrderNumber", Type: "string", RulesLabel: "no rules"}, {Name: "Money", Type: "decimal", RulesLabel: "no rules"}}, Entities: []EntitySummary{{Name: "Order", Fields: []FieldSummary{}}, {Name: "OrderLine", Fields: []FieldSummary{}}}},
		},
	}
	if !reflect.DeepEqual(plan.Config, expectedSummary) {
		t.Fatalf("expected config summary %#v, got %#v", expectedSummary, plan.Config)
	}
	expectedReadiness := ReadinessSummary{ProjectPresent: true, ServiceCount: 2, EntityCount: 3, FieldCount: 2, ValueObjectCount: 3, OutputForceRequired: true, Hints: []string{"Review output replacement; --force is required to write."}}
	if !reflect.DeepEqual(plan.Readiness, expectedReadiness) {
		t.Fatalf("expected readiness summary %#v, got %#v", expectedReadiness, plan.Readiness)
	}
	expectedFiles := []PlannedFile{{Path: "README.md", Action: "replace"}, {Path: "src/ProductService/Product.cs", Action: "unchanged"}}
	if !reflect.DeepEqual(plan.Files, expectedFiles) {
		t.Fatalf("expected planned files %#v, got %#v", expectedFiles, plan.Files)
	}
}

func TestServicePlanGenerationDerivesStarterReadinessHints(t *testing.T) {
	service := NewService(Ports{
		ConfigLoader:    &fakeConfigLoader{cfg: starterConfig()},
		ConfigValidator: specValidator{},
		Generator:       &fakeGenerator{files: []GeneratedFile{{Path: "README.md", Content: []byte("readme")}}},
		OutputPlanner:   fakeOutputPlanner{plan: OutputPlan{OutputDir: "/abs/generated", Action: "create", Files: []OutputPlannedFile{{Path: "README.md", Action: "create"}}}},
	})

	plan, err := service.PlanGeneration(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated"})

	if err != nil {
		t.Fatalf("expected starter plan, got %v", err)
	}
	expected := ReadinessSummary{
		ProjectPresent:      true,
		ServiceCount:        1,
		EntityCount:         1,
		FieldCount:          2,
		ValueObjectCount:    0,
		OutputForceRequired: false,
		Hints: []string{
			"Rename the starter project.",
			"Rename the starter service.",
			"Rename the starter entity and add domain fields.",
			"Review the output preview before generating.",
		},
	}
	if !reflect.DeepEqual(plan.Readiness, expected) {
		t.Fatalf("expected starter readiness %#v, got %#v", expected, plan.Readiness)
	}
}

func TestServiceCreateStarterConfigSavesValidMinimalConfig(t *testing.T) {
	saver := &fakeConfigSaver{}
	service := NewService(Ports{ConfigSaver: saver, ConfigValidator: specValidator{}})
	path := filepath.Join(t.TempDir(), "microgen.json")

	summary, err := service.CreateStarterConfig(path)

	if err != nil {
		t.Fatalf("expected starter config creation, got %v", err)
	}
	if !saver.called || saver.path != path {
		t.Fatalf("expected starter config to be saved at %s, called=%v path=%q", path, saver.called, saver.path)
	}
	if saver.cfg.SchemaVersion != spec.ConfigSchemaVersion || saver.cfg.Generation.TargetFramework != spec.DefaultTargetFramework || saver.cfg.Generation.SolutionFormat != "sln" {
		t.Fatalf("expected schema and generation defaults, got %#v", saver.cfg)
	}
	if saver.cfg.Solution.Name != "StarterPlatform" || len(saver.cfg.Services) != 1 || len(saver.cfg.Services[0].Entities) != 1 {
		t.Fatalf("expected starter solution/service/entity, got %#v", saver.cfg)
	}
	fields := saver.cfg.Services[0].Entities[0].Fields
	if len(fields) != 2 || fields[0].Name != "Id" || fields[0].Type != "Guid" || fields[1].Name != "Name" || fields[1].Type != "string" {
		t.Fatalf("expected starter entity with Guid Id and string Name, got %#v", fields)
	}
	if summary.SolutionName != "StarterPlatform" || summary.TargetFramework != spec.DefaultTargetFramework || summary.ServiceCount != 1 || summary.EntityCount != 1 {
		t.Fatalf("expected starter summary, got %#v", summary)
	}
}

func TestServiceCreateStarterConfigRefusesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "microgen.json")
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write existing config: %v", err)
	}
	saver := &fakeConfigSaver{}
	service := NewService(Ports{ConfigSaver: saver, ConfigValidator: specValidator{}})

	_, err := service.CreateStarterConfig(path)

	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected existing-file refusal, got %v", err)
	}
	if saver.called {
		t.Fatal("expected existing starter config not to be overwritten")
	}
}

func TestServiceGenerateWritesGeneratedFilesThroughOutputPort(t *testing.T) {
	files := []GeneratedFile{{Path: "README.md", Content: []byte("readme")}}
	writer := &fakeOutputWriter{result: WriteResult{OutputDir: "/published/generated", OutputAction: "create", Warning: "cleanup warning"}}
	service := NewService(Ports{
		ConfigLoader:    &fakeConfigLoader{cfg: validConfig()},
		ConfigValidator: &fakeConfigValidator{},
		Generator:       &fakeGenerator{files: files},
		OutputWriter:    writer,
		OutputPlanner:   fakeOutputPlanner{plan: OutputPlan{OutputDir: "/planned/generated", Action: "create", Files: []OutputPlannedFile{{Path: "README.md", Action: "create"}}}},
	})

	result, err := service.Generate(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated", Force: true})

	if err != nil {
		t.Fatalf("expected generate success, got %v", err)
	}
	if writer.outputDir != "generated" || !writer.force {
		t.Fatalf("expected writer outputDir generated with force, got outputDir=%q force=%v", writer.outputDir, writer.force)
	}
	if !reflect.DeepEqual(writer.files, files) {
		t.Fatalf("expected writer to receive generated files")
	}
	if result.OutputDir != "/published/generated" || result.Warning != "cleanup warning" {
		t.Fatalf("unexpected write result: %#v", result)
	}
	if result.Plan.FileCount != 1 || result.Plan.OutputDir != "/published/generated" || result.Plan.OutputAction != "create" {
		t.Fatalf("unexpected plan in generate result: %#v", result.Plan)
	}
}

func TestServiceGenerateReturnsPublishTimeOutputStatus(t *testing.T) {
	files := []GeneratedFile{{Path: "README.md", Content: []byte("readme")}}
	writer := &fakeOutputWriter{result: WriteResult{
		OutputDir:     "/published/generated",
		OutputAction:  "replace",
		ForceRequired: true,
		ForceUsed:     true,
	}}
	service := NewService(Ports{
		ConfigLoader:    &fakeConfigLoader{cfg: validConfig()},
		ConfigValidator: &fakeConfigValidator{},
		Generator:       &fakeGenerator{files: files},
		OutputWriter:    writer,
		OutputPlanner: fakeOutputPlanner{plan: OutputPlan{
			OutputDir:     "/planned/generated",
			Action:        "create",
			ForceRequired: false,
			ForceUsed:     false,
			Files:         []OutputPlannedFile{{Path: "README.md", Action: "create"}},
		}},
	})

	result, err := service.Generate(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated", Force: true})

	if err != nil {
		t.Fatalf("expected generate success, got %v", err)
	}
	if result.Plan.OutputDir != "/published/generated" || result.Plan.OutputAction != "replace" || !result.Plan.ForceRequired || !result.Plan.ForceUsed {
		t.Fatalf("expected publish-time output status, got %#v", result.Plan)
	}
	expectedFiles := []PlannedFile{{Path: "README.md", Action: "create"}}
	if result.Plan.FileCount != 1 || !reflect.DeepEqual(result.Plan.Files, expectedFiles) {
		t.Fatalf("expected preflight file plan to be preserved, got %#v", result.Plan)
	}
}

func TestServicePlanGenerationStopsAtValidationError(t *testing.T) {
	expectedErr := errors.New("invalid config")
	gen := &fakeGenerator{}
	service := NewService(Ports{
		ConfigLoader:    &fakeConfigLoader{cfg: validConfig()},
		ConfigValidator: &fakeConfigValidator{err: expectedErr},
		Generator:       gen,
		OutputWriter:    &fakeOutputWriter{},
	})

	_, err := service.PlanGeneration(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated"})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if gen.called {
		t.Fatal("expected validation error to stop before generation")
	}
}

func TestServiceUpdateSolutionSettingsRejectsInvalidSettingsWithoutSaving(t *testing.T) {
	tests := []struct {
		name     string
		settings SolutionSettings
		wantErr  string
	}{
		{
			name:     "invalid solution name",
			settings: SolutionSettings{SolutionName: "1Bad", SolutionDescription: "Updated", TargetFramework: "net9.0"},
			wantErr:  "solution.name must be a valid C# identifier",
		},
		{name: "invalid target framework", settings: SolutionSettings{SolutionName: "CommercePlatform", SolutionDescription: "Updated", TargetFramework: "latest"}, wantErr: "generation.targetFramework must be netN.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saver := &fakeConfigSaver{}
			gen := &fakeGenerator{}
			service := NewService(Ports{
				ConfigLoader:    &fakeConfigLoader{cfg: validPersistableConfig()},
				ConfigSaver:     saver,
				ConfigValidator: specValidator{},
				Generator:       gen,
				OutputPlanner:   fakeOutputPlanner{},
			})

			_, err := service.UpdateSolutionSettings(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated"}, tt.settings)

			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
			if saver.called {
				t.Fatal("expected invalid settings not to be saved")
			}
			if gen.called {
				t.Fatal("expected invalid settings not to regenerate plan")
			}
		})
	}
}

func TestServiceUpdateSolutionSettingsSavesValidSettingsAndReturnsPlan(t *testing.T) {
	saver := &fakeConfigSaver{}
	service := NewService(Ports{
		ConfigLoader:    &fakeConfigLoader{cfg: validPersistableConfig()},
		ConfigSaver:     saver,
		ConfigValidator: specValidator{},
		Generator:       &fakeGenerator{files: []GeneratedFile{{Path: "README.md", Content: []byte("readme")}}},
		OutputPlanner:   fakeOutputPlanner{plan: OutputPlan{OutputDir: "/planned/generated", Action: "create", Files: []OutputPlannedFile{{Path: "README.md", Action: "create"}}}},
	})
	settings := SolutionSettings{SolutionName: "CatalogPlatform", SolutionDescription: "Catalog management.", TargetFramework: "net9.0"}

	result, err := service.UpdateSolutionSettings(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated", Force: true}, settings)

	if err != nil {
		t.Fatalf("expected update success, got %v", err)
	}
	if !saver.called || saver.path != "microgen.json" {
		t.Fatalf("expected config save at microgen.json, got called=%v path=%q", saver.called, saver.path)
	}
	if saver.cfg.Solution.Name != settings.SolutionName || saver.cfg.Solution.Description != settings.SolutionDescription || saver.cfg.Generation.TargetFramework != "net9.0" {
		t.Fatalf("expected saved solution settings, got %#v", saver.cfg)
	}
	if saver.cfg.Generation.SolutionFormat != "sln" {
		t.Fatalf("expected net9.0 to persist sln solution format, got %q", saver.cfg.Generation.SolutionFormat)
	}
	if len(saver.cfg.Services) != 1 || len(saver.cfg.Services[0].Entities) != 1 || len(saver.cfg.Services[0].ValueObjects) != 1 {
		t.Fatalf("expected service/entity/value-object data to be preserved, got %#v", saver.cfg.Services)
	}
	if saver.cfg.SchemaVersion != spec.ConfigSchemaVersion {
		t.Fatalf("expected schemaVersion %d, got %d", spec.ConfigSchemaVersion, saver.cfg.SchemaVersion)
	}
	if !result.Saved || result.PlanError != nil {
		t.Fatalf("expected saved result without plan error, got %#v", result)
	}
	if result.Config.SolutionName != settings.SolutionName || result.Config.SolutionDescription != settings.SolutionDescription || result.Config.TargetFramework != "net9.0" {
		t.Fatalf("expected saved config summary, got %#v", result.Config)
	}
	if result.Config.SolutionFormat != "sln" {
		t.Fatalf("expected saved config summary solution format sln, got %#v", result.Config)
	}
	if result.Plan.Config.SolutionName != settings.SolutionName || result.Plan.Config.TargetFramework != "net9.0" || result.Plan.FileCount != 1 || result.Plan.OutputDir != "/planned/generated" {
		t.Fatalf("expected refreshed plan from saved settings, got %#v", result.Plan)
	}
}

func TestServiceUpdateSolutionSettingsNormalizesManualTargetFrameworkAndDefaultsSolutionFormat(t *testing.T) {
	saver := &fakeConfigSaver{}
	service := NewService(Ports{
		ConfigLoader:    &fakeConfigLoader{cfg: validPersistableConfig()},
		ConfigSaver:     saver,
		ConfigValidator: specValidator{},
		Generator:       &fakeGenerator{files: []GeneratedFile{{Path: "README.md", Content: []byte("readme")}}},
		OutputPlanner:   fakeOutputPlanner{plan: OutputPlan{OutputDir: "/planned/generated", Action: "create", Files: []OutputPlannedFile{{Path: "README.md", Action: "create"}}}},
	})

	result, err := service.UpdateSolutionSettings(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated"}, SolutionSettings{SolutionName: "CatalogPlatform", SolutionDescription: "Catalog management.", TargetFramework: "10"})

	if err != nil {
		t.Fatalf("expected update success, got %v", err)
	}
	if saver.cfg.Generation.TargetFramework != "net10.0" || saver.cfg.Generation.SolutionFormat != "slnx" {
		t.Fatalf("expected normalized net10.0/slnx save, got %#v", saver.cfg.Generation)
	}
	if result.Config.TargetFramework != "net10.0" || result.Config.SolutionFormat != "slnx" {
		t.Fatalf("expected net10.0/slnx summary, got %#v", result.Config)
	}
}

func TestTargetFrameworkSuggestionsParseInstalledSDKMajorsNewestFirst(t *testing.T) {
	got := targetFrameworksFromSDKList("8.0.404 [/sdk]\n10.0.110 [/sdk]\nnot-a-version [/sdk]\n10.0.111 [/sdk]\n")
	want := []string{"net10.0", "net8.0"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected suggestions %#v, got %#v", want, got)
	}
}

func TestServiceUpdateSolutionSettingsReportsPostSavePlanError(t *testing.T) {
	saver := &fakeConfigSaver{}
	planErr := errors.New("plan failed")
	service := NewService(Ports{
		ConfigLoader:    &fakeConfigLoader{cfg: validPersistableConfig()},
		ConfigSaver:     saver,
		ConfigValidator: specValidator{},
		Generator:       &fakeGenerator{err: planErr},
		OutputPlanner:   fakeOutputPlanner{},
	})
	settings := SolutionSettings{SolutionName: "CatalogPlatform", SolutionDescription: "Catalog management.", TargetFramework: "net9.0"}

	result, err := service.UpdateSolutionSettings(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated"}, settings)

	if err != nil {
		t.Fatalf("expected save success with plan error result, got %v", err)
	}
	if !saver.called || !result.Saved || !errors.Is(result.PlanError, planErr) {
		t.Fatalf("expected saved result with plan error, got saver.called=%v result=%#v", saver.called, result)
	}
	if result.Config.SolutionName != settings.SolutionName || result.Config.SolutionDescription != settings.SolutionDescription || result.Config.TargetFramework != "net9.0" {
		t.Fatalf("expected saved config summary despite plan error, got %#v", result.Config)
	}
	if result.Config.ServiceCount != 1 || result.Config.EntityCount != 1 || result.Config.ValueObjectCount != 1 {
		t.Fatalf("expected preserved config summary counts, got %#v", result.Config)
	}
}

func TestServiceUpdateServiceSettingsAddsRenamesAndDeletesServices(t *testing.T) {
	saver := &fakeConfigSaver{}
	service := NewService(Ports{
		ConfigLoader:    &fakeConfigLoader{cfg: validConfig()},
		ConfigSaver:     saver,
		ConfigValidator: specValidator{},
		Generator:       &fakeGenerator{files: []GeneratedFile{{Path: "README.md", Content: []byte("readme")}}},
		OutputPlanner:   fakeOutputPlanner{plan: OutputPlan{OutputDir: "/planned/generated", Action: "create", Files: []OutputPlannedFile{{Path: "README.md", Action: "create"}}}},
	})

	result, err := service.UpdateServiceSettings(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated"}, ServiceSettings{Services: []ServiceNameSetting{{OriginalName: "ProductService", Name: "CatalogService"}, {Name: "BillingService"}}})

	if err != nil {
		t.Fatalf("expected update success, got %v", err)
	}
	if !saver.called || saver.path != "microgen.json" {
		t.Fatalf("expected config save at microgen.json, got called=%v path=%q", saver.called, saver.path)
	}
	if got := serviceNames(saver.cfg.Services); !reflect.DeepEqual(got, []string{"CatalogService", "BillingService"}) {
		t.Fatalf("expected renamed/deleted/added services, got %#v", got)
	}
	if len(saver.cfg.Services[0].Entities) != 1 || saver.cfg.Services[0].Entities[0].Name != "Product" || len(saver.cfg.Services[0].ValueObjects) != 1 {
		t.Fatalf("expected renamed service to preserve existing entities and value objects, got %#v", saver.cfg.Services[0])
	}
	billing := saver.cfg.Services[1]
	if len(billing.Entities) != 1 || billing.Entities[0].Name != "Billing" {
		t.Fatalf("expected new service default Billing entity, got %#v", billing)
	}
	fields := billing.Entities[0].Fields
	if len(fields) != 1 || fields[0].Name != "Id" || fields[0].Type != "Guid" {
		t.Fatalf("expected new service default Guid Id field, got %#v", fields)
	}
	if !result.Saved || result.PlanError != nil || result.Plan.FileCount != 1 || result.Config.ServiceCount != 2 || result.Config.EntityCount != 2 {
		t.Fatalf("expected saved result with refreshed plan summary, got %#v", result)
	}
}

func TestServiceUpdateServiceSettingsRejectsDeletingLastServiceWithoutSaving(t *testing.T) {
	saver := &fakeConfigSaver{}
	gen := &fakeGenerator{}
	service := NewService(Ports{
		ConfigLoader:    &fakeConfigLoader{cfg: validPersistableConfig()},
		ConfigSaver:     saver,
		ConfigValidator: specValidator{},
		Generator:       gen,
		OutputPlanner:   fakeOutputPlanner{},
	})

	_, err := service.UpdateServiceSettings(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated"}, ServiceSettings{})

	if err == nil || !strings.Contains(err.Error(), "services must contain at least 1 item") {
		t.Fatalf("expected service count validation error, got %v", err)
	}
	if saver.called {
		t.Fatal("expected invalid service list not to be saved")
	}
	if gen.called {
		t.Fatal("expected invalid service list not to regenerate plan")
	}
}

func TestServiceUpdateServiceSettingsRejectsInvalidServiceNameWithoutSaving(t *testing.T) {
	saver := &fakeConfigSaver{}
	gen := &fakeGenerator{}
	service := NewService(Ports{
		ConfigLoader:    &fakeConfigLoader{cfg: validPersistableConfig()},
		ConfigSaver:     saver,
		ConfigValidator: specValidator{},
		Generator:       gen,
		OutputPlanner:   fakeOutputPlanner{},
	})

	_, err := service.UpdateServiceSettings(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated"}, ServiceSettings{ServiceNames: []string{"1BadService"}})

	if err == nil || !strings.Contains(err.Error(), "services[0].name must be a valid C# identifier") {
		t.Fatalf("expected service identifier validation error, got %v", err)
	}
	if saver.called {
		t.Fatal("expected invalid service name not to be saved")
	}
	if gen.called {
		t.Fatal("expected invalid service name not to regenerate plan")
	}
}

func TestServiceUpdateServiceSettingsReportsPostSavePlanError(t *testing.T) {
	saver := &fakeConfigSaver{}
	planErr := errors.New("plan failed")
	service := NewService(Ports{
		ConfigLoader:    &fakeConfigLoader{cfg: validPersistableConfig()},
		ConfigSaver:     saver,
		ConfigValidator: specValidator{},
		Generator:       &fakeGenerator{err: planErr},
		OutputPlanner:   fakeOutputPlanner{},
	})

	result, err := service.UpdateServiceSettings(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated"}, ServiceSettings{Services: []ServiceNameSetting{{OriginalName: "ProductService", Name: "CatalogService"}, {Name: "OrderService"}}})

	if err != nil {
		t.Fatalf("expected save success with plan error result, got %v", err)
	}
	if !saver.called || !result.Saved || !errors.Is(result.PlanError, planErr) {
		t.Fatalf("expected saved result with plan error, got saver.called=%v result=%#v", saver.called, result)
	}
	if result.Config.ServiceCount != 2 || result.Config.EntityCount != 2 || !reflect.DeepEqual(result.Config.ServiceNames, []string{"CatalogService", "OrderService"}) {
		t.Fatalf("expected saved service summary despite plan error, got %#v", result.Config)
	}
}

func TestServiceUpdateEntitySettingsAddsRenamesAndDeletesEntities(t *testing.T) {
	saver := &fakeConfigSaver{}
	service := NewService(Ports{
		ConfigLoader:    &fakeConfigLoader{cfg: validConfigWithEntities("Product", "Legacy")},
		ConfigSaver:     saver,
		ConfigValidator: specValidator{},
		Generator:       &fakeGenerator{files: []GeneratedFile{{Path: "README.md", Content: []byte("readme")}}},
		OutputPlanner:   fakeOutputPlanner{plan: OutputPlan{OutputDir: "/planned/generated", Action: "create", Files: []OutputPlannedFile{{Path: "README.md", Action: "create"}}}},
	})

	result, err := service.UpdateEntitySettings(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated"}, EntitySettings{ServiceName: "ProductService", Entities: []EntityNameSetting{{OriginalName: "Product", Name: "Catalog"}, {Name: "Inventory"}}})

	if err != nil {
		t.Fatalf("expected update success, got %v", err)
	}
	if !saver.called || saver.path != "microgen.json" {
		t.Fatalf("expected config save at microgen.json, got called=%v path=%q", saver.called, saver.path)
	}
	entities := saver.cfg.Services[0].Entities
	if len(entities) != 2 || entities[0].Name != "Catalog" || entities[1].Name != "Inventory" {
		t.Fatalf("expected renamed/deleted/added entities, got %#v", entities)
	}
	if len(entities[0].Fields) != 2 || entities[0].Fields[1].Name != "Name" || entities[0].Fields[1].Type != "ProductName" {
		t.Fatalf("expected renamed entity to preserve existing fields, got %#v", entities[0].Fields)
	}
	fields := entities[1].Fields
	if len(fields) != 1 || fields[0].Name != "Id" || fields[0].Type != "Guid" {
		t.Fatalf("expected new entity default Guid Id field, got %#v", fields)
	}
	if !result.Saved || result.PlanError != nil || result.Plan.FileCount != 1 || result.Config.EntityCount != 2 {
		t.Fatalf("expected saved result with refreshed plan summary, got %#v", result)
	}
	if len(result.Config.Services) != 1 || !reflect.DeepEqual(result.Config.Services[0].EntityNames, []string{"Catalog", "Inventory"}) {
		t.Fatalf("expected service entity summary, got %#v", result.Config.Services)
	}
}

func TestServiceUpdateEntitySettingsRejectsInvalidEntityNameWithoutSaving(t *testing.T) {
	saver := &fakeConfigSaver{}
	gen := &fakeGenerator{}
	service := NewService(Ports{
		ConfigLoader:    &fakeConfigLoader{cfg: validPersistableConfig()},
		ConfigSaver:     saver,
		ConfigValidator: specValidator{},
		Generator:       gen,
		OutputPlanner:   fakeOutputPlanner{},
	})

	_, err := service.UpdateEntitySettings(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated"}, EntitySettings{ServiceName: "ProductService", Entities: []EntityNameSetting{{OriginalName: "Product", Name: "1Bad"}}})

	if err == nil || !strings.Contains(err.Error(), "services[0].entities[0].name must be a valid C# identifier") {
		t.Fatalf("expected entity identifier validation error, got %v", err)
	}
	if saver.called {
		t.Fatal("expected invalid entity name not to be saved")
	}
	if gen.called {
		t.Fatal("expected invalid entity name not to regenerate plan")
	}
}

func TestServiceUpdateEntitySettingsRejectsDeletingLastEntityWithoutSaving(t *testing.T) {
	saver := &fakeConfigSaver{}
	gen := &fakeGenerator{}
	service := NewService(Ports{
		ConfigLoader:    &fakeConfigLoader{cfg: validPersistableConfig()},
		ConfigSaver:     saver,
		ConfigValidator: specValidator{},
		Generator:       gen,
		OutputPlanner:   fakeOutputPlanner{},
	})

	_, err := service.UpdateEntitySettings(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated"}, EntitySettings{ServiceName: "ProductService"})

	if err == nil || !strings.Contains(err.Error(), "services must keep at least one entity") {
		t.Fatalf("expected last entity deletion error, got %v", err)
	}
	if saver.called {
		t.Fatal("expected invalid entity list not to be saved")
	}
	if gen.called {
		t.Fatal("expected invalid entity list not to regenerate plan")
	}
}

func TestServiceUpdateEntitySettingsReportsPostSavePlanError(t *testing.T) {
	saver := &fakeConfigSaver{}
	planErr := errors.New("plan failed")
	service := NewService(Ports{
		ConfigLoader:    &fakeConfigLoader{cfg: validPersistableConfig()},
		ConfigSaver:     saver,
		ConfigValidator: specValidator{},
		Generator:       &fakeGenerator{err: planErr},
		OutputPlanner:   fakeOutputPlanner{},
	})

	result, err := service.UpdateEntitySettings(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated"}, EntitySettings{ServiceName: "ProductService", Entities: []EntityNameSetting{{OriginalName: "Product", Name: "Catalog"}}})

	if err != nil {
		t.Fatalf("expected save success with plan error result, got %v", err)
	}
	if !saver.called || !result.Saved || !errors.Is(result.PlanError, planErr) {
		t.Fatalf("expected saved result with plan error, got saver.called=%v result=%#v", saver.called, result)
	}
	if result.Config.EntityCount != 1 || len(result.Config.Services) != 1 || !reflect.DeepEqual(result.Config.Services[0].EntityNames, []string{"Catalog"}) {
		t.Fatalf("expected saved entity summary despite plan error, got %#v", result.Config)
	}
}

func TestServiceUpdateFieldSettingsAddsRenamesDeletesAndPreservesTypes(t *testing.T) {
	saver := &fakeConfigSaver{}
	service := NewService(Ports{
		ConfigLoader:    &fakeConfigLoader{cfg: validPersistableConfig()},
		ConfigSaver:     saver,
		ConfigValidator: specValidator{},
		Generator:       &fakeGenerator{files: []GeneratedFile{{Path: "README.md", Content: []byte("readme")}}},
		OutputPlanner:   fakeOutputPlanner{plan: OutputPlan{OutputDir: "/planned/generated", Action: "create", Files: []OutputPlannedFile{{Path: "README.md", Action: "create"}}}},
	})

	result, err := service.UpdateFieldSettings(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated"}, FieldSettings{ServiceName: "ProductService", EntityName: "Product", Fields: []FieldSetting{{OriginalName: "Id", Name: "Id", Type: "Guid"}, {OriginalName: "Name", Name: "Title", Type: "ProductName"}, {Name: "Sku", Type: "string"}}})

	if err != nil {
		t.Fatalf("expected update success, got %v", err)
	}
	if !saver.called || saver.path != "microgen.json" {
		t.Fatalf("expected config save at microgen.json, got called=%v path=%q", saver.called, saver.path)
	}
	fields := saver.cfg.Services[0].Entities[0].Fields
	wantFields := []spec.Field{{Name: "Id", Type: "Guid"}, {Name: "Title", Type: "ProductName"}, {Name: "Sku", Type: "string"}}
	if !reflect.DeepEqual(fields, wantFields) {
		t.Fatalf("expected renamed/deleted/added fields with type preservation, got %#v", fields)
	}
	if !result.Saved || result.PlanError != nil || result.Plan.FileCount != 1 {
		t.Fatalf("expected saved result with refreshed plan summary, got %#v", result)
	}
	gotSummaryFields := result.Config.Services[0].Entities[0].Fields
	wantSummaryFields := []FieldSummary{{Name: "Id", Type: "Guid"}, {Name: "Title", Type: "ProductName"}, {Name: "Sku", Type: "string"}}
	if !reflect.DeepEqual(gotSummaryFields, wantSummaryFields) {
		t.Fatalf("expected field summary, got %#v", gotSummaryFields)
	}
}

func TestServiceUpdateFieldSettingsRejectsInvalidFieldsWithoutSaving(t *testing.T) {
	tests := []struct {
		name    string
		fields  []FieldSetting
		wantErr string
	}{
		{name: "blank name", fields: []FieldSetting{{OriginalName: "Id", Name: "Id", Type: "Guid"}, {OriginalName: "Name", Name: "", Type: "ProductName"}}, wantErr: "services[0].entities[0].fields[1].name is required"},
		{name: "duplicate name", fields: []FieldSetting{{OriginalName: "Id", Name: "Id", Type: "Guid"}, {OriginalName: "Name", Name: "Id", Type: "Guid"}}, wantErr: "duplicate field in entity Product name"},
		{name: "invalid type", fields: []FieldSetting{{OriginalName: "Id", Name: "Id", Type: "Guid"}, {OriginalName: "Name", Name: "Name", Type: "unknown"}}, wantErr: "services[0].entities[0].fields[1].type must be one of"},
		{name: "missing id", fields: []FieldSetting{{OriginalName: "Name", Name: "Name", Type: "ProductName"}}, wantErr: "services[0].entities[0].fields must contain exactly one Id field of type Guid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saver := &fakeConfigSaver{}
			gen := &fakeGenerator{}
			service := NewService(Ports{
				ConfigLoader:    &fakeConfigLoader{cfg: validPersistableConfig()},
				ConfigSaver:     saver,
				ConfigValidator: specValidator{},
				Generator:       gen,
				OutputPlanner:   fakeOutputPlanner{},
			})

			_, err := service.UpdateFieldSettings(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated"}, FieldSettings{ServiceName: "ProductService", EntityName: "Product", Fields: tt.fields})

			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
			if saver.called {
				t.Fatal("expected invalid fields not to be saved")
			}
			if gen.called {
				t.Fatal("expected invalid fields not to regenerate plan")
			}
		})
	}
}

func TestServiceUpdateFieldSettingsRejectsDeletingLastFieldWithoutSaving(t *testing.T) {
	saver := &fakeConfigSaver{}
	gen := &fakeGenerator{}
	service := NewService(Ports{
		ConfigLoader:    &fakeConfigLoader{cfg: validPersistableConfig()},
		ConfigSaver:     saver,
		ConfigValidator: specValidator{},
		Generator:       gen,
		OutputPlanner:   fakeOutputPlanner{},
	})

	_, err := service.UpdateFieldSettings(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated"}, FieldSettings{ServiceName: "ProductService", EntityName: "Product"})

	if err == nil || !strings.Contains(err.Error(), "entities must keep at least one field") {
		t.Fatalf("expected last field deletion error, got %v", err)
	}
	if saver.called {
		t.Fatal("expected invalid field list not to be saved")
	}
	if gen.called {
		t.Fatal("expected invalid field list not to regenerate plan")
	}
}

func TestServiceUpdateFieldSettingsReportsPostSavePlanError(t *testing.T) {
	saver := &fakeConfigSaver{}
	planErr := errors.New("plan failed")
	service := NewService(Ports{
		ConfigLoader:    &fakeConfigLoader{cfg: validPersistableConfig()},
		ConfigSaver:     saver,
		ConfigValidator: specValidator{},
		Generator:       &fakeGenerator{err: planErr},
		OutputPlanner:   fakeOutputPlanner{},
	})

	result, err := service.UpdateFieldSettings(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated"}, FieldSettings{ServiceName: "ProductService", EntityName: "Product", Fields: []FieldSetting{{OriginalName: "Id", Name: "Id", Type: "Guid"}, {OriginalName: "Name", Name: "Title", Type: "ProductName"}}})

	if err != nil {
		t.Fatalf("expected save success with plan error result, got %v", err)
	}
	if !saver.called || !result.Saved || !errors.Is(result.PlanError, planErr) {
		t.Fatalf("expected saved result with plan error, got saver.called=%v result=%#v", saver.called, result)
	}
	if got := result.Config.Services[0].Entities[0].Fields[1]; got.Name != "Title" || got.Type != "ProductName" {
		t.Fatalf("expected saved field summary despite plan error, got %#v", result.Config)
	}
}

func TestServiceUpdateValueObjectSettingsAddsRenamesDeletesAndPreservesDetails(t *testing.T) {
	saver := &fakeConfigSaver{}
	service := NewService(Ports{
		ConfigLoader:    &fakeConfigLoader{cfg: validConfigWithValueObjectsForEditing()},
		ConfigSaver:     saver,
		ConfigValidator: specValidator{},
		Generator:       &fakeGenerator{files: []GeneratedFile{{Path: "README.md", Content: []byte("readme")}}},
		OutputPlanner:   fakeOutputPlanner{plan: OutputPlan{OutputDir: "/planned/generated", Action: "create", Files: []OutputPlannedFile{{Path: "README.md", Action: "create"}}}},
	})

	result, err := service.UpdateValueObjectSettings(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated"}, ValueObjectSettings{ServiceName: "ProductService", ValueObjects: []ValueObjectNameSetting{{OriginalName: "ProductCode", Name: "CatalogCode"}, {Name: "Sku"}}})

	if err != nil {
		t.Fatalf("expected update success, got %v", err)
	}
	if !saver.called || saver.path != "microgen.json" {
		t.Fatalf("expected config save at microgen.json, got called=%v path=%q", saver.called, saver.path)
	}
	valueObjects := saver.cfg.Services[0].ValueObjects
	if len(valueObjects) != 2 || valueObjects[0].Name != "CatalogCode" || valueObjects[1].Name != "Sku" {
		t.Fatalf("expected renamed/deleted/added value objects, got %#v", valueObjects)
	}
	if valueObjects[0].Type != "string" || valueObjects[0].Validations.MinLength == nil || *valueObjects[0].Validations.MinLength != 2 || valueObjects[0].Validations.MaxLength == nil || *valueObjects[0].Validations.MaxLength != 20 {
		t.Fatalf("expected renamed value object to preserve details, got %#v", valueObjects[0])
	}
	if valueObjects[1].Type != "string" || valueObjects[1].Validations.Required == nil || !*valueObjects[1].Validations.Required || valueObjects[1].Validations.MinLength == nil || *valueObjects[1].Validations.MinLength != 1 || valueObjects[1].Validations.MaxLength == nil || *valueObjects[1].Validations.MaxLength != 100 || valueObjects[1].Validations.ValidExample == nil || *valueObjects[1].Validations.ValidExample != "Sample" {
		t.Fatalf("expected new value object to use generator-valid string defaults, got %#v", valueObjects[1])
	}
	if !result.Saved || result.PlanError != nil || result.Plan.FileCount != 1 {
		t.Fatalf("expected saved result with refreshed plan summary, got %#v", result)
	}
	if !reflect.DeepEqual(result.Config.Services[0].ValueObjectNames, []string{"CatalogCode", "Sku"}) {
		t.Fatalf("expected value object summary names, got %#v", result.Config.Services[0])
	}
}

func TestServiceUpdateValueObjectSettingsUpdatesStringRules(t *testing.T) {
	required := true
	minLength := 3
	maxLength := 40
	pattern := "^[A-Z]+$"
	validExample := "SKU"
	invalidExample := "bad"
	saver := &fakeConfigSaver{}
	service := NewService(Ports{ConfigLoader: &fakeConfigLoader{cfg: validConfigWithValueObjectsForEditing()}, ConfigSaver: saver, ConfigValidator: specValidator{}, Generator: &fakeGenerator{}, OutputPlanner: fakeOutputPlanner{}})

	_, err := service.UpdateValueObjectSettings(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated"}, ValueObjectSettings{ServiceName: "ProductService", ValueObjects: []ValueObjectNameSetting{{OriginalName: "ProductCode", Name: "ProductCode", Type: "string", Validations: ValidationRuleSettings{Required: &required, MinLength: &minLength, MaxLength: &maxLength, Pattern: &pattern, ValidExample: &validExample, InvalidExample: &invalidExample}}, {OriginalName: "LegacyCode", Name: "LegacyCode"}}})

	if err != nil {
		t.Fatalf("expected string rule update success, got %v", err)
	}
	rules := saver.cfg.Services[0].ValueObjects[0].Validations
	if rules.Required == nil || !*rules.Required || rules.MinLength == nil || *rules.MinLength != 3 || rules.MaxLength == nil || *rules.MaxLength != 40 || rules.Pattern == nil || *rules.Pattern != pattern || rules.ValidExample == nil || *rules.ValidExample != validExample || rules.InvalidExample == nil || *rules.InvalidExample != invalidExample {
		t.Fatalf("expected updated string rules, got %#v", rules)
	}
}

func TestServiceUpdateValueObjectSettingsUpdatesNumericRules(t *testing.T) {
	minimum := "0"
	maximum := "999999.99"
	saver := &fakeConfigSaver{}
	service := NewService(Ports{ConfigLoader: &fakeConfigLoader{cfg: validConfigWithValueObjectsForEditing()}, ConfigSaver: saver, ConfigValidator: specValidator{}, Generator: &fakeGenerator{}, OutputPlanner: fakeOutputPlanner{}})

	_, err := service.UpdateValueObjectSettings(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated"}, ValueObjectSettings{ServiceName: "ProductService", ValueObjects: []ValueObjectNameSetting{{OriginalName: "ProductCode", Name: "ProductCode", Type: "decimal", Validations: ValidationRuleSettings{Minimum: &minimum, Maximum: &maximum}}, {OriginalName: "LegacyCode", Name: "LegacyCode"}}})

	if err != nil {
		t.Fatalf("expected numeric rule update success, got %v", err)
	}
	valueObject := saver.cfg.Services[0].ValueObjects[0]
	if valueObject.Type != "decimal" || valueObject.Validations.Minimum == nil || valueObject.Validations.Minimum.String() != "0" || valueObject.Validations.Maximum == nil || valueObject.Validations.Maximum.String() != "999999.99" {
		t.Fatalf("expected decimal rules, got %#v", valueObject)
	}
}

func TestServiceUpdateValueObjectSettingsTypeChangeClearsIncompatibleRules(t *testing.T) {
	minimum := "1"
	saver := &fakeConfigSaver{}
	service := NewService(Ports{ConfigLoader: &fakeConfigLoader{cfg: validConfigWithValueObjectsForEditing()}, ConfigSaver: saver, ConfigValidator: specValidator{}, Generator: &fakeGenerator{}, OutputPlanner: fakeOutputPlanner{}})

	_, err := service.UpdateValueObjectSettings(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated"}, ValueObjectSettings{ServiceName: "ProductService", ValueObjects: []ValueObjectNameSetting{{OriginalName: "ProductCode", Name: "ProductCode", Type: "int", Validations: ValidationRuleSettings{Minimum: &minimum}}, {OriginalName: "LegacyCode", Name: "LegacyCode"}}})

	if err != nil {
		t.Fatalf("expected type change success, got %v", err)
	}
	valueObject := saver.cfg.Services[0].ValueObjects[0]
	if valueObject.Type != "int" || valueObject.Validations.MinLength != nil || valueObject.Validations.MaxLength != nil || valueObject.Validations.ValidExample != nil || valueObject.Validations.Minimum == nil || valueObject.Validations.Minimum.String() != "1" {
		t.Fatalf("expected incompatible string rules cleared for int, got %#v", valueObject)
	}
}

func TestServiceUpdateValueObjectSettingsValidationFailureDoesNotSave(t *testing.T) {
	pattern := "["
	validExample := "ABC"
	invalidExample := "abc"
	saver := &fakeConfigSaver{}
	gen := &fakeGenerator{}
	service := NewService(Ports{ConfigLoader: &fakeConfigLoader{cfg: validConfigWithValueObjectsForEditing()}, ConfigSaver: saver, ConfigValidator: specValidator{}, Generator: gen, OutputPlanner: fakeOutputPlanner{}})

	_, err := service.UpdateValueObjectSettings(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated"}, ValueObjectSettings{ServiceName: "ProductService", ValueObjects: []ValueObjectNameSetting{{OriginalName: "ProductCode", Name: "ProductCode", Type: "string", Validations: ValidationRuleSettings{Pattern: &pattern, ValidExample: &validExample, InvalidExample: &invalidExample}}, {OriginalName: "LegacyCode", Name: "LegacyCode"}}})

	if err == nil || !strings.Contains(err.Error(), "pattern must compile as a regular expression") {
		t.Fatalf("expected validation failure, got %v", err)
	}
	if saver.called || gen.called {
		t.Fatalf("expected validation failure not to save or plan, saver=%v gen=%v", saver.called, gen.called)
	}
}

func TestServiceUpdateValueObjectSettingsRejectsUnsupportedEditorTypeWithoutSaving(t *testing.T) {
	saver := &fakeConfigSaver{}
	gen := &fakeGenerator{}
	service := NewService(Ports{ConfigLoader: &fakeConfigLoader{cfg: validConfigWithValueObjectsForEditing()}, ConfigSaver: saver, ConfigValidator: specValidator{}, Generator: gen, OutputPlanner: fakeOutputPlanner{}})

	_, err := service.UpdateValueObjectSettings(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated"}, ValueObjectSettings{ServiceName: "ProductService", ValueObjects: []ValueObjectNameSetting{{OriginalName: "ProductCode", Name: "ProductCode", Type: "double"}, {OriginalName: "LegacyCode", Name: "LegacyCode"}}})

	if err == nil || !strings.Contains(err.Error(), "type \"double\" is not editable in the basic rules editor") {
		t.Fatalf("expected unsupported editor type failure, got %v", err)
	}
	if saver.called || gen.called {
		t.Fatalf("expected unsupported editor type not to save or plan, saver=%v gen=%v", saver.called, gen.called)
	}
}

func TestServiceUpdateValueObjectSettingsReferencedObjectCanUpdateRulesButNotName(t *testing.T) {
	notEmpty := true
	saver := &fakeConfigSaver{}
	service := NewService(Ports{ConfigLoader: &fakeConfigLoader{cfg: validPersistableConfig()}, ConfigSaver: saver, ConfigValidator: specValidator{}, Generator: &fakeGenerator{}, OutputPlanner: fakeOutputPlanner{}})

	_, err := service.UpdateValueObjectSettings(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated"}, ValueObjectSettings{ServiceName: "ProductService", ValueObjects: []ValueObjectNameSetting{{OriginalName: "ProductName", Name: "ProductName", Type: "Guid", Validations: ValidationRuleSettings{NotEmpty: &notEmpty}}}})

	if err != nil {
		t.Fatalf("expected referenced value object rule update to preserve identity, got %v", err)
	}
	if got := saver.cfg.Services[0].ValueObjects[0]; got.Name != "ProductName" || got.Type != "Guid" || got.Validations.NotEmpty == nil || !*got.Validations.NotEmpty {
		t.Fatalf("expected referenced value object rules updated without renaming, got %#v", got)
	}
}

func TestServiceUpdateValueObjectSettingsSummarizesFieldReferences(t *testing.T) {
	service := NewService(Ports{
		ConfigLoader:    &fakeConfigLoader{cfg: validPersistableConfig()},
		ConfigValidator: specValidator{},
		Generator:       &fakeGenerator{},
		OutputPlanner:   fakeOutputPlanner{},
	})

	plan, err := service.PlanGeneration(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated"})

	if err != nil {
		t.Fatalf("expected plan success, got %v", err)
	}
	want := []ValueObjectReferenceSummary{{ValueObjectName: "ProductName", EntityName: "Product", FieldName: "Name"}}
	if !reflect.DeepEqual(plan.Config.Services[0].ValueObjectReferences, want) {
		t.Fatalf("expected value object field references, got %#v", plan.Config.Services[0].ValueObjectReferences)
	}
}

func TestServiceUpdateValueObjectSettingsRejectsInvalidValueObjectsWithoutSaving(t *testing.T) {
	tests := []struct {
		name         string
		valueObjects []ValueObjectNameSetting
		wantErr      string
	}{
		{name: "blank name", valueObjects: []ValueObjectNameSetting{{OriginalName: "ProductCode", Name: ""}}, wantErr: "services[0].valueObjects[0].name is required"},
		{name: "duplicate name", valueObjects: []ValueObjectNameSetting{{OriginalName: "ProductCode", Name: "ProductCode"}, {OriginalName: "LegacyCode", Name: "ProductCode"}}, wantErr: "duplicate value object in service ProductService name"},
		{name: "colliding entity name", valueObjects: []ValueObjectNameSetting{{OriginalName: "ProductCode", Name: "Product"}}, wantErr: "services[0].valueObjects[0].name must not collide with entity \"Product\""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saver := &fakeConfigSaver{}
			gen := &fakeGenerator{}
			service := NewService(Ports{
				ConfigLoader:    &fakeConfigLoader{cfg: validConfigWithValueObjectsForEditing()},
				ConfigSaver:     saver,
				ConfigValidator: specValidator{},
				Generator:       gen,
				OutputPlanner:   fakeOutputPlanner{},
			})

			_, err := service.UpdateValueObjectSettings(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated"}, ValueObjectSettings{ServiceName: "ProductService", ValueObjects: tt.valueObjects})

			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
			if saver.called {
				t.Fatal("expected invalid value objects not to be saved")
			}
			if gen.called {
				t.Fatal("expected invalid value objects not to regenerate plan")
			}
		})
	}
}

func TestServiceUpdateValueObjectSettingsRejectsReferencedRenameAndDeleteWithoutSaving(t *testing.T) {
	tests := []struct {
		name         string
		valueObjects []ValueObjectNameSetting
		wantErr      string
	}{
		{name: "referenced rename", valueObjects: []ValueObjectNameSetting{{OriginalName: "ProductName", Name: "CatalogName"}}, wantErr: "value object \"ProductName\" is referenced by entity fields and cannot be renamed"},
		{name: "referenced delete", valueObjects: nil, wantErr: "value object \"ProductName\" is referenced by entity fields and cannot be deleted"},
		{name: "referenced replacement", valueObjects: []ValueObjectNameSetting{{OriginalName: "LegacyName", Name: "ProductName"}}, wantErr: "value object \"ProductName\" is referenced by entity fields and cannot be renamed, deleted, or replaced"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saver := &fakeConfigSaver{}
			gen := &fakeGenerator{}
			service := NewService(Ports{
				ConfigLoader:    &fakeConfigLoader{cfg: validPersistableConfig()},
				ConfigSaver:     saver,
				ConfigValidator: specValidator{},
				Generator:       gen,
				OutputPlanner:   fakeOutputPlanner{},
			})

			_, err := service.UpdateValueObjectSettings(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated"}, ValueObjectSettings{ServiceName: "ProductService", ValueObjects: tt.valueObjects})

			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
			if saver.called {
				t.Fatal("expected unsafe value object change not to be saved")
			}
			if gen.called {
				t.Fatal("expected unsafe value object change not to regenerate plan")
			}
		})
	}
}

func TestServiceUpdateValueObjectSettingsRejectsReplacingReferencedValueObjectIdentityWithoutSaving(t *testing.T) {
	saver := &fakeConfigSaver{}
	gen := &fakeGenerator{}
	cfg := validPersistableConfig()
	cfg.Services[0].ValueObjects = append(cfg.Services[0].ValueObjects, spec.ValueObject{Name: "LegacyName", Type: "string"})
	service := NewService(Ports{
		ConfigLoader:    &fakeConfigLoader{cfg: cfg},
		ConfigSaver:     saver,
		ConfigValidator: specValidator{},
		Generator:       gen,
		OutputPlanner:   fakeOutputPlanner{},
	})

	_, err := service.UpdateValueObjectSettings(GenerateRequest{ConfigPath: "microgen.json", OutputDir: "generated"}, ValueObjectSettings{ServiceName: "ProductService", ValueObjects: []ValueObjectNameSetting{{OriginalName: "LegacyName", Name: "ProductName"}}})

	if err == nil || !strings.Contains(err.Error(), "value object \"ProductName\" is referenced by entity fields and cannot be renamed, deleted, or replaced") {
		t.Fatalf("expected referenced value object replacement error, got %v", err)
	}
	if saver.called {
		t.Fatal("expected referenced value object replacement not to be saved")
	}
	if gen.called {
		t.Fatal("expected referenced value object replacement not to regenerate plan")
	}
}

func TestDefaultServicePlanGenerationUsesRealPortsWithoutWriting(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "microgen.json")
	if err := os.WriteFile(configPath, []byte(validJSONConfig), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	outputDir := filepath.Join(t.TempDir(), "generated")
	service, err := DefaultService()
	if err != nil {
		t.Fatalf("create default service: %v", err)
	}

	plan, err := service.PlanGeneration(GenerateRequest{ConfigPath: configPath, OutputDir: outputDir})

	if err != nil {
		t.Fatalf("expected plan, got %v", err)
	}
	if plan.OutputDir != outputDir {
		t.Fatalf("expected output dir %q, got %q", outputDir, plan.OutputDir)
	}
	if plan.FileCount != 44 || len(plan.Files) != 44 {
		t.Fatalf("expected 44 planned files, got count=%d len=%d", plan.FileCount, len(plan.Files))
	}
	if plan.Files[0].Path != "CommercePlatform.sln" {
		t.Fatalf("expected first deterministic planned path, got %q", plan.Files[0].Path)
	}
	if _, err := os.Stat(outputDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected planning not to create output dir, stat err=%v", err)
	}
}

func TestDefaultServicePlanGenerationRejectsSymlinkOutputAncestor(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "microgen.json")
	if err := os.WriteFile(configPath, []byte(validJSONConfig), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	root := t.TempDir()
	realOutputParent := filepath.Join(root, "real-output-parent")
	if err := os.Mkdir(realOutputParent, 0o755); err != nil {
		t.Fatalf("create real output parent: %v", err)
	}
	symlinkOutputParent := filepath.Join(root, "linked-output-parent")
	if err := os.Symlink(realOutputParent, symlinkOutputParent); err != nil {
		t.Fatalf("create symlink output parent: %v", err)
	}
	service, err := DefaultService()
	if err != nil {
		t.Fatalf("create default service: %v", err)
	}

	_, err = service.PlanGeneration(GenerateRequest{ConfigPath: configPath, OutputDir: filepath.Join(symlinkOutputParent, "generated")})

	if err == nil {
		t.Fatal("expected symlink output ancestor to be rejected")
	}
	if !strings.Contains(err.Error(), "because existing ancestor") || !strings.Contains(err.Error(), "is a symlink") {
		t.Fatalf("expected symlink ancestor rejection, got %v", err)
	}
}

type fakeConfigLoader struct {
	path string
	cfg  spec.Config
	err  error
}

func (loader *fakeConfigLoader) LoadConfig(path string) (spec.Config, error) {
	loader.path = path
	return loader.cfg, loader.err
}

type fakeConfigSaver struct {
	called bool
	path   string
	cfg    spec.Config
	err    error
}

func (saver *fakeConfigSaver) SaveConfig(path string, cfg spec.Config) error {
	saver.called = true
	saver.path = path
	saver.cfg = cfg
	return saver.err
}

type fakeConfigValidator struct {
	cfg spec.Config
	err error
}

func (validator *fakeConfigValidator) ValidateConfig(cfg spec.Config) error {
	validator.cfg = cfg
	return validator.err
}

type fakeGenerator struct {
	called bool
	cfg    spec.Config
	files  []GeneratedFile
	err    error
}

func (gen *fakeGenerator) Generate(cfg spec.Config) ([]GeneratedFile, error) {
	gen.called = true
	gen.cfg = cfg
	return gen.files, gen.err
}

type fakeOutputWriter struct {
	called    bool
	outputDir string
	files     []GeneratedFile
	force     bool
	result    WriteResult
	err       error
}

func (writer *fakeOutputWriter) WriteGeneration(outputDir string, files []GeneratedFile, force bool) (WriteResult, error) {
	writer.called = true
	writer.outputDir = outputDir
	writer.files = files
	writer.force = force
	return writer.result, writer.err
}

type fakeOutputPlanner struct {
	outputDir string
	files     []GeneratedFile
	force     bool
	plan      OutputPlan
	err       error
}

func (planner fakeOutputPlanner) PlanOutput(outputDir string, files []GeneratedFile, force bool) (OutputPlan, error) {
	planner.outputDir = outputDir
	planner.files = files
	planner.force = force
	return planner.plan, planner.err
}

func validConfig() spec.Config {
	return spec.Config{
		Generation: spec.GenerationOptions{TargetFramework: "net8.0"},
		Solution:   spec.Solution{Name: "CommercePlatform", Description: "Product management."},
		Services: []spec.Service{
			{
				Name: "ProductService",
				ValueObjects: []spec.ValueObject{
					{Name: "ProductName", Type: "string"},
				},
				Entities: []spec.Entity{
					{
						Name: "Product",
						Fields: []spec.Field{
							{Name: "Id", Type: "Guid"},
							{Name: "Name", Type: "string"},
						},
					},
				},
			},
			{
				Name: "OrderService",
				ValueObjects: []spec.ValueObject{
					{Name: "OrderNumber", Type: "string"},
					{Name: "Money", Type: "decimal"},
				},
				Entities: []spec.Entity{
					{Name: "Order"},
					{Name: "OrderLine"},
				},
			},
		},
	}
}

func validPersistableConfig() spec.Config {
	return spec.Config{
		SchemaVersion: spec.ConfigSchemaVersion,
		Generation:    spec.GenerationOptions{TargetFramework: "net8.0"},
		Solution:      spec.Solution{Name: "CommercePlatform", Description: "Product management."},
		Services: []spec.Service{
			{
				Name: "ProductService",
				ValueObjects: []spec.ValueObject{
					{Name: "ProductName", Type: "string"},
				},
				Entities: []spec.Entity{
					{
						Name: "Product",
						Fields: []spec.Field{
							{Name: "Id", Type: "Guid"},
							{Name: "Name", Type: "ProductName"},
						},
					},
				},
			},
		},
	}
}

func validConfigWithEntities(names ...string) spec.Config {
	cfg := validPersistableConfig()
	cfg.Services[0].Entities = make([]spec.Entity, len(names))
	for index, name := range names {
		cfg.Services[0].Entities[index] = spec.Entity{
			Name: name,
			Fields: []spec.Field{
				{Name: "Id", Type: "Guid"},
				{Name: "Name", Type: "ProductName"},
			},
		}
	}
	return cfg
}

func validConfigWithValueObjectsForEditing() spec.Config {
	cfg := validPersistableConfig()
	minLength := 2
	maxLength := 20
	validExample := "Code"
	cfg.Services[0].ValueObjects = []spec.ValueObject{
		{Name: "ProductCode", Type: "string", Validations: spec.ValidationRules{MinLength: &minLength, MaxLength: &maxLength, ValidExample: &validExample}},
		{Name: "LegacyCode", Type: "string", Validations: spec.ValidationRules{MinLength: &minLength, MaxLength: &maxLength, ValidExample: &validExample}},
	}
	cfg.Services[0].Entities[0].Fields[1].Type = "string"
	return cfg
}

func serviceNames(services []spec.Service) []string {
	names := make([]string, len(services))
	for index, service := range services {
		names[index] = service.Name
	}
	return names
}

const validJSONConfig = `{
  "solution": { "name": "CommercePlatform", "description": "Product management." },
  "services": [
    {
      "name": "ProductService",
      "entities": [
        {
          "name": "Product",
          "fields": [
            { "name": "Id", "type": "Guid" },
            { "name": "Name", "type": "string" }
          ]
        }
      ]
    }
  ]
}`
