# Feature: TUI UX v0.9.1

## Objetivo
Mejorar la usabilidad del TUI con soporte de mouse, atajos de teclado y mejor discoverability de features existentes.

## Estado
🟡 En progreso

## Problemas identificados
1. ❌ No hay soporte de mouse (todo es por teclado/tabs)
2. ❌ Ctrl+S no funciona en modales
3. ❌ No hay selector de generation mode (simple/enterprise)
4. ❌ Connection string no es discoverable (existe con tecla 'c' pero no visible)
5. ❌ Editor de fields poco intuitivo

## Especificaciones

### S1: Soporte de mouse
- Click simple: seleccionar elementos en listas
- Doble click: ejecutar acciones/botones (evitar misclicks)
- Scroll con rueda: navegar en listas y modals
- Implementar usando bubbletea mouse support

### S2: Atajos de teclado mejorados
- Ctrl+S: guardar en todos los modales (equivalente a Enter/Apply)
- Mantener Enter como alternativa para compatibilidad

### S3: Generation mode selector
- Agregar campo visible en Project editor
- Permitir seleccionar entre "simple" y "enterprise"
- Mostrar descripción de cada modo
- Persistir en config.json

### S4: Connection string visible
- Mostrar connection string en lista de servicios (si está configurado)
- Agregar botón/campo visible para editar
- Mantener tecla 'c' como atajo
- Mostrar "(no configurado)" si está vacío

### S5: Editor de fields mejorado
- Hacer más discoverable (botón visible, no solo tecla 'f')
- Mostrar ayuda contextual sobre campos vs Value Objects
- Agregar indicador visual de tipos de campo

## Tasks

### T1: Agregar soporte de mouse básico
- Route: inline
- Implementar mouse handling en bubbletea
- Click simple para seleccionar
- Doble click para acciones
- Scroll wheel para navegación
- Tests: verificar eventos de mouse

### T2: Agregar Ctrl+S en modales
- Route: inline
- Capturar Ctrl+S en update handlers
- Ejecutar mismo handler que Enter
- Tests: verificar guardado con Ctrl+S

### T3: Agregar generation mode selector
- Route: inline
- Agregar campo en editState
- Renderizar selector en project editor
- Persistir en config
- Tests: verificar selección y persistencia

### T4: Hacer connection string visible
- Route: inline
- Mostrar en lista de servicios
- Agregar botón para editar
- Tests: verificar visibilidad y edición

### T5: Mejorar editor de fields
- Route: inline
- Agregar botón visible (no solo tecla 'f')
- Mostrar ayuda contextual
- Tests: verificar discoverability

### T6: Testing integral del TUI
- Route: inline
- Probar todas las features con mouse
- Verificar Ctrl+S en todos los modales
- Testear generation mode
- Testear connection string
- Testear editor de fields

## Prioridades
1. T1 (mouse) - fundamental para UX moderna
2. T2 (Ctrl+S) - atajo esperado
3. T4 (connection string visible) - feature existente debe ser visible
4. T3 (generation mode) - feature faltante
5. T5 (fields editor) - mejora de UX
6. T6 (testing) - validación final

## Criterios de aceptación
- ✅ Mouse funciona en todo el TUI
- ✅ Ctrl+S guarda en todos los modales
- ✅ Generation mode es seleccionable
- ✅ Connection string es visible y editable
- ✅ Editor de fields es discoverable
- ✅ Todos los tests pasan

## Notas técnicas
- Bubbletea soporta mouse events nativamente
- Doble click requiere timing (ej: 500ms entre clicks)
- Mantener compatibilidad con navegación por teclado existente
- No breaking changes en API
