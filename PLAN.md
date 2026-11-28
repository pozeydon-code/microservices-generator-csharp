# PLAN.md - microgen v0.9.0 y siguientes versiones

## v0.9.0 - Connection String Configuration (ACTUAL)

**Objetivo**: Permitir configurar el connection string de cada servicio desde el TUI.

### Cambios implementados:

1. **Schema de configuración**
   - Agregado campo `connectionString` a `Service` en `internal/spec/config.go`
   - El campo es opcional (omitempty)

2. **Generación de templates**
   - Actualizados `appsettings.json` y `appsettings.Development.json` para usar `{{ .ConnectionString }}`
   - Agregado campo `ConnectionString` a `ServiceView` en `internal/generator/view_builder.go`

3. **Capa de aplicación**
   - Agregado `ConnectionString` a `ServiceNameSetting` en `internal/application/service.go`
   - Agregado `ConnectionString` a `ServiceSummary` para persistencia
   - Actualizada función `updatedServices` para preservar connection strings al renombrar

4. **TUI (Terminal User Interface)**
   - Agregado `connectionStrings` y `connStringFields` a `servicesEditState`
   - Agregada tecla `c` para editar el connection string del servicio seleccionado
   - El connection string se muestra debajo del nombre del servicio
   - Si no está configurado, muestra "(not set)"
   - Actualizada función `saveServicesCmd` para guardar connection strings
   - Actualizada función `startServicesEditing` para cargar connection strings existentes

5. **Tests**
   - Actualizado test `TestModelUpdateEditsServicesAndSaves` para incluir la nueva tecla `c`
   - Todos los tests pasan correctamente

### Uso:

En el TUI, al editar servicios:
- Presionar `c` sobre un servicio para editar su connection string
- El connection string se guarda en el archivo de configuración JSON
- Al regenerar, el connection string se inyecta en `appsettings.json`

### Ejemplo de configuración:

```json
{
  "services": [
    {
      "name": "ProductService",
      "connectionString": "Server=localhost;Database=Products;Trusted_Connection=True;",
      "entities": [...]
    }
  ]
}
```

### Estado: ✅ COMPLETADO

---

## Próximas versiones

### v0.10.0 - Documentación de relaciones
- Agregar sección de relaciones en README.md
- Documentar one-to-one, one-to-many, many-to-many
- Agregar ejemplos de configuración de relaciones

### v0.11.0 - Validación CI del código generado
- Pipeline CI que genera, compila y teste código generado
- Validación en múltiples target frameworks
- Matriz de cobertura de modos y features

### v0.12.0 - Gateway funcional con JWT
- Middleware de validación JWT en gateway
- Routing a servicios downstream
- Health checks del gateway
- Tests de integración

### v1.0.0 - Release estable
- Todas las features anteriores integradas
- Documentación completa
- Pruebas de producción

---

## Historial de versiones anteriores

### v0.8.2 - Regeneración incremental selectiva ✅
- Regenera solo archivos cuyo contenido cambió
- Preserva ediciones manuales
- Reporta conflictos sin sobrescribir

### v0.8.1 - Foundation
- Generador de microservicios .NET con Clean Architecture
- CLI y TUI
- CQRS con MediatR
- Query shapes
