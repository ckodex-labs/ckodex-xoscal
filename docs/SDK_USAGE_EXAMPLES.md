# OSCAL SDK Usage Examples

This document provides usage examples for the generated OSCAL SDKs across multiple languages.

## Python SDK

### Installation

```bash
# Add the generated Python SDK to your path
export PYTHONPATH="/path/to/mcp-server-for-oscal/proto/oscal/gen/python:$PYTHONPATH"
```

### Basic Usage

```python
from common.v1 import common_pb2
from catalog.v1 import catalog_pb2

# Create a UUID
uuid = common_pb2.UUID()
uuid.value = "550e8400-e29b-41d4-a716-446655440000"

# Create metadata
metadata = common_pb2.Metadata()
metadata.title = "Example OSCAL Document"
metadata.version = "1.0.0"
metadata.last_modified = "2024-01-01T00:00:00Z"

# Create a catalog
catalog = catalog_pb2.Catalog()
catalog.uuid.value = uuid.value
catalog.metadata.CopyFrom(metadata)

# Add a control
control = catalog_pb2.Control()
control.id.value = "ac-1"
control.title.value = "Access Control Policy and Procedures"

catalog.controls.append(control)

# Serialize to bytes
serialized = catalog.SerializeToString()

# Deserialize from bytes
catalog_copy = catalog_pb2.Catalog()
catalog_copy.ParseFromString(serialized)

# Convert to JSON
import json
from google.protobuf.json_format import MessageToJson
catalog_json = MessageToJson(catalog)
catalog_dict = json.loads(catalog_json)
```

### Working with Profile

```python
from oscal_profile.v1 import profile_pb2

# Create a profile
profile = profile_pb2.Profile()
profile.uuid.value = "550e8400-e29b-41d4-a716-446655440000"

# Add imports
import_profile = profile_pb2.Import()
import_profile.href.value = "https://example.com/catalog.json"
profile.imports.append(import_profile)

# Set controls
set_control = profile_pb2.SetControl()
set_control.id.value = "ac-1"
set_control.optional = False
profile.set_controls.append(set_control)
```

### Working with System Security Plan (SSP)

```python
from ssp.v1 import ssp_pb2

# Create an SSP
ssp = ssp_pb2.SystemSecurityPlan()
ssp.uuid.value = "550e8400-e29b-41d4-a716-446655440000"

# Set system characteristics
system_characteristics = ssp_pb2.SystemCharacteristics()
system_characteristics.system_name = "Example System"
system_characteristics.system_short_name = "Example"
ssp.system_characteristics.CopyFrom(system_characteristics)

# Add controls implementation
control_implementation = ssp_pb2.ControlImplementation()
control_implementation.uuid.value = "550e8400-e29b-41d4-a716-446655440001"
ssp.control_implementations.append(control_implementation)
```

## Go SDK

### Installation

```bash
# Ensure the generated Go code is in your module
go mod init example.com/oscal-client
# Add the path to your go.mod or use replace directive
```

### Basic Usage

```go
package main

import (
    "fmt"
    "encoding/json"
    "path/to/gen/go/common/v1"
    "path/to/gen/go/catalog/v1"
)

func main() {
    // Create a UUID
    uuid := &common.UUID{
        Value: "550e8400-e29b-41d4-a716-446655440000",
    }

    // Create metadata
    metadata := &common.Metadata{
        Title:        "Example OSCAL Document",
        Version:      "1.0.0",
        LastModified: "2024-01-01T00:00:00Z",
    }

    // Create a catalog
    catalog := &catalog.Catalog{
        Uuid:     uuid,
        Metadata: metadata,
    }

    // Add a control
    control := &catalog.Control{
        Id:    &common.StringValue{Value: "ac-1"},
        Title: &common.StringValue{Value: "Access Control Policy and Procedures"},
    }
    catalog.Controls = append(catalog.Controls, control)

    // Serialize to bytes
    data, err := proto.Marshal(catalog)
    if err != nil {
        panic(err)
    }

    // Deserialize from bytes
    catalogCopy := &catalog.Catalog{}
    err = proto.Unmarshal(data, catalogCopy)
    if err != nil {
        panic(err)
    }

    // Convert to JSON
    jsonData, err := json.MarshalIndent(catalog, "", "  ")
    if err != nil {
        panic(err)
    }
    fmt.Println(string(jsonData))
}
```

## Rust SDK

### Installation

```toml
# Cargo.toml
[dependencies]
oscal = { path = "../proto/oscal/gen/rust" }
prost = "0.12"
```

### Basic Usage

```rust
use oscal::common::v1::Uuid;
use oscal::catalog::v1::{Catalog, Control};
use prost::Message;

fn main() {
    // Create a UUID
    let uuid = Uuid {
        value: "550e8400-e29b-41d4-a716-446655440000".to_string(),
    };

    // Create a catalog
    let catalog = Catalog {
        uuid: Some(uuid),
        metadata: None,
        controls: vec![
            Control {
                id: Some("ac-1".to_string()),
                title: Some("Access Control Policy and Procedures".to_string()),
                ..Default::default()
            }
        ],
        ..Default::default()
    };

    // Serialize to bytes
    let mut buf = Vec::new();
    catalog.encode(&mut buf).unwrap();

    // Deserialize from bytes
    let catalog_copy = Catalog::decode(&*buf).unwrap();

    // Convert to JSON
    let json = serde_json::to_string_pretty(&catalog).unwrap();
    println!("{}", json);
}
```

