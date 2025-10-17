# Buf Usage in HTTPQueue

This project uses [Buf](https://buf.build) for Protocol Buffer management instead of the traditional `protoc` command.

## Configuration Files

### `buf.yaml`
Main configuration file for buf:
- Enables linting with DEFAULT rules
- Enables breaking change detection
- Declares dependencies

### `buf.gen.yaml`
Code generation configuration:
- Uses official Go and gRPC Go plugins
- Generates code with `source_relative` paths

## Commands

### Generate Code
```bash
# Generate protobuf and gRPC code
make proto

# Or directly with buf
buf generate
```

### Linting
```bash
# Lint protobuf files
make proto-lint

# Or directly with buf
buf lint
```

### Formatting
```bash
# Format protobuf files
make proto-format

# Or directly with buf
buf format -w
```

### Breaking Change Detection
```bash
# Check for breaking changes against main branch
buf breaking --against '.git#branch=main'
```

## Benefits of Using Buf

1. **Better Linting**: More comprehensive style and API design checks
2. **Breaking Change Detection**: Automatic detection of API breaking changes
3. **Remote Plugins**: Uses remote plugins, no need to install protoc locally
4. **Better Error Messages**: More descriptive error messages and suggestions
5. **Dependency Management**: Manages protobuf dependencies automatically
6. **Format Consistency**: Automatic code formatting for proto files

## File Structure

```
proto/
├── webhook.proto          # Service definitions
├── webhook.pb.go          # Generated Go structs (generated)
└── webhook_grpc.pb.go     # Generated gRPC code (generated)

buf.yaml                   # Buf configuration
buf.gen.yaml              # Code generation configuration
```

## Migration from protoc

The old `generate_proto.sh` script has been replaced with buf. The equivalent commands are:

| Old (protoc) | New (buf) |
|-------------|-----------|
| `./generate_proto.sh` | `make proto` |
| Manual protoc command | `buf generate` |
| No equivalent | `buf lint` |
| No equivalent | `buf format` |

All generated files remain the same and are compatible with existing code.