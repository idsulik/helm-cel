# Common CEL Expressions for Helm Values Validation

This document contains common CEL (Common Expression Language) expressions that you can use with helm-cel plugin to validate your Helm charts' values.

## Table of Contents
- [Basic Validations](#basic-validations)
- [String Validations](#string-validations)
- [Numeric Validations](#numeric-validations)
- [Array and Map Validations](#array-and-map-validations)
- [Kubernetes Specific Validations](#kubernetes-specific-validations)
- [Named Expression Examples](#named-expression-examples)

## Basic Validations

### Required Fields
```yaml
rules:
  - expr: "has(values.service)"
    desc: "service configuration is required"
  - expr: "has(values.service.port)"
    desc: "service port must be specified"
```

### Type Checking
```yaml
rules:
  - expr: "type(values.replicas) == int"
    desc: "replicas must be an integer"
  - expr: "type(values.labels) == map"
    desc: "labels must be a map"
```

## String Validations

### String Pattern Matching
```yaml
expressions:
  imagePattern: 'matches(string(value), r"^[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?/[^/]+/[^/]+$")'
  k8sNamePattern: 'matches(string(value), r"^[a-z0-9][a-z0-9-]*[a-z0-9]$")'
  
rules:
  - expr: "${imagePattern}"
    desc: "image name must be valid"
  - expr: "size(values.name) <= 63 && ${k8sNamePattern}"
    desc: "name must be a valid Kubernetes name"
```

### Enum Values
```yaml
rules:
  - expr: 'values.service.type in ["ClusterIP", "NodePort", "LoadBalancer"]'
    desc: "service type must be one of: ClusterIP, NodePort, LoadBalancer"
  - expr: 'values.logLevel in ["debug", "info", "warn", "error"]'
    desc: "invalid log level"
```

## Numeric Validations

### Port Validations
```yaml
expressions:
  validPort: "value >= 1 && value <= 65535"
  
rules:
  - expr: "${validPort}"
    desc: "port number must be between 1 and 65535"
  - expr: "values.service.port != 443 && values.service.port != 80"
    desc: "port 80 and 443 are reserved"
```

### Resource Validations
```yaml
expressions:
  memoryPattern: 'matches(string(value), r"^[0-9]+(Mi|Gi)$")'
  cpuPattern: 'matches(string(value), r"^[0-9]+m$|^[0-9]+$|^[0-9]+\\.[0-9]+$")'
  
rules:
  - expr: "has(values.resources.requests) && has(values.resources.limits)"
    desc: "both resource requests and limits must be specified"
  - expr: 'values.resources.requests.memory.matches("^[0-9]+(Mi|Gi)$")'
    desc: "memory request must be in Mi or Gi"
  - expr: 'double(values.resources.limits.cpu) <= 4.0'
    desc: "CPU limit must not exceed 4 cores"
```

## Array and Map Validations

### Array Length and Content
```yaml
rules:
  - expr: "size(values.ingress.hosts) > 0"
    desc: "at least one ingress host must be specified"
  - expr: "size(values.ingress.hosts) <= 5"
    desc: "maximum 5 ingress hosts allowed"
```

### Map Properties
```yaml
rules:
  - expr: "size(values.labels) <= 10"
    desc: "maximum 10 labels allowed"
  - expr: "!has(values.labels.kubernetes) && !has(values.labels.k8s)"
    desc: "kubernetes and k8s labels are reserved"
```

## Kubernetes Specific Validations

### Pod Security
```yaml
rules:
  - expr: "!values.securityContext.privileged"
    desc: "privileged containers are not allowed"
  - expr: "values.securityContext.runAsNonRoot"
    desc: "containers must run as non-root"
```

### Container Image
```yaml
expressions:
  noLatestTag: '!values.image.tag.endsWith(":latest")'
  trustedRegistry: 'values.image.repository.startsWith("gcr.io/") || values.image.repository.startsWith("docker.io/company/")'
  
rules:
  - expr: "${noLatestTag}"
    desc: "latest tag is not allowed"
  - expr: "${trustedRegistry}"
    desc: "image must come from trusted registry"
```

## Named Expression Examples

Here's an example of how to use named expressions to create reusable validation rules:

```yaml
expressions:
  # Resource validation patterns
  memoryPattern: 'matches(string(value), r"^[0-9]+(Mi|Gi)$")'
  cpuPattern: 'matches(string(value), r"^[0-9]+m$|^[0-9]+$|^[0-9]+\\.[0-9]+$")'
  
  # Common validations
  hasResources: "has(values.resources.requests) && has(values.resources.limits)"
  validPort: "int(value) >= 1 && int(value) <= 65535"
  k8sName: 'matches(string(value), r"^[a-z0-9][a-z0-9-]*[a-z0-9]$")'
  
  # Composite validations
  validateResources: """
    ${hasResources} && 
    ${memoryPattern} && 
    ${cpuPattern} &&
    double(values.resources.limits.cpu) <= 4.0 &&
    double(values.resources.requests.cpu) <= double(values.resources.limits.cpu)
  """

rules:
  - expr: "${validateResources}"
    desc: "Invalid resource configuration"
  - expr: "${validPort}"
    desc: "Invalid port number"
  - expr: "${k8sName}"
    desc: "Invalid Kubernetes resource name"
```

## Usage Tips

1. Use named expressions to make your rules more readable and maintainable
2. Break down complex validations into smaller, reusable expressions
3. Add descriptive error messages to help users understand validation failures
4. Use warnings for recommended practices that shouldn't block deployment
5. Keep expressions simple and focused on a single aspect of validation

## Examples for Common Use Cases

### Basic Web Application
```yaml
expressions:
  validPort: "int(value) >= 1 && int(value) <= 65535"
  hasProbes: "has(values.livenessProbe) && has(values.readinessProbe)"
  
rules:
  - expr: "${validPort}"
    desc: "Invalid port configuration"
  - expr: "${hasProbes}"
    desc: "Health probes should be configured"
    severity: warning
  - expr: "values.replicas >= 2"
    desc: "Running single replica in production"
    severity: warning
```

### Database
```yaml
expressions:
  validStorageSize: 'matches(string(values.persistence.size), r"^[0-9]+(Gi)$")'
  
rules:
  - expr: "${validStorageSize}"
    desc: "Storage size must be specified in Gi"
  - expr: "int(values.persistence.size.replace('Gi', '')) >= 10"
    desc: "Minimum storage size is 10Gi"
  - expr: "has(values.backup.schedule)"
    desc: "Backup schedule should be configured"
    severity: warning
```