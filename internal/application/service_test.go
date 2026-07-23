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
			{Name: "ProductService", EntityNames: []string{"Product"}},
			{Name: "OrderService", EntityNames: []string{"Order", "OrderLine"}},
		},
	}
	if !reflect.DeepEqual(plan.Config, expectedSummary) {
		t.Fatalf("expected config summary %#v, got %#v", expectedSummary, plan.Config)
	}
	expectedFiles := []PlannedFile{{Path: "README.md", Action: "replace"}, {Path: "src/ProductService/Product.cs", Action: "unchanged"}}
	if !reflect.DeepEqual(plan.Files, expectedFiles) {
		t.Fatalf("expected planned files %#v, got %#v", expectedFiles, plan.Files)
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
