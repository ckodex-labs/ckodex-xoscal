# OSCAL gRPC Service Usage Examples

This document provides usage examples for the OSCAL gRPC services.

## Python gRPC Client

### Installation

```bash
pip install grpcio
pip install grpcio-tools
```

### Basic Usage

```python
import grpc
from oscal.services.v1 import oscal_service_pb2
from oscal.services.v1 import oscal_service_pb2_grpc

# Connect to gRPC server
channel = grpc.insecure_channel('localhost:50051')
stub = oscal_service_pb2_grpc.OscalServiceStub(channel)

# Create a catalog request
request = oscal_service_pb2.GetCatalogRequest()
request.uuid.value = "550e8400-e29b-41d4-a716-446655440000"

# Call the service
try:
    response = stub.GetCatalog(request)
    print(f"Catalog UUID: {response.uuid.value}")
    print(f"Catalog Title: {response.metadata.title}")
    print(f"Number of controls: {len(response.controls)}")
except grpc.RpcError as e:
    print(f"gRPC error: {e.code()}")
    print(f"Details: {e.details()}")

# Close channel
channel.close()
```

### Using LanceDB Storage

```python
import asyncio
from src.mcp_server_for_oscal.storage.lancedb_store import LanceDBStore, StoreConfig, SearchOptions

async def main():
    # Initialize LanceDB store
    config = StoreConfig(
        uri="./data/lancedb",
        embedding_model="all-MiniLM-L6-v2"
    )
    store = LanceDBStore(config)
    
    # Use async context manager
    async with store:
        # Store a catalog
        await store.store(
            model_type="catalog",
            uuid="550e8400-e29b-41d4-a716-446655440000",
            title="Example Catalog",
            content='{"uuid": "550e8400-e29b-41d4-a716-446655440000"}',
            metadata={"version": "1.0.0"}
        )
        
        # Retrieve a catalog
        catalog = await store.get("catalog", "550e8400-e29b-41d4-a716-446655440000")
        print(f"Retrieved: {catalog['title']}")
        
        # Search catalogs
        options = SearchOptions(limit=5, include_metadata=True)
        results = await store.search("access control", ["catalog"], options)
        for result in results:
            print(f"Found: {result['title']} (score: {result['score']})")
        
        # Batch store
        records = [
            {
                "model_type": "catalog",
                "uuid": f"uuid-{i}",
                "title": f"Catalog {i}",
                "content": "{}",
                "metadata": {"index": i}
            }
            for i in range(10)
        ]
        uuids = await store.batch_store(records)
        print(f"Stored {len(uuids)} records")

asyncio.run(main())
```

### Streaming Operations

```python
# List all catalogs
request = oscal_service_pb2.ListCatalogsRequest()
request.page_size = 10

for response in stub.ListCatalogs(request):
    print(f"Catalog: {response.uuid.value}")
    print(f"Title: {response.metadata.title}")

# Search catalogs
search_request = oscal_service_pb2.SearchCatalogsRequest()
search_request.query = "access control"
search_request.limit = 5

for response in stub.SearchCatalogs(search_request):
    print(f"Found: {response.title}")
    print(f"Score: {response.score}")
```

## Go gRPC Client

### Installation

```bash
go get google.golang.org/grpc
go get google.golang.org/protobuf
```

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    
    oscalv1 "path/to/gen/go/services/v1"
)

func main() {
    // Connect to gRPC server
    conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer conn.Close()
    
    client := oscalv1.NewOscalServiceClient(conn)
    
    ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
    defer cancel()
    
    // Get a catalog
    req := &oscalv1.GetCatalogRequest{
        Uuid: &oscalv1.UUID{Value: "550e8400-e29b-41d4-a716-446655440000"},
    }
    
    resp, err := client.GetCatalog(ctx, req)
    if err != nil {
        log.Fatalf("Failed to get catalog: %v", err)
    }
    
    fmt.Printf("Catalog UUID: %s\n", resp.Uuid.Value)
    fmt.Printf("Catalog Title: %s\n", resp.Metadata.Title)
}
```

### Streaming Operations

```go
// List all catalogs
req := &oscalv1.ListCatalogsRequest{
    PageSize: 10,
}

