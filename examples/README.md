# gref Examples

This directory contains examples demonstrating gref's capabilities.

## Running Examples

Each example is a standalone program in its own directory:

```bash
cd json_serializer && go run main.go
cd config_loader && go run main.go
cd orm_lite && go run main.go
cd validator && go run main.go
cd plugin_system && go run main.go
```

## Examples Overview

| Example | Description | Key Features |
|---------|-------------|--------------|
| `json_serializer` | Custom JSON serializer respecting struct tags | `ParsedTag`, `Fields().Each()`, `HasOption()` |
| `config_loader` | Load config from environment variables | Dot notation, `TryField()`, nested struct navigation |
| `orm_lite` | Build structs dynamically from DB schema | `Def().Struct()`, `FieldDef`, runtime type creation |
| `validator` | Struct validation with custom tags | `ParsedTag`, rule parsing, type-safe extraction |
| `plugin_system` | Dynamic function wrappers and middleware | `Def().Func()`, `CallWith()`, function introspection |

## Complexity Progression

1. **json_serializer** (Beginner) - Basic struct introspection and tag handling
2. **config_loader** (Beginner) - Nested field access with dot notation
3. **validator** (Intermediate) - Complex tag parsing with multiple rules
4. **orm_lite** (Intermediate) - Runtime struct definition
5. **plugin_system** (Advanced) - Dynamic function creation and composition
