# Zig SDK for OSCAL Protobuf

## Status

Zig SDK generation is currently **not supported** through standard protobuf tooling.

## Why?

There is no standard `protoc-gen-zig` plugin available in the Buf plugin registry or in the broader protobuf ecosystem. Unlike other languages (Go, Java, C#, Python, TypeScript, Swift, Rust), Zig does not have an officially maintained protobuf code generator.

## Potential Approaches

### Option 1: Custom Protoc Plugin (Recommended for Production)

Create a custom `protoc-gen-zig` plugin:

```zig
// This would be a Zig program that reads protobuf FileDescriptorSet
// and generates Zig code

const std = @import("std");

pub fn main() !void {
    // 1. Read protobuf FileDescriptorSet from stdin
    // 2. Parse and analyze the descriptor
    // 3. Generate Zig struct definitions
    // 4. Output generated Zig code to stdout
}
```

**Pros:**

- Full control over generated code
- Can generate idiomatic Zig code
- Can be integrated with Buf build pipeline

**Cons:**

- Requires significant development effort
- Must maintain compatibility with protobuf schema changes
- Need to handle all protobuf features (nested messages, enums, oneofs, etc.)

### Option 2: Manual Zig Bindings

Create manual Zig struct definitions based on the protobuf schema:

```zig
// Example: oscal/common/v1/common.zig

const std = @import("std");

pub const UUID = struct {
    value: []const u8,
};

pub const Link = struct {
    href: []const u8,
    rel: ?[]const u8 = null,
    media_type: ?[]const u8 = null,
    // ... other fields
};

pub const Property = struct {
    name: []const u8,
    value: []const u8,
    ns: ?[]const u8 = null,
    class: ?[]const u8 = null,
    // ... other fields
};
```

**Pros:**

- Full control over API design
- Can use Zig-specific features effectively
- No dependency on external tools

**Cons:**

- Manual maintenance burden
- Must manually sync with protobuf schema changes
- No automatic updates when proto files change

### Option 3: C Interop via Generated C Headers

Use protobuf C library with Zig C interop:

1. Generate C headers using `protoc --c_out=.`
2. Use Zig's `@cImport` to include generated headers:

```zig
const c = @cImport({
    @cInclude("common.pb-c.h");
});

pub const UUID = c.Oscal_Common_V1_UUID;
```

**Pros:**

- Leverages existing protobuf C library
- Can use standard protobuf serialization
- Less development effort

**Cons:**

- Not idiomatic Zig code
- Requires linking against C protobuf library
- C ABI compatibility concerns

### Option 4: JSON-based Approach

Parse OSCAL JSON directly in Zig without protobuf:

```zig
const std = @import("std");

pub const Catalog = struct {
    uuid: []const u8,
    metadata: Metadata,
    back_matter: ?BackMatter = null,
    // ... other fields
    
    pub fn fromJson(allocator: std.mem.Allocator, json: []const u8) !Catalog {
        // Use std.json to parse
    }
};
```

**Pros:**

- No protobuf dependency
- Works directly with OSCAL JSON format
- Zig has excellent JSON parsing support

**Cons:**

- Loses protobuf benefits (binary efficiency, schema evolution)
- Different from other language SDKs
- Performance trade-offs

## Recommendation

For immediate needs, **Option 4 (JSON-based approach)** is recommended:

- OSCAL is primarily a JSON-based standard
- Zig has excellent JSON support
- No protobuf dependency required
- Can be implemented quickly

For long-term use, **Option 1 (Custom Protoc Plugin)** would be ideal but requires significant development effort.

## Current Implementation Status

- ❌ No automatic Zig SDK generation
- ✅ All other languages (Go, Java, C#, Python, TypeScript, Swift, Rust) have generated SDKs
- ✅ OpenAPI and JSON Schema specifications available

## Next Steps

If Zig SDK is required, choose one of the approaches above and implement accordingly. The JSON-based approach (Option 4) is the quickest path to Zig support for OSCAL.