stream, err := client.ListCatalogs(ctx, req)
if err != nil {
    log.Fatalf("Failed to list catalogs: %v", err)
}

for {
    resp, err := stream.Recv()
    if err == io.EOF {
        break
    }
    if err != nil {
        log.Fatalf("Stream error: %v", err)
    }
    fmt.Printf("Catalog: %s\n", resp.Uuid.Value)
}
```

## Java gRPC Client

### Installation

```xml
<!-- pom.xml -->
<dependency>
    <groupId>io.grpc</groupId>
    <artifactId>grpc-netty-shaded</artifactId>
    <version>1.60.0</version>
</dependency>
<dependency>
    <groupId>io.grpc</groupId>
    <artifactId>grpc-protobuf</artifactId>
    <version>1.60.0</version>
</dependency>
<dependency>
    <groupId>io.grpc</groupId>
    <artifactId>grpc-stub</artifactId>
    <version>1.60.0</version>
</dependency>
```

### Basic Usage

```java
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import com.oscal.services.v1.OscalServiceGrpc;
import com.oscal.services.v1.OscalServiceOuterClass;

public class OscalClient {
    public static void main(String[] args) {
        // Connect to gRPC server
        ManagedChannel channel = ManagedChannelBuilder.forAddress("localhost", 50051)
            .usePlaintext()
            .build();
        
        try {
            OscalServiceGrpc.OscalServiceBlockingStub stub = OscalServiceGrpc.newBlockingStub(channel);
            
            // Get a catalog
            OscalServiceOuterClass.UUID uuid = OscalServiceOuterClass.UUID.newBuilder()
                .setValue("550e8400-e29b-41d4-a716-446655440000")
                .build();
            
            OscalServiceOuterClass.GetCatalogRequest request = OscalServiceOuterClass.GetCatalogRequest.newBuilder()
                .setUuid(uuid)
                .build();
            
            OscalServiceOuterClass.Catalog response = stub.getCatalog(request);
            
            System.out.println("Catalog UUID: " + response.getUuid().getValue());
            System.out.println("Catalog Title: " + response.getMetadata().getTitle());
            
        } finally {
            channel.shutdown();
        }
    }
}
```

## TypeScript gRPC Client

### Installation

```bash
npm install @grpc/grpc-js @grpc/proto-loader
```

### Basic Usage

```typescript
import grpc from '@grpc/grpc-js';
import protoLoader from '@grpc/proto-loader';

// Load proto file
const PROTO_PATH = '../proto/oscal/services/v1/oscal_service.proto';
const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
    keepCase: true,
    longs: String,
    enums: String,
    defaults: true,
    oneofs: true
});

const oscalProto = grpc.loadPackageDefinition(packageDefinition).oscal.services.v1;

// Connect to gRPC server
const client = new oscalProto.OscalService(
    'localhost:50051',
    grpc.credentials.createInsecure()
);

// Get a catalog
const request = {
    uuid: { value: '550e8400-e29b-41d4-a716-446655440000' }
};

client.GetCatalog(request, (error, response) => {
    if (error) {
        console.error('Error:', error);
        return;
    }
    console.log('Catalog UUID:', response.uuid.value);
    console.log('Catalog Title:', response.metadata.title);
});
```

## LanceDB Storage Integration

### Python Integration with gRPC Server

```python
from src.mcp_server_for_oscal.grpc.server import OscalGrpcServer
from src.mcp_server_for_oscal.storage.lancedb_store import LanceDBStore, StoreConfig

async def run_server():
    # Initialize LanceDB store
    config = StoreConfig(uri="./data/lancedb")
    store = LanceDBStore(config)
    
    # Start gRPC server with LanceDB backend
    server = OscalGrpcServer(store=store)
    await server.start(port=50051)
    
    # Keep server running
    await server.wait_for_termination()

