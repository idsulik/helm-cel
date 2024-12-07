[![License](https://img.shields.io/github/license/idsulik/helm-cel.svg)](https://github.com/idsulik/helm-cel/blob/main/LICENSE)
[![Current Release](https://img.shields.io/github/release/idsulik/helm-cel.svg?logo=github)](https://github.com/idsulik/helm-cel/releases/latest)
[![GitHub Repo stars](https://img.shields.io/github/stars/idsulik/helm-cel?style=flat&logo=github)](https://github.com/idsulik/helm-cel/stargazers)
[![GitHub issues](https://img.shields.io/github/issues/idsulik/helm-cel.svg)](https://github.com/idsulik/helm-cel/issues)
[![GitHub pull requests](https://img.shields.io/github/issues-pr/idsulik/helm-cel.svg)](https://github.com/idsulik/helm-cel/pulls)
[![codecov](https://codecov.io/gh/idsulik/helm-cel/branch/main/graph/badge.svg?token=4qAukyB2yX)](https://codecov.io/gh/idsulik/helm-cel)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/cel)](https://artifacthub.io/packages/helm-plugin/cel/cel)
# Helm CEL Plugin

A Helm plugin that uses Common Expression Language (CEL) to validate values. Instead of using JSON Schema in `values.schema.json`, you can write more expressive validation rules using CEL in `values.cel.yaml`.

## Installation

### Using Helm Plugin Manager
```bash
helm plugin install https://github.com/idsulik/helm-cel
```

### Using Docker
```bash
# Pull the image
docker pull idsulik/helm-cel:latest # main branch
docker pull idsulik/helm-cel:2.1.2  # specific version

# Run validation
docker run --rm -v $(pwd):/charts idsulik/helm-cel validate /charts/mychart

# Generate validation rules
docker run --rm -v $(pwd):/charts idsulik/helm-cel generate /charts/mychart
```

### Building from Source
```bash
git clone https://github.com/idsulik/helm-cel
cd helm-cel
make install
```

## Usage

### Validation

Validate your chart values using the validate command:
```bash
helm cel validate ./mychart
```

Options:
```bash
--values-file, -v    Values files to validate (comma-separated or multiple flags)
                     Defaults to values.yaml
--rules-file, -r     Rules files to validate against (comma-separated or multiple flags)
                     Defaults to values.cel.yaml
```

Example with custom files:
```bash
# Using single values and rules files
helm cel validate ./mychart --values-file prod.values.yaml --rules-file prod.cel.yaml

# Using multiple values files (later files take precedence)
helm cel validate ./mychart --values-file common.yaml --values-file prod.yaml

# Using comma-separated values files
helm cel validate ./mychart --values-file common.yaml,prod.yaml,overrides.yaml

# Using multiple rules files
helm cel validate ./mychart --rules-file global.cel.yaml --rules-file ingress.cel.yaml

# Combining multiple values and rules files
helm cel validate ./mychart \
  --values-file common.yaml,prod.yaml \
  --rules-file global.cel.yaml,ingress.cel.yaml,deployment.cel.yaml
```

### Generating Rules

You can automatically generate validation rules based on your values file structure:
```bash
helm cel generate ./mychart
```

Options:
```bash
--force, -f          Force overwrite existing rules file
--values-file, -v    Values file to generate rules from (defaults to values.yaml)
--output-file, -o    Output file for generated rules (defaults to values.cel.yaml)
```

Example with custom files:
```bash
helm cel generate ./mychart --values-file prod.values.yaml --output-file prod.cel.yaml --force
```

## Rule Organization

You can organize your validation rules into multiple files for better maintainability. Files must have the `.cel.yaml` extension. Example structure:

```
mychart/
├── Chart.yaml
├── values.yaml
└── cel/
    ├── global.cel.yaml     # Global configuration rules
    ├── ingress.cel.yaml    # Ingress-specific rules
    └── deployment.cel.yaml # Deployment-specific rules
```

When using multiple rule files, expressions are shared across all files but must be unique (no duplicate expression names allowed).

## Rule Structure

Each rule in `values.cel.yaml` consists of:
- `expr`: A CEL expression that should evaluate to `true` for valid values
- `desc`: A description of what the rule validates
- `severity`: Optional severity level ("error" or "warning", defaults to "error")

Example `values.cel.yaml`:
```yaml
rules:
  - expr: "has(values.service) && has(values.service.port)"
    desc: "service port is required"
  
  - expr: "values.service.port >= 1 && values.service.port <= 65535"
    desc: "service port must be between 1 and 65535"
    severity: warning
  
  - expr: "!(has(values.replicaCount)) || values.replicaCount >= 1"
    desc: "if replicaCount is set, it must be at least 1"
```

### Severity Levels

Rules can have two severity levels:
- `error`: Validation fails if the rule is not satisfied (default)
- `warning`: Shows a warning but allows validation to pass

### Common Validation Patterns

1. Required fields:
```yaml
- expr: "has(values.fieldName)"
  desc: "fieldName is required"
```

2. Value constraints:
```yaml
- expr: "values.number >= 0 && values.number <= 100"
  desc: "number must be between 0 and 100"
```

3. Type validation:
```yaml
- expr: "type(values.ports) == list"
  desc: "ports must be a list"
```

4. Resource validation:
```yaml
- expr: 'values.resources.requests.memory.matches("^[0-9]+(Mi|Gi)$")'
  desc: "memory requests must be in Mi or Gi"
```

5. Port validation:
```yaml
- expr: "values.service.port >= 1 && values.service.port <= 65535"
  desc: "port must be valid"
```

### Reusable Expressions

You can define expressions to reuse across rules:
```yaml
expressions:
  portRange: 'values.service.port >= 1 && values.service.port <= 65535'
  nodePortRange: 'values.service.nodePort >= 30000 && values.service.nodePort <= 32767'

rules:
  - expr: "${portRange}"
    desc: "Service port must be valid"
  
  - expr: 'values.service.type == "NodePort" ? ${nodePortRange} : true'
    desc: "NodePort must be valid when type is NodePort"
```

### Validation Results

If validation fails, you'll get a clear error message:
```
❌ Validation failed: replica count must be at least 1
   Rule: values.replicaCount >= 1
   Path: replicaCount
   Current value: 0
```

With warnings:
```
Found 1 warning(s):

⚠️ Service port must be between 1 and 65535
   Rule: values.service.port >= 1 && values.service.port <= 65535
   Path: service.port
   Current value: 80801
-------------------------------------------------
⚠️✅ Values validation successful with warnings!
```

If all rules pass, you'll see a success message:
```
✅ Values validation successful!
```

## Development

Requirements:
- Go 1.22 or later

Build:
```bash
make build
```

Install locally:
```bash
make install
```

Run tests:
```bash
make test
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

Distributed under the MIT License. See `LICENSE` for more information.