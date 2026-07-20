package configloader

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pozeydon-code/generator-microservices-go/internal/spec"
)

const MaxConfigBytes int64 = 1 << 20

func LoadJSON(path string) (spec.Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return spec.Config{}, fmt.Errorf("read config: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, MaxConfigBytes+1))
	if err != nil {
		return spec.Config{}, fmt.Errorf("read config: %w", err)
	}
	if int64(len(content)) > MaxConfigBytes {
		return spec.Config{}, fmt.Errorf("config exceeds %d byte limit", MaxConfigBytes)
	}

	return loadJSONConfig(content)
}

func loadJSONConfig(content []byte) (spec.Config, error) {
	schemaVersion, err := detectRootSchemaVersion(content)
	if err != nil {
		return spec.Config{}, fmt.Errorf("parse config JSON: %w", err)
	}

	if err := validateJSONKeys(content); err != nil {
		return spec.Config{}, fmt.Errorf("parse config JSON: %w", err)
	}

	var cfg spec.Config
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return spec.Config{}, fmt.Errorf("parse config JSON: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return spec.Config{}, errors.New("parse config JSON: trailing data after top-level value")
	}

	if err := cfg.Validate(); err != nil {
		return spec.Config{}, err
	}

	return migrateConfig(cfg, schemaVersion)
}

type schemaVersion struct {
	value  int
	legacy bool
}

type objectSchema struct {
	name          string
	allowedKeys   map[string]string
	arrayTargets  map[string]objectSchema
	objectTargets map[string]objectSchema
}

var validationSchema = objectSchema{name: "validations", allowedKeys: allowed("required", "minLength", "maxLength", "pattern", "validExample", "invalidExample", "minimum", "maximum", "notEmpty", "notDefault")}
var valueObjectSchema = objectSchema{name: "valueObject", allowedKeys: allowed("name", "type", "validations"), objectTargets: map[string]objectSchema{"validations": validationSchema}}
var fieldSchema = objectSchema{name: "field", allowedKeys: allowed("name", "type")}
var entitySchema = objectSchema{name: "entity", allowedKeys: allowed("name", "fields"), arrayTargets: map[string]objectSchema{"fields": fieldSchema}}
var serviceSchema = objectSchema{name: "service", allowedKeys: allowed("name", "valueObjects", "entities"), arrayTargets: map[string]objectSchema{"valueObjects": valueObjectSchema, "entities": entitySchema}}
var rootSchema = objectSchema{name: "root", allowedKeys: allowed("schemaVersion", "generation", "solution", "services"), arrayTargets: map[string]objectSchema{"services": serviceSchema}}
var generationSchema = objectSchema{name: "generation", allowedKeys: allowed("targetFramework")}
var solutionSchema = objectSchema{name: "solution", allowedKeys: allowed("name", "description")}

func allowed(keys ...string) map[string]string {
	result := map[string]string{}
	for _, key := range keys {
		result[key] = key
	}
	return result
}

func validateJSONKeys(content []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()
	if err := readValue(decoder, rootSchema); err != nil {
		return err
	}
	if token, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err != nil {
			return err
		}
		return fmt.Errorf("trailing data after top-level value near %v", token)
	}
	return nil
}

func detectRootSchemaVersion(content []byte) (schemaVersion, error) {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()
	token, err := decoder.Token()
	if err != nil {
		return schemaVersion{}, err
	}
	delim, ok := token.(json.Delim)
	if !ok || delim != '{' {
		return schemaVersion{}, errors.New("top-level config must be a JSON object")
	}

	seenSchemaVersion := false
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return schemaVersion{}, err
		}
		key, ok := keyToken.(string)
		if !ok {
			return schemaVersion{}, errors.New("expected object key in root")
		}
		if key != "schemaVersion" {
			var ignored json.RawMessage
			if err := decoder.Decode(&ignored); err != nil {
				return schemaVersion{}, err
			}
			continue
		}
		if seenSchemaVersion {
			return schemaVersion{}, errors.New("duplicate key \"schemaVersion\" in root object")
		}
		seenSchemaVersion = true

		version, err := decodeSchemaVersionValue(decoder)
		if err != nil {
			return schemaVersion{}, err
		}
		if version <= 0 {
			return schemaVersion{}, fmt.Errorf("schemaVersion must be %d when present", spec.ConfigSchemaVersion)
		}
		if version > spec.ConfigSchemaVersion {
			return schemaVersion{}, fmt.Errorf("unsupported schemaVersion %d; current schemaVersion is %d", version, spec.ConfigSchemaVersion)
		}
		return schemaVersion{value: version}, nil
	}
	_, err = decoder.Token()
	if err != nil {
		return schemaVersion{}, err
	}
	return schemaVersion{value: spec.ConfigSchemaVersion, legacy: true}, nil
}

func decodeSchemaVersionValue(decoder *json.Decoder) (int, error) {
	var value any
	if err := decoder.Decode(&value); err != nil {
		return 0, err
	}
	version, ok := value.(json.Number)
	if !ok {
		return 0, errors.New("schemaVersion must be an integer")
	}
	parsed, err := version.Int64()
	if err != nil {
		return 0, errors.New("schemaVersion must be an integer")
	}
	return int(parsed), nil
}

func migrateConfig(cfg spec.Config, schemaVersion schemaVersion) (spec.Config, error) {
	switch {
	case schemaVersion.legacy:
		cfg.SchemaVersion = spec.ConfigSchemaVersion
		return cfg, nil
	case schemaVersion.value == spec.ConfigSchemaVersion:
		return cfg, nil
	default:
		return spec.Config{}, fmt.Errorf("unsupported schemaVersion %d; current schemaVersion is %d", schemaVersion.value, spec.ConfigSchemaVersion)
	}
}

func readValue(decoder *json.Decoder, schema objectSchema) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		return readObject(decoder, schema)
	case '[':
		for decoder.More() {
			if err := readValue(decoder, schema); err != nil {
				return err
			}
		}
		_, err := decoder.Token()
		return err
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delim)
	}
}

func readObject(decoder *json.Decoder, schema objectSchema) error {
	seen := map[string]struct{}{}
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return err
		}
		key, ok := keyToken.(string)
		if !ok {
			return fmt.Errorf("expected object key in %s", schema.name)
		}
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate key %q in %s object", key, schema.name)
		}
		seen[key] = struct{}{}
		canonical, ok := schema.allowedKeys[key]
		if !ok {
			for allowedKey := range schema.allowedKeys {
				if strings.EqualFold(key, allowedKey) {
					return fmt.Errorf("incorrectly cased key %q in %s object; expected %q", key, schema.name, allowedKey)
				}
			}
			return fmt.Errorf("unknown key %q in %s object", key, schema.name)
		}
		childSchema := objectSchema{}
		if schema.name == "root" && canonical == "generation" {
			childSchema = generationSchema
		}
		if schema.name == "root" && canonical == "solution" {
			childSchema = solutionSchema
		}
		if target, ok := schema.arrayTargets[canonical]; ok {
			childSchema = target
		}
		if target, ok := schema.objectTargets[canonical]; ok {
			childSchema = target
		}
		if schema.name == "root" && canonical == "schemaVersion" {
			if err := readSchemaVersion(decoder); err != nil {
				return err
			}
			continue
		}
		if err := readValue(decoder, childSchema); err != nil {
			return err
		}
	}
	_, err := decoder.Token()
	return err
}

func readSchemaVersion(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	version, ok := token.(json.Number)
	if !ok {
		return fmt.Errorf("schemaVersion must be %d", spec.ConfigSchemaVersion)
	}
	value, err := version.Int64()
	if err != nil || value != spec.ConfigSchemaVersion {
		return fmt.Errorf("schemaVersion must be %d", spec.ConfigSchemaVersion)
	}
	return nil
}