if __name__ == "__main__":
    import asyncio
    asyncio.run(run_server())
```

### Advanced Search with LanceDB

```python
from src.mcp_server_for_oscal.storage.lancedb_store import LanceDBStore, StoreConfig, SearchOptions

async def advanced_search():
    config = StoreConfig(uri="./data/lancedb")
    store = LanceDBStore(config)
    
    async with store:
        # Search with options
        options = SearchOptions(
            limit=10,
            metric="cosine",
            filter_str="metadata.version = '1.0.0'",
            include_metadata=True,
            include_content=False
        )
        
        results = await store.search(
            query="security controls",
            model_types=["catalog", "profile"],
            options=options
        )
        
        for result in results:
            print(f"Title: {result['title']}")
            print(f"Score: {result['score']}")
            print(f"Metadata: {result['metadata']}")
        
        # Get table statistics
        stats = await store.get_table_stats("catalog")
        print(f"Catalog table has {stats['num_rows']} rows")
        
        # Export table
        success = await store.export_table(
            model_type="catalog",
            output_path="./catalog_export.parquet",
            format="parquet"
        )
        print(f"Export successful: {success}")

import asyncio
asyncio.run(advanced_search())
```

## Error Handling

### Python

```python
import grpc
from grpc import StatusCode

try:
    response = stub.GetCatalog(request)
except grpc.RpcError as e:
    if e.code() == StatusCode.NOT_FOUND:
        print("Catalog not found")
    elif e.code() == StatusCode.INVALID_ARGUMENT:
        print("Invalid request")
    elif e.code() == StatusCode.UNAVAILABLE:
        print("Server unavailable")
    else:
        print(f"Unexpected error: {e.code()}")
```

### Go

```go
import (
    "status"
    "google.golang.org/grpc/codes"
)

resp, err := client.GetCatalog(ctx, req)
if err != nil {
    st, ok := status.FromError(err)
    if !ok {
        log.Fatalf("Unknown error: %v", err)
    }
    
    switch st.Code() {
    case codes.NotFound:
        log.Println("Catalog not found")
    case codes.InvalidArgument:
        log.Println("Invalid request")
    case codes.Unavailable:
        log.Println("Server unavailable")
    default:
        log.Fatalf("Unexpected error: %s", st.Code())
    }
}
```

## Authentication

### TLS Configuration

```python
import grpc
import ssl

# Create SSL credentials
creds = grpc.ssl_channel_credentials(
    root_certificates=open('ca.crt', 'rb').read(),
    private_key=open('client.key', 'rb').read(),
    certificate_chain=open('client.crt', 'rb').read()
)

# Connect with TLS
channel = grpc.secure_channel('localhost:50051', creds)
```

### Token-based Authentication

```python
from grpc import intercept_channel
from grpc_interceptor import ClientInterceptor

class AuthInterceptor(ClientInterceptor):
    def __init__(self, token):
        self.token = token
    
    def intercept(
        self,
        method,
        request,
        call_details,
        response_iterator
    ):
        call_details.metadata.add('authorization', f'Bearer {self.token}')
        return response_iterator

# Add interceptor
interceptor = AuthInterceptor('your-token-here')
channel = intercept_channel(grpc.insecure_channel('localhost:50051'), interceptor)
```

## Performance Tips

1. **Use connection pooling**: Reuse gRPC channels instead of creating new ones for each request
2. **Enable compression**: Use message compression for large payloads
3. **Streaming for bulk operations**: Use streaming APIs for listing or searching large datasets
4. **Batch operations**: Use batch store in LanceDB for bulk inserts
5. **Caching**: Cache frequently accessed documents to reduce database load

## Additional Resources

- [gRPC Documentation](https://grpc.io/docs/)
- [LanceDB Documentation](https://lancedb.github.io/lancedb/)
- [OSCAL gRPC Service Definition](../proto/oscal/services/v1/oscal_service.proto)
