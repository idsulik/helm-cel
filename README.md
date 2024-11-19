[![License](https://img.shields.io/github/license/idsulik/helm-cel.svg)](https://github.com/idsulik/helm-cel/blob/main/LICENSE)
[![Current Release](https://img.shields.io/github/release/idsulik/helm-cel.svg?logo=github)](https://github.com/idsulik/helm-cel/releases/latest)
[![GitHub Repo stars](https://img.shields.io/github/stars/idsulik/helm-cel?style=flat&logo=github)](https://github.com/idsulik/helm-cel/stargazers)
[![GitHub all releases](https://img.shields.io/github/downloads/idsulik/helm-cel/total?logo=github)](https://github.com/idsulik/helm-cel/releases/latest)
[![GitHub issues](https://img.shields.io/github/issues/idsulik/helm-cel.svg)](https://github.com/idsulik/helm-cel/issues)
[![GitHub pull requests](https://img.shields.io/github/issues-pr/idsulik/helm-cel.svg)](https://github.com/idsulik/helm-cel/pulls)
[![codecov](https://codecov.io/gh/idsulik/helm-cel/branch/main/graph/badge.svg?token=4qAukyB2yX)](https://codecov.io/gh/idsulik/helm-cel)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/cel)](https://artifacthub.io/packages/helm-plugin/cel/cel)
# Helm CEL Plugin

A Helm plugin that uses Common Expression Language (CEL) to validate values. Instead of using JSON Schema in `values.schema.json`, you can write more expressive validation rules using CEL in `values.cel.yaml`.

## Installation

```bash
helm plugin install https://github.com/idsulik/helm-cel
```

## Usage

Create a `values.cel.yaml` file in your chart directory alongside your `values.yaml` file:

```yaml
rules:
  - expr: "has(values.service) && has(values.service.port)"
    desc: "service port is required"
  
  - expr: "values.service.port >= 1 && values.service.port <= 65535"
    desc: "service port must be between 1 and 65535"
  
  - expr: "!(has(values.replicaCount)) || values.replicaCount >= 1"
    desc: "if replicaCount is set, it must be at least 1"
```

Then run validation:

```bash
helm cel /path/to/your/chart
```

## Example

Given this `values.yaml`:

```yaml
service:
  type: ClusterIP
  port: 80

replicaCount: 1

image:
  repository: nginx
  tag: latest
```

And this `values.cel.yaml`:

```yaml
rules:
  - expr: "has(values.service) && has(values.service.port)"
    desc: "service port is required"
  
  - expr: "values.service.port >= 1 && values.service.port <= 65535"
    desc: "service port must be between 1 and 65535"
  
  - expr: "values.replicaCount >= 1"
    desc: "replica count must be at least 1"
  
  - expr: |
      has(values.image) && 
      has(values.image.repository) && 
      has(values.image.tag)
    desc: "image repository and tag are required"
```

If validation fails, you'll get a clear error message:

```
❌ Validation failed: replica count must be at least 1
   Rule: values.replicaCount >= 1
   Path: replicaCount
   Current value: 0
```

## Writing Validation Rules

Each rule in `values.cel.yaml` consists of:
- `expr`: A CEL expression that should evaluate to `true` for valid values
- `desc`: A description of what the rule validates

CEL expressions have access to the entire values structure through the `values` variable. Some useful CEL functions:

- `has(values.path)` - checks if a field exists
- `type(value)` - returns the type of a value
- `size(list)` - returns the length of a list or map

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

3. Conditional requirements:
```yaml
- expr: "!(has(values.optional)) || values.optional >= 0"
  desc: "if optional is set, it must be non-negative"
```

4. Type validation:
```yaml
- expr: "type(values.ports) == list"
  desc: "ports must be a list"
```

5. Complex object validation:
```yaml
- expr: |
    has(values.container) && 
    has(values.container.image) && 
    has(values.container.tag)
  desc: "container must have image and tag"
```

## Development

Requirements:
- Go 1.20 or later

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
