# Parameterized Expressions

This document explains how to use parameterized expressions in helm-cel to create reusable validation patterns.

## Overview

Parameterized expressions allow you to define reusable validation logic that can be applied to different fields or values. This reduces duplication and makes your validation rules more maintainable.

## Syntax

### Basic Syntax

Define an expression with positional parameters using `$0`, `$1`, `$2`, etc.:

```yaml
expressions:
  hasField: 'has($0)'
  inRange: '$0 >= $1 && $0 <= $2'

rules:
  - expr: '${hasField(values.foo.bar)}'
    desc: "foo.bar must be defined"
  
  - expr: '${inRange(values.port, 1, 65535)}'
    desc: "port must be between 1 and 65535"
```

### How It Works

When you use `${hasField(values.foo.bar)}`, the expression processor:
1. Finds the `hasField` expression definition: `has($0)`
2. Replaces `$0` with the argument `values.foo.bar`
3. Produces the final expression: `has(values.foo.bar)`

## Examples

### Example 1: Field Existence Check

Instead of writing:
```yaml
rules:
  - expr: 'has(values.service.port)'
    desc: "service.port is required"
  - expr: 'has(values.service.type)'
    desc: "service.type is required"
  - expr: 'has(values.ingress.host)'
    desc: "ingress.host is required"
```

You can write:
```yaml
expressions:
  hasField: 'has($0)'

rules:
  - expr: '${hasField(values.service.port)}'
    desc: "service.port is required"
  - expr: '${hasField(values.service.type)}'
    desc: "service.type is required"
  - expr: '${hasField(values.ingress.host)}'
    desc: "ingress.host is required"
```

### Example 2: Range Validation

```yaml
expressions:
  inRange: '$0 >= $1 && $0 <= $2'

rules:
  - expr: '${inRange(values.service.port, 1, 65535)}'
    desc: "service port must be valid"
  
  - expr: '${inRange(values.service.nodePort, 30000, 32767)}'
    desc: "nodePort must be in valid range"
  
  - expr: '${inRange(values.replicaCount, 1, 100)}'
    desc: "replicaCount must be between 1 and 100"
```

### Example 3: Complex Validation with Multiple Parameters

```yaml
expressions:
  validPort: '$0 >= 1 && $0 <= 65535'
  hasNonEmpty: 'has($0) && $0 != ""'
  matchesPattern: '$0.matches($1)'

rules:
  - expr: '${validPort(values.service.port)}'
    desc: "service port must be valid"
  
  - expr: '${hasNonEmpty(values.image.repository)}'
    desc: "image repository must be set"
  
  - expr: '${matchesPattern(values.resources.memory, "^[0-9]+(Mi|Gi)$")}'
    desc: "memory must be in Mi or Gi"
```

### Example 4: Nested Parameterized Expressions

You can use parameterized expressions within other expressions:

```yaml
expressions:
  portRange: '$0 >= 1 && $0 <= 65535'
  validatePort: 'has($0) && ${portRange($0)}'

rules:
  - expr: '${validatePort(values.service.port)}'
    desc: "service port must exist and be valid"
```

This expands to:
```
has(values.service.port) && (values.service.port >= 1 && values.service.port <= 65535)
```

### Example 5: Handling Complex Arguments

Parameterized expressions can handle complex arguments including:
- Nested function calls: `${hasField(size(values.items))}`
- Arrays: `${matchesAny(values.type, ['ClusterIP', 'NodePort', 'LoadBalancer'])}`
- Expressions with parentheses: `${inRange(int(values.port), 1, 65535)}`

```yaml
expressions:
  hasMinSize: '$0 > 0'
  inList: '$0 in $1'

rules:
  - expr: '${hasMinSize(size(values.hosts))}'
    desc: "must have at least one host"
  
  - expr: '${inList(values.service.type, ["ClusterIP", "NodePort", "LoadBalancer"])}'
    desc: "service type must be valid"
```