## Java SDK

### Installation

```xml
<!-- pom.xml -->
<dependency>
    <groupId>com.oscal</groupId>
    <artifactId>oscal-java</artifactId>
    <version>1.0.0</version>
</dependency>
```

### Basic Usage

```java
import com.oscal.common.v1.CommonProtos;
import com.oscal.catalog.v1.CatalogProtos;

public class OscalExample {
    public static void main(String[] args) {
        // Create a UUID
        CommonProtos.UUID uuid = CommonProtos.UUID.newBuilder()
            .setValue("550e8400-e29b-41d4-a716-446655440000")
            .build();

        // Create a catalog
        CatalogProtos.Catalog catalog = CatalogProtos.Catalog.newBuilder()
            .setUuid(uuid)
            .addControls(CatalogProtos.Control.newBuilder()
                .setId(CommonProtos.StringValue.newBuilder()
                    .setValue("ac-1")
                    .build())
                .setTitle(CommonProtos.StringValue.newBuilder()
                    .setValue("Access Control Policy and Procedures")
                    .build())
                .build())
            .build();

        // Serialize to bytes
        byte[] data = catalog.toByteArray();

        // Deserialize from bytes
        CatalogProtos.Catalog catalogCopy = CatalogProtos.Catalog.parseFrom(data);

        // Convert to JSON
        String json = com.google.protobuf.util.JsonFormat.printer().print(catalog);
        System.out.println(json);
    }
}
```

## TypeScript/JavaScript SDK

### Installation

```bash
npm install ../proto/oscal/gen/ts
```

### Basic Usage

```typescript
import { UUID } from './gen/common/v1/common_pb';
import { Catalog, Control } from './gen/catalog/v1/catalog_pb';

// Create a UUID
const uuid = new UUID();
uuid.setValue("550e8400-e29b-41d4-a716-446655440000");

// Create a catalog
const catalog = new Catalog();
catalog.setUuid(uuid);

// Add a control
const control = new Control();
control.setId("ac-1");
control.setTitle("Access Control Policy and Procedures");
catalog.addControls(control);

// Serialize to bytes
const data = catalog.serializeBinary();

// Deserialize from bytes
const catalogCopy = Catalog.deserializeBinary(data);

// Convert to JSON
const json = catalog.toObject();
console.log(JSON.stringify(json, null, 2));
```

## C# SDK

### Installation

```xml
<!-- .csproj -->
<ItemGroup>
  <ProjectReference Include="../proto/oscal/gen/csharp/oscal.csproj" />
</ItemGroup>
```

### Basic Usage

```csharp
using Oscal.Common.V1;
using Oscal.Catalog.V1;

class Program {
    static void Main() {
        // Create a UUID
        var uuid = new UUID { Value = "550e8400-e29b-41d4-a716-446655440000" };

        // Create a catalog
        var catalog = new Catalog {
            Uuid = uuid,
        };

        // Add a control
        var control = new Control {
            Id = "ac-1",
            Title = "Access Control Policy and Procedures"
        };
        catalog.Controls.Add(control);

        // Serialize to bytes
        byte[] data = catalog.ToByteArray();

        // Deserialize from bytes
        var catalogCopy = Catalog.Parser.ParseFrom(data);

        // Convert to JSON
        string json = catalog.ToString();
        Console.WriteLine(json);
    }
}
```

## Swift SDK

### Installation

```swift
// Package.swift
dependencies: [
    .package(path: "../proto/oscal/gen/swift")
]
```

### Basic Usage

```swift
import Oscal_Common_V1
import Oscal_Catalog_V1

// Create a UUID
var uuid = UUID()
uuid.value = "550e8400-e29b-41d4-a716-446655440000"

// Create a catalog
var catalog = Catalog()
catalog.uuid = uuid

// Add a control
var control = Control()
control.id = "ac-1"
control.title = "Access Control Policy and Procedures"
catalog.controls.append(control)

// Serialize to bytes
let data = try catalog.serializedData()

// Deserialize from bytes
let catalogCopy = try Catalog(serializedData: data)

// Convert to JSON
let json = try catalog.jsonString()
print(json)
```

## Common Patterns

### Validation

All generated SDKs support validation through the protobuf runtime. Always validate data before use:

```python
# Python
try:
    catalog.ParseFromString(serialized_data)
except DecodeError as e:
    print(f"Invalid data: {e}")
```

### Error Handling

```python
# Python
from google.protobuf import message

try:
    catalog.ParseFromString(data)
except message.DecodeError as e:
    # Handle decode error
    pass
```

### Version Compatibility

The generated SDKs use protobuf 6.31.1 runtime. Ensure your runtime matches this version:

```bash
pip install protobuf==6.31.1
```

## Additional Resources

- [OSCAL Specification](https://pages.nist.gov/OSCAL/)
- [Protocol Buffers Documentation](https://protobuf.dev/)
- [Buf Documentation](https://buf.build/docs)