## Benefits

1. **Reduced Duplication**: Define validation logic once, use it many times
2. **Easier Maintenance**: Update validation logic in one place
3. **Better Readability**: Clear, descriptive expression names
4. **Consistency**: Ensure the same validation logic is applied everywhere
5. **Flexibility**: Combine parameterized expressions with regular expressions

## Parameter Rules

- Parameters are positional and use the format `$0`, `$1`, `$2`, etc.
- `$0` is the first argument, `$1` is the second, and so on
- You can use the same parameter multiple times in an expression
- Parameters are replaced with the exact argument text provided

## Comparison: Before and After

### Before (Without Parameterized Expressions)

```yaml
rules:
  - expr: 'has(values.foo.bar)'
    desc: "foo.bar is required"
  
  - expr: 'has(values.baz.qux)'
    desc: "baz.qux is required"
  
  - expr: 'values.service.port >= 1 && values.service.port <= 65535'
    desc: "service port must be valid"
  
  - expr: 'values.service.nodePort >= 30000 && values.service.nodePort <= 32767'
    desc: "nodePort must be valid"
```

### After (With Parameterized Expressions)

```yaml
expressions:
  hasField: 'has($0)'
  inRange: '$0 >= $1 && $0 <= $2'

rules:
  - expr: '${hasField(values.foo.bar)}'
    desc: "foo.bar is required"
  
  - expr: '${hasField(values.baz.qux)}'
    desc: "baz.qux is required"
  
  - expr: '${inRange(values.service.port, 1, 65535)}'
    desc: "service port must be valid"
  
  - expr: '${inRange(values.service.nodePort, 30000, 32767)}'
    desc: "nodePort must be valid"
```

## Complete Example

Here's a complete example showing various parameterized expressions:

```yaml
expressions:
  # Basic field checks
  hasField: 'has($0)'
  hasNonEmpty: 'has($0) && $0 != ""'
  
  # Range validations
  inRange: '$0 >= $1 && $0 <= $2'
  portRange: '$0 >= 1 && $0 <= 65535'
  nodePortRange: '$0 >= 30000 && $0 <= 32767'
  
  # Type checks
  isString: 'type($0) == string'
  isInt: 'type($0) == int'
  isBool: 'type($0) == bool'
  
  # Pattern matching
  matchesPattern: '$0.matches($1)'
  
  # Combined validations
  validPort: 'has($0) && ${portRange($0)}'
  validNodePort: 'has($0) && ${nodePortRange($0)}'

rules:
  # Service validation
  - expr: '${hasField(values.service)}'
    desc: "service must be defined"
  
  - expr: '${validPort(values.service.port)}'
    desc: "service port must be defined and valid"
  
  - expr: '${isString(values.service.type)}'
    desc: "service type must be a string"
  
  # Replica validation
  - expr: '${isInt(values.replicaCount)}'
    desc: "replicaCount must be an integer"
  
  - expr: '${inRange(values.replicaCount, 1, 100)}'
    desc: "replicaCount must be between 1 and 100"
  
  # Image validation
  - expr: '${hasNonEmpty(values.image.repository)}'
    desc: "image repository must be set"
  
  - expr: '${matchesPattern(values.image.tag, "^[a-zA-Z0-9._-]+$")}'
    desc: "image tag must be valid"
  
  # Resource validation
  - expr: '${matchesPattern(values.resources.memory, "^[0-9]+(Mi|Gi)$")}'
    desc: "memory must be in Mi or Gi format"
```

## Tips

1. **Use Descriptive Names**: Choose clear names for your expressions (e.g., `hasField`, `inRange`, `validPort`)
2. **Keep It Simple**: Each expression should do one thing well
3. **Document Complex Logic**: Add comments to explain complex parameterized expressions
4. **Test Thoroughly**: Test your parameterized expressions with various inputs
5. **Combine When Needed**: You can mix parameterized and non-parameterized expressions in the same rules file

