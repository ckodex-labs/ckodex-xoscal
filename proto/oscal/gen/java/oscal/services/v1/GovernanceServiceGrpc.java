package oscal.services.v1;

import static io.grpc.MethodDescriptor.generateFullMethodName;

/**
 * <pre>
 * GovernanceService provides advanced governance, ingestion, and semantic
 * search operations over the OSCALify knowledge graph.
 * </pre>
 */
@io.grpc.stub.annotations.GrpcGenerated
public final class GovernanceServiceGrpc {

  private GovernanceServiceGrpc() {}

  public static final java.lang.String SERVICE_NAME = "oscal.services.v1.GovernanceService";

  // Static method descriptors that strictly reflect the proto.
  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.CreateEntityRequest,
      oscal.services.v1.GovernanceServiceOuterClass.CreateEntityResponse> getCreateEntityMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "CreateEntity",
      requestType = oscal.services.v1.GovernanceServiceOuterClass.CreateEntityRequest.class,
      responseType = oscal.services.v1.GovernanceServiceOuterClass.CreateEntityResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.CreateEntityRequest,
      oscal.services.v1.GovernanceServiceOuterClass.CreateEntityResponse> getCreateEntityMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.CreateEntityRequest, oscal.services.v1.GovernanceServiceOuterClass.CreateEntityResponse> getCreateEntityMethod;
    if ((getCreateEntityMethod = GovernanceServiceGrpc.getCreateEntityMethod) == null) {
      synchronized (GovernanceServiceGrpc.class) {
        if ((getCreateEntityMethod = GovernanceServiceGrpc.getCreateEntityMethod) == null) {
          GovernanceServiceGrpc.getCreateEntityMethod = getCreateEntityMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.GovernanceServiceOuterClass.CreateEntityRequest, oscal.services.v1.GovernanceServiceOuterClass.CreateEntityResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "CreateEntity"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.CreateEntityRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.CreateEntityResponse.getDefaultInstance()))
              .setSchemaDescriptor(new GovernanceServiceMethodDescriptorSupplier("CreateEntity"))
              .build();
        }
      }
    }
    return getCreateEntityMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GetEntityRequest,
      oscal.services.v1.GovernanceServiceOuterClass.GetEntityResponse> getGetEntityMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GetEntity",
      requestType = oscal.services.v1.GovernanceServiceOuterClass.GetEntityRequest.class,
      responseType = oscal.services.v1.GovernanceServiceOuterClass.GetEntityResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GetEntityRequest,
      oscal.services.v1.GovernanceServiceOuterClass.GetEntityResponse> getGetEntityMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GetEntityRequest, oscal.services.v1.GovernanceServiceOuterClass.GetEntityResponse> getGetEntityMethod;
    if ((getGetEntityMethod = GovernanceServiceGrpc.getGetEntityMethod) == null) {
      synchronized (GovernanceServiceGrpc.class) {
        if ((getGetEntityMethod = GovernanceServiceGrpc.getGetEntityMethod) == null) {
          GovernanceServiceGrpc.getGetEntityMethod = getGetEntityMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.GovernanceServiceOuterClass.GetEntityRequest, oscal.services.v1.GovernanceServiceOuterClass.GetEntityResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GetEntity"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.GetEntityRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.GetEntityResponse.getDefaultInstance()))
              .setSchemaDescriptor(new GovernanceServiceMethodDescriptorSupplier("GetEntity"))
              .build();
        }
      }
    }
    return getGetEntityMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.UpdateEntityRequest,
      oscal.services.v1.GovernanceServiceOuterClass.UpdateEntityResponse> getUpdateEntityMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "UpdateEntity",
      requestType = oscal.services.v1.GovernanceServiceOuterClass.UpdateEntityRequest.class,
      responseType = oscal.services.v1.GovernanceServiceOuterClass.UpdateEntityResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.UpdateEntityRequest,
      oscal.services.v1.GovernanceServiceOuterClass.UpdateEntityResponse> getUpdateEntityMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.UpdateEntityRequest, oscal.services.v1.GovernanceServiceOuterClass.UpdateEntityResponse> getUpdateEntityMethod;
    if ((getUpdateEntityMethod = GovernanceServiceGrpc.getUpdateEntityMethod) == null) {
      synchronized (GovernanceServiceGrpc.class) {
        if ((getUpdateEntityMethod = GovernanceServiceGrpc.getUpdateEntityMethod) == null) {
          GovernanceServiceGrpc.getUpdateEntityMethod = getUpdateEntityMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.GovernanceServiceOuterClass.UpdateEntityRequest, oscal.services.v1.GovernanceServiceOuterClass.UpdateEntityResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "UpdateEntity"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.UpdateEntityRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.UpdateEntityResponse.getDefaultInstance()))
              .setSchemaDescriptor(new GovernanceServiceMethodDescriptorSupplier("UpdateEntity"))
              .build();
        }
      }
    }
    return getUpdateEntityMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.ListEntitiesRequest,
      oscal.services.v1.GovernanceServiceOuterClass.ListEntitiesResponse> getListEntitiesMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "ListEntities",
      requestType = oscal.services.v1.GovernanceServiceOuterClass.ListEntitiesRequest.class,
      responseType = oscal.services.v1.GovernanceServiceOuterClass.ListEntitiesResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.ListEntitiesRequest,
      oscal.services.v1.GovernanceServiceOuterClass.ListEntitiesResponse> getListEntitiesMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.ListEntitiesRequest, oscal.services.v1.GovernanceServiceOuterClass.ListEntitiesResponse> getListEntitiesMethod;
    if ((getListEntitiesMethod = GovernanceServiceGrpc.getListEntitiesMethod) == null) {
      synchronized (GovernanceServiceGrpc.class) {
        if ((getListEntitiesMethod = GovernanceServiceGrpc.getListEntitiesMethod) == null) {
          GovernanceServiceGrpc.getListEntitiesMethod = getListEntitiesMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.GovernanceServiceOuterClass.ListEntitiesRequest, oscal.services.v1.GovernanceServiceOuterClass.ListEntitiesResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "ListEntities"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.ListEntitiesRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.ListEntitiesResponse.getDefaultInstance()))
              .setSchemaDescriptor(new GovernanceServiceMethodDescriptorSupplier("ListEntities"))
              .build();
        }
      }
    }
    return getListEntitiesMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.CreateSnapshotRequest,
      oscal.services.v1.GovernanceServiceOuterClass.CreateSnapshotResponse> getCreateSnapshotMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "CreateSnapshot",
      requestType = oscal.services.v1.GovernanceServiceOuterClass.CreateSnapshotRequest.class,
      responseType = oscal.services.v1.GovernanceServiceOuterClass.CreateSnapshotResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.CreateSnapshotRequest,
      oscal.services.v1.GovernanceServiceOuterClass.CreateSnapshotResponse> getCreateSnapshotMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.CreateSnapshotRequest, oscal.services.v1.GovernanceServiceOuterClass.CreateSnapshotResponse> getCreateSnapshotMethod;
    if ((getCreateSnapshotMethod = GovernanceServiceGrpc.getCreateSnapshotMethod) == null) {
      synchronized (GovernanceServiceGrpc.class) {
        if ((getCreateSnapshotMethod = GovernanceServiceGrpc.getCreateSnapshotMethod) == null) {
          GovernanceServiceGrpc.getCreateSnapshotMethod = getCreateSnapshotMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.GovernanceServiceOuterClass.CreateSnapshotRequest, oscal.services.v1.GovernanceServiceOuterClass.CreateSnapshotResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "CreateSnapshot"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.CreateSnapshotRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.CreateSnapshotResponse.getDefaultInstance()))
              .setSchemaDescriptor(new GovernanceServiceMethodDescriptorSupplier("CreateSnapshot"))
              .build();
        }
      }
    }
    return getCreateSnapshotMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GetSnapshotRequest,
      oscal.services.v1.GovernanceServiceOuterClass.GetSnapshotResponse> getGetSnapshotMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GetSnapshot",
      requestType = oscal.services.v1.GovernanceServiceOuterClass.GetSnapshotRequest.class,
      responseType = oscal.services.v1.GovernanceServiceOuterClass.GetSnapshotResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GetSnapshotRequest,
      oscal.services.v1.GovernanceServiceOuterClass.GetSnapshotResponse> getGetSnapshotMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GetSnapshotRequest, oscal.services.v1.GovernanceServiceOuterClass.GetSnapshotResponse> getGetSnapshotMethod;
    if ((getGetSnapshotMethod = GovernanceServiceGrpc.getGetSnapshotMethod) == null) {
      synchronized (GovernanceServiceGrpc.class) {
        if ((getGetSnapshotMethod = GovernanceServiceGrpc.getGetSnapshotMethod) == null) {
          GovernanceServiceGrpc.getGetSnapshotMethod = getGetSnapshotMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.GovernanceServiceOuterClass.GetSnapshotRequest, oscal.services.v1.GovernanceServiceOuterClass.GetSnapshotResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GetSnapshot"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.GetSnapshotRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.GetSnapshotResponse.getDefaultInstance()))
              .setSchemaDescriptor(new GovernanceServiceMethodDescriptorSupplier("GetSnapshot"))
              .build();
        }
      }
    }
    return getGetSnapshotMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.ListSnapshotsRequest,
      oscal.services.v1.GovernanceServiceOuterClass.ListSnapshotsResponse> getListSnapshotsMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "ListSnapshots",
      requestType = oscal.services.v1.GovernanceServiceOuterClass.ListSnapshotsRequest.class,
      responseType = oscal.services.v1.GovernanceServiceOuterClass.ListSnapshotsResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.ListSnapshotsRequest,
      oscal.services.v1.GovernanceServiceOuterClass.ListSnapshotsResponse> getListSnapshotsMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.ListSnapshotsRequest, oscal.services.v1.GovernanceServiceOuterClass.ListSnapshotsResponse> getListSnapshotsMethod;
    if ((getListSnapshotsMethod = GovernanceServiceGrpc.getListSnapshotsMethod) == null) {
      synchronized (GovernanceServiceGrpc.class) {
        if ((getListSnapshotsMethod = GovernanceServiceGrpc.getListSnapshotsMethod) == null) {
          GovernanceServiceGrpc.getListSnapshotsMethod = getListSnapshotsMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.GovernanceServiceOuterClass.ListSnapshotsRequest, oscal.services.v1.GovernanceServiceOuterClass.ListSnapshotsResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "ListSnapshots"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.ListSnapshotsRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.ListSnapshotsResponse.getDefaultInstance()))
              .setSchemaDescriptor(new GovernanceServiceMethodDescriptorSupplier("ListSnapshots"))
              .build();
        }
      }
    }
    return getListSnapshotsMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.CreateReleaseRequest,
      oscal.services.v1.GovernanceServiceOuterClass.CreateReleaseResponse> getCreateReleaseMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "CreateRelease",
      requestType = oscal.services.v1.GovernanceServiceOuterClass.CreateReleaseRequest.class,
      responseType = oscal.services.v1.GovernanceServiceOuterClass.CreateReleaseResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.CreateReleaseRequest,
      oscal.services.v1.GovernanceServiceOuterClass.CreateReleaseResponse> getCreateReleaseMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.CreateReleaseRequest, oscal.services.v1.GovernanceServiceOuterClass.CreateReleaseResponse> getCreateReleaseMethod;
    if ((getCreateReleaseMethod = GovernanceServiceGrpc.getCreateReleaseMethod) == null) {
      synchronized (GovernanceServiceGrpc.class) {
        if ((getCreateReleaseMethod = GovernanceServiceGrpc.getCreateReleaseMethod) == null) {
          GovernanceServiceGrpc.getCreateReleaseMethod = getCreateReleaseMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.GovernanceServiceOuterClass.CreateReleaseRequest, oscal.services.v1.GovernanceServiceOuterClass.CreateReleaseResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "CreateRelease"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.CreateReleaseRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.CreateReleaseResponse.getDefaultInstance()))
              .setSchemaDescriptor(new GovernanceServiceMethodDescriptorSupplier("CreateRelease"))
              .build();
        }
      }
    }
    return getCreateReleaseMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.ListReleasesRequest,
      oscal.services.v1.GovernanceServiceOuterClass.ListReleasesResponse> getListReleasesMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "ListReleases",
      requestType = oscal.services.v1.GovernanceServiceOuterClass.ListReleasesRequest.class,
      responseType = oscal.services.v1.GovernanceServiceOuterClass.ListReleasesResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.ListReleasesRequest,
      oscal.services.v1.GovernanceServiceOuterClass.ListReleasesResponse> getListReleasesMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.ListReleasesRequest, oscal.services.v1.GovernanceServiceOuterClass.ListReleasesResponse> getListReleasesMethod;
    if ((getListReleasesMethod = GovernanceServiceGrpc.getListReleasesMethod) == null) {
      synchronized (GovernanceServiceGrpc.class) {
        if ((getListReleasesMethod = GovernanceServiceGrpc.getListReleasesMethod) == null) {
          GovernanceServiceGrpc.getListReleasesMethod = getListReleasesMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.GovernanceServiceOuterClass.ListReleasesRequest, oscal.services.v1.GovernanceServiceOuterClass.ListReleasesResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "ListReleases"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.ListReleasesRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.ListReleasesResponse.getDefaultInstance()))
              .setSchemaDescriptor(new GovernanceServiceMethodDescriptorSupplier("ListReleases"))
              .build();
        }
      }
    }
    return getListReleasesMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.IngestRequirementsRequest,
      oscal.services.v1.GovernanceServiceOuterClass.IngestRequirementsResponse> getIngestRequirementsMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "IngestRequirements",
      requestType = oscal.services.v1.GovernanceServiceOuterClass.IngestRequirementsRequest.class,
      responseType = oscal.services.v1.GovernanceServiceOuterClass.IngestRequirementsResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.IngestRequirementsRequest,
      oscal.services.v1.GovernanceServiceOuterClass.IngestRequirementsResponse> getIngestRequirementsMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.IngestRequirementsRequest, oscal.services.v1.GovernanceServiceOuterClass.IngestRequirementsResponse> getIngestRequirementsMethod;
    if ((getIngestRequirementsMethod = GovernanceServiceGrpc.getIngestRequirementsMethod) == null) {
      synchronized (GovernanceServiceGrpc.class) {
        if ((getIngestRequirementsMethod = GovernanceServiceGrpc.getIngestRequirementsMethod) == null) {
          GovernanceServiceGrpc.getIngestRequirementsMethod = getIngestRequirementsMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.GovernanceServiceOuterClass.IngestRequirementsRequest, oscal.services.v1.GovernanceServiceOuterClass.IngestRequirementsResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "IngestRequirements"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.IngestRequirementsRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.IngestRequirementsResponse.getDefaultInstance()))
              .setSchemaDescriptor(new GovernanceServiceMethodDescriptorSupplier("IngestRequirements"))
              .build();
        }
      }
    }
    return getIngestRequirementsMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.SemanticSearchRequest,
      oscal.services.v1.GovernanceServiceOuterClass.SemanticSearchResponse> getSemanticSearchMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "SemanticSearch",
      requestType = oscal.services.v1.GovernanceServiceOuterClass.SemanticSearchRequest.class,
      responseType = oscal.services.v1.GovernanceServiceOuterClass.SemanticSearchResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.SemanticSearchRequest,
      oscal.services.v1.GovernanceServiceOuterClass.SemanticSearchResponse> getSemanticSearchMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.SemanticSearchRequest, oscal.services.v1.GovernanceServiceOuterClass.SemanticSearchResponse> getSemanticSearchMethod;
    if ((getSemanticSearchMethod = GovernanceServiceGrpc.getSemanticSearchMethod) == null) {
      synchronized (GovernanceServiceGrpc.class) {
        if ((getSemanticSearchMethod = GovernanceServiceGrpc.getSemanticSearchMethod) == null) {
          GovernanceServiceGrpc.getSemanticSearchMethod = getSemanticSearchMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.GovernanceServiceOuterClass.SemanticSearchRequest, oscal.services.v1.GovernanceServiceOuterClass.SemanticSearchResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "SemanticSearch"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.SemanticSearchRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.SemanticSearchResponse.getDefaultInstance()))
              .setSchemaDescriptor(new GovernanceServiceMethodDescriptorSupplier("SemanticSearch"))
              .build();
        }
      }
    }
    return getSemanticSearchMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GenerateCatalogRequest,
      oscal.services.v1.GovernanceServiceOuterClass.GenerateCatalogResponse> getGenerateCatalogMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GenerateCatalog",
      requestType = oscal.services.v1.GovernanceServiceOuterClass.GenerateCatalogRequest.class,
      responseType = oscal.services.v1.GovernanceServiceOuterClass.GenerateCatalogResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GenerateCatalogRequest,
      oscal.services.v1.GovernanceServiceOuterClass.GenerateCatalogResponse> getGenerateCatalogMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GenerateCatalogRequest, oscal.services.v1.GovernanceServiceOuterClass.GenerateCatalogResponse> getGenerateCatalogMethod;
    if ((getGenerateCatalogMethod = GovernanceServiceGrpc.getGenerateCatalogMethod) == null) {
      synchronized (GovernanceServiceGrpc.class) {
        if ((getGenerateCatalogMethod = GovernanceServiceGrpc.getGenerateCatalogMethod) == null) {
          GovernanceServiceGrpc.getGenerateCatalogMethod = getGenerateCatalogMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.GovernanceServiceOuterClass.GenerateCatalogRequest, oscal.services.v1.GovernanceServiceOuterClass.GenerateCatalogResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GenerateCatalog"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.GenerateCatalogRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.GenerateCatalogResponse.getDefaultInstance()))
              .setSchemaDescriptor(new GovernanceServiceMethodDescriptorSupplier("GenerateCatalog"))
              .build();
        }
      }
    }
    return getGenerateCatalogMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GenerateProfileRequest,
      oscal.services.v1.GovernanceServiceOuterClass.GenerateProfileResponse> getGenerateProfileMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GenerateProfile",
      requestType = oscal.services.v1.GovernanceServiceOuterClass.GenerateProfileRequest.class,
      responseType = oscal.services.v1.GovernanceServiceOuterClass.GenerateProfileResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GenerateProfileRequest,
      oscal.services.v1.GovernanceServiceOuterClass.GenerateProfileResponse> getGenerateProfileMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GenerateProfileRequest, oscal.services.v1.GovernanceServiceOuterClass.GenerateProfileResponse> getGenerateProfileMethod;
    if ((getGenerateProfileMethod = GovernanceServiceGrpc.getGenerateProfileMethod) == null) {
      synchronized (GovernanceServiceGrpc.class) {
        if ((getGenerateProfileMethod = GovernanceServiceGrpc.getGenerateProfileMethod) == null) {
          GovernanceServiceGrpc.getGenerateProfileMethod = getGenerateProfileMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.GovernanceServiceOuterClass.GenerateProfileRequest, oscal.services.v1.GovernanceServiceOuterClass.GenerateProfileResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GenerateProfile"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.GenerateProfileRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.GenerateProfileResponse.getDefaultInstance()))
              .setSchemaDescriptor(new GovernanceServiceMethodDescriptorSupplier("GenerateProfile"))
              .build();
        }
      }
    }
    return getGenerateProfileMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GenerateMappingsRequest,
      oscal.services.v1.GovernanceServiceOuterClass.GenerateMappingsResponse> getGenerateMappingsMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GenerateMappings",
      requestType = oscal.services.v1.GovernanceServiceOuterClass.GenerateMappingsRequest.class,
      responseType = oscal.services.v1.GovernanceServiceOuterClass.GenerateMappingsResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GenerateMappingsRequest,
      oscal.services.v1.GovernanceServiceOuterClass.GenerateMappingsResponse> getGenerateMappingsMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GenerateMappingsRequest, oscal.services.v1.GovernanceServiceOuterClass.GenerateMappingsResponse> getGenerateMappingsMethod;
    if ((getGenerateMappingsMethod = GovernanceServiceGrpc.getGenerateMappingsMethod) == null) {
      synchronized (GovernanceServiceGrpc.class) {
        if ((getGenerateMappingsMethod = GovernanceServiceGrpc.getGenerateMappingsMethod) == null) {
          GovernanceServiceGrpc.getGenerateMappingsMethod = getGenerateMappingsMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.GovernanceServiceOuterClass.GenerateMappingsRequest, oscal.services.v1.GovernanceServiceOuterClass.GenerateMappingsResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GenerateMappings"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.GenerateMappingsRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.GenerateMappingsResponse.getDefaultInstance()))
              .setSchemaDescriptor(new GovernanceServiceMethodDescriptorSupplier("GenerateMappings"))
              .build();
        }
      }
    }
    return getGenerateMappingsMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GenerateSSPRequest,
      oscal.services.v1.GovernanceServiceOuterClass.GenerateSSPResponse> getGenerateSSPMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GenerateSSP",
      requestType = oscal.services.v1.GovernanceServiceOuterClass.GenerateSSPRequest.class,
      responseType = oscal.services.v1.GovernanceServiceOuterClass.GenerateSSPResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GenerateSSPRequest,
      oscal.services.v1.GovernanceServiceOuterClass.GenerateSSPResponse> getGenerateSSPMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GenerateSSPRequest, oscal.services.v1.GovernanceServiceOuterClass.GenerateSSPResponse> getGenerateSSPMethod;
    if ((getGenerateSSPMethod = GovernanceServiceGrpc.getGenerateSSPMethod) == null) {
      synchronized (GovernanceServiceGrpc.class) {
        if ((getGenerateSSPMethod = GovernanceServiceGrpc.getGenerateSSPMethod) == null) {
          GovernanceServiceGrpc.getGenerateSSPMethod = getGenerateSSPMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.GovernanceServiceOuterClass.GenerateSSPRequest, oscal.services.v1.GovernanceServiceOuterClass.GenerateSSPResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GenerateSSP"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.GenerateSSPRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.GenerateSSPResponse.getDefaultInstance()))
              .setSchemaDescriptor(new GovernanceServiceMethodDescriptorSupplier("GenerateSSP"))
              .build();
        }
      }
    }
    return getGenerateSSPMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GenerateComponentDefinitionRequest,
      oscal.services.v1.GovernanceServiceOuterClass.GenerateComponentDefinitionResponse> getGenerateComponentDefinitionMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GenerateComponentDefinition",
      requestType = oscal.services.v1.GovernanceServiceOuterClass.GenerateComponentDefinitionRequest.class,
      responseType = oscal.services.v1.GovernanceServiceOuterClass.GenerateComponentDefinitionResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GenerateComponentDefinitionRequest,
      oscal.services.v1.GovernanceServiceOuterClass.GenerateComponentDefinitionResponse> getGenerateComponentDefinitionMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GenerateComponentDefinitionRequest, oscal.services.v1.GovernanceServiceOuterClass.GenerateComponentDefinitionResponse> getGenerateComponentDefinitionMethod;
    if ((getGenerateComponentDefinitionMethod = GovernanceServiceGrpc.getGenerateComponentDefinitionMethod) == null) {
      synchronized (GovernanceServiceGrpc.class) {
        if ((getGenerateComponentDefinitionMethod = GovernanceServiceGrpc.getGenerateComponentDefinitionMethod) == null) {
          GovernanceServiceGrpc.getGenerateComponentDefinitionMethod = getGenerateComponentDefinitionMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.GovernanceServiceOuterClass.GenerateComponentDefinitionRequest, oscal.services.v1.GovernanceServiceOuterClass.GenerateComponentDefinitionResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GenerateComponentDefinition"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.GenerateComponentDefinitionRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.GenerateComponentDefinitionResponse.getDefaultInstance()))
              .setSchemaDescriptor(new GovernanceServiceMethodDescriptorSupplier("GenerateComponentDefinition"))
              .build();
        }
      }
    }
    return getGenerateComponentDefinitionMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentPlanRequest,
      oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentPlanResponse> getGenerateAssessmentPlanMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GenerateAssessmentPlan",
      requestType = oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentPlanRequest.class,
      responseType = oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentPlanResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentPlanRequest,
      oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentPlanResponse> getGenerateAssessmentPlanMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentPlanRequest, oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentPlanResponse> getGenerateAssessmentPlanMethod;
    if ((getGenerateAssessmentPlanMethod = GovernanceServiceGrpc.getGenerateAssessmentPlanMethod) == null) {
      synchronized (GovernanceServiceGrpc.class) {
        if ((getGenerateAssessmentPlanMethod = GovernanceServiceGrpc.getGenerateAssessmentPlanMethod) == null) {
          GovernanceServiceGrpc.getGenerateAssessmentPlanMethod = getGenerateAssessmentPlanMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentPlanRequest, oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentPlanResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GenerateAssessmentPlan"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentPlanRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentPlanResponse.getDefaultInstance()))
              .setSchemaDescriptor(new GovernanceServiceMethodDescriptorSupplier("GenerateAssessmentPlan"))
              .build();
        }
      }
    }
    return getGenerateAssessmentPlanMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GeneratePOAMRequest,
      oscal.services.v1.GovernanceServiceOuterClass.GeneratePOAMResponse> getGeneratePOAMMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GeneratePOAM",
      requestType = oscal.services.v1.GovernanceServiceOuterClass.GeneratePOAMRequest.class,
      responseType = oscal.services.v1.GovernanceServiceOuterClass.GeneratePOAMResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GeneratePOAMRequest,
      oscal.services.v1.GovernanceServiceOuterClass.GeneratePOAMResponse> getGeneratePOAMMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GeneratePOAMRequest, oscal.services.v1.GovernanceServiceOuterClass.GeneratePOAMResponse> getGeneratePOAMMethod;
    if ((getGeneratePOAMMethod = GovernanceServiceGrpc.getGeneratePOAMMethod) == null) {
      synchronized (GovernanceServiceGrpc.class) {
        if ((getGeneratePOAMMethod = GovernanceServiceGrpc.getGeneratePOAMMethod) == null) {
          GovernanceServiceGrpc.getGeneratePOAMMethod = getGeneratePOAMMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.GovernanceServiceOuterClass.GeneratePOAMRequest, oscal.services.v1.GovernanceServiceOuterClass.GeneratePOAMResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GeneratePOAM"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.GeneratePOAMRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.GeneratePOAMResponse.getDefaultInstance()))
              .setSchemaDescriptor(new GovernanceServiceMethodDescriptorSupplier("GeneratePOAM"))
              .build();
        }
      }
    }
    return getGeneratePOAMMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentResultsRequest,
      oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentResultsResponse> getGenerateAssessmentResultsMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GenerateAssessmentResults",
      requestType = oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentResultsRequest.class,
      responseType = oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentResultsResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentResultsRequest,
      oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentResultsResponse> getGenerateAssessmentResultsMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentResultsRequest, oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentResultsResponse> getGenerateAssessmentResultsMethod;
    if ((getGenerateAssessmentResultsMethod = GovernanceServiceGrpc.getGenerateAssessmentResultsMethod) == null) {
      synchronized (GovernanceServiceGrpc.class) {
        if ((getGenerateAssessmentResultsMethod = GovernanceServiceGrpc.getGenerateAssessmentResultsMethod) == null) {
          GovernanceServiceGrpc.getGenerateAssessmentResultsMethod = getGenerateAssessmentResultsMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentResultsRequest, oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentResultsResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GenerateAssessmentResults"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentResultsRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentResultsResponse.getDefaultInstance()))
              .setSchemaDescriptor(new GovernanceServiceMethodDescriptorSupplier("GenerateAssessmentResults"))
              .build();
        }
      }
    }
    return getGenerateAssessmentResultsMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.BulkIngestFrameworksRequest,
      oscal.services.v1.GovernanceServiceOuterClass.BulkIngestFrameworksResponse> getBulkIngestFrameworksMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "BulkIngestFrameworks",
      requestType = oscal.services.v1.GovernanceServiceOuterClass.BulkIngestFrameworksRequest.class,
      responseType = oscal.services.v1.GovernanceServiceOuterClass.BulkIngestFrameworksResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.BulkIngestFrameworksRequest,
      oscal.services.v1.GovernanceServiceOuterClass.BulkIngestFrameworksResponse> getBulkIngestFrameworksMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.BulkIngestFrameworksRequest, oscal.services.v1.GovernanceServiceOuterClass.BulkIngestFrameworksResponse> getBulkIngestFrameworksMethod;
    if ((getBulkIngestFrameworksMethod = GovernanceServiceGrpc.getBulkIngestFrameworksMethod) == null) {
      synchronized (GovernanceServiceGrpc.class) {
        if ((getBulkIngestFrameworksMethod = GovernanceServiceGrpc.getBulkIngestFrameworksMethod) == null) {
          GovernanceServiceGrpc.getBulkIngestFrameworksMethod = getBulkIngestFrameworksMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.GovernanceServiceOuterClass.BulkIngestFrameworksRequest, oscal.services.v1.GovernanceServiceOuterClass.BulkIngestFrameworksResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "BulkIngestFrameworks"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.BulkIngestFrameworksRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.BulkIngestFrameworksResponse.getDefaultInstance()))
              .setSchemaDescriptor(new GovernanceServiceMethodDescriptorSupplier("BulkIngestFrameworks"))
              .build();
        }
      }
    }
    return getBulkIngestFrameworksMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.ListFrameworksRequest,
      oscal.services.v1.GovernanceServiceOuterClass.ListFrameworksResponse> getListFrameworksMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "ListFrameworks",
      requestType = oscal.services.v1.GovernanceServiceOuterClass.ListFrameworksRequest.class,
      responseType = oscal.services.v1.GovernanceServiceOuterClass.ListFrameworksResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.ListFrameworksRequest,
      oscal.services.v1.GovernanceServiceOuterClass.ListFrameworksResponse> getListFrameworksMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.ListFrameworksRequest, oscal.services.v1.GovernanceServiceOuterClass.ListFrameworksResponse> getListFrameworksMethod;
    if ((getListFrameworksMethod = GovernanceServiceGrpc.getListFrameworksMethod) == null) {
      synchronized (GovernanceServiceGrpc.class) {
        if ((getListFrameworksMethod = GovernanceServiceGrpc.getListFrameworksMethod) == null) {
          GovernanceServiceGrpc.getListFrameworksMethod = getListFrameworksMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.GovernanceServiceOuterClass.ListFrameworksRequest, oscal.services.v1.GovernanceServiceOuterClass.ListFrameworksResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "ListFrameworks"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.ListFrameworksRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.ListFrameworksResponse.getDefaultInstance()))
              .setSchemaDescriptor(new GovernanceServiceMethodDescriptorSupplier("ListFrameworks"))
              .build();
        }
      }
    }
    return getListFrameworksMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GetFrameworkRequest,
      oscal.services.v1.GovernanceServiceOuterClass.GetFrameworkResponse> getGetFrameworkMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GetFramework",
      requestType = oscal.services.v1.GovernanceServiceOuterClass.GetFrameworkRequest.class,
      responseType = oscal.services.v1.GovernanceServiceOuterClass.GetFrameworkResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GetFrameworkRequest,
      oscal.services.v1.GovernanceServiceOuterClass.GetFrameworkResponse> getGetFrameworkMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GetFrameworkRequest, oscal.services.v1.GovernanceServiceOuterClass.GetFrameworkResponse> getGetFrameworkMethod;
    if ((getGetFrameworkMethod = GovernanceServiceGrpc.getGetFrameworkMethod) == null) {
      synchronized (GovernanceServiceGrpc.class) {
        if ((getGetFrameworkMethod = GovernanceServiceGrpc.getGetFrameworkMethod) == null) {
          GovernanceServiceGrpc.getGetFrameworkMethod = getGetFrameworkMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.GovernanceServiceOuterClass.GetFrameworkRequest, oscal.services.v1.GovernanceServiceOuterClass.GetFrameworkResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GetFramework"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.GetFrameworkRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.GetFrameworkResponse.getDefaultInstance()))
              .setSchemaDescriptor(new GovernanceServiceMethodDescriptorSupplier("GetFramework"))
              .build();
        }
      }
    }
    return getGetFrameworkMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GenerateCrossFrameworkMappingsRequest,
      oscal.services.v1.GovernanceServiceOuterClass.GenerateCrossFrameworkMappingsResponse> getGenerateCrossFrameworkMappingsMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GenerateCrossFrameworkMappings",
      requestType = oscal.services.v1.GovernanceServiceOuterClass.GenerateCrossFrameworkMappingsRequest.class,
      responseType = oscal.services.v1.GovernanceServiceOuterClass.GenerateCrossFrameworkMappingsResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GenerateCrossFrameworkMappingsRequest,
      oscal.services.v1.GovernanceServiceOuterClass.GenerateCrossFrameworkMappingsResponse> getGenerateCrossFrameworkMappingsMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.GenerateCrossFrameworkMappingsRequest, oscal.services.v1.GovernanceServiceOuterClass.GenerateCrossFrameworkMappingsResponse> getGenerateCrossFrameworkMappingsMethod;
    if ((getGenerateCrossFrameworkMappingsMethod = GovernanceServiceGrpc.getGenerateCrossFrameworkMappingsMethod) == null) {
      synchronized (GovernanceServiceGrpc.class) {
        if ((getGenerateCrossFrameworkMappingsMethod = GovernanceServiceGrpc.getGenerateCrossFrameworkMappingsMethod) == null) {
          GovernanceServiceGrpc.getGenerateCrossFrameworkMappingsMethod = getGenerateCrossFrameworkMappingsMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.GovernanceServiceOuterClass.GenerateCrossFrameworkMappingsRequest, oscal.services.v1.GovernanceServiceOuterClass.GenerateCrossFrameworkMappingsResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GenerateCrossFrameworkMappings"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.GenerateCrossFrameworkMappingsRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.GenerateCrossFrameworkMappingsResponse.getDefaultInstance()))
              .setSchemaDescriptor(new GovernanceServiceMethodDescriptorSupplier("GenerateCrossFrameworkMappings"))
              .build();
        }
      }
    }
    return getGenerateCrossFrameworkMappingsMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.ProposeRequest,
      oscal.services.v1.GovernanceServiceOuterClass.ProposeResponse> getProposeMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "Propose",
      requestType = oscal.services.v1.GovernanceServiceOuterClass.ProposeRequest.class,
      responseType = oscal.services.v1.GovernanceServiceOuterClass.ProposeResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.ProposeRequest,
      oscal.services.v1.GovernanceServiceOuterClass.ProposeResponse> getProposeMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.ProposeRequest, oscal.services.v1.GovernanceServiceOuterClass.ProposeResponse> getProposeMethod;
    if ((getProposeMethod = GovernanceServiceGrpc.getProposeMethod) == null) {
      synchronized (GovernanceServiceGrpc.class) {
        if ((getProposeMethod = GovernanceServiceGrpc.getProposeMethod) == null) {
          GovernanceServiceGrpc.getProposeMethod = getProposeMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.GovernanceServiceOuterClass.ProposeRequest, oscal.services.v1.GovernanceServiceOuterClass.ProposeResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "Propose"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.ProposeRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.ProposeResponse.getDefaultInstance()))
              .setSchemaDescriptor(new GovernanceServiceMethodDescriptorSupplier("Propose"))
              .build();
        }
      }
    }
    return getProposeMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.ListConflictsRequest,
      oscal.services.v1.GovernanceServiceOuterClass.ListConflictsResponse> getListConflictsMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "ListConflicts",
      requestType = oscal.services.v1.GovernanceServiceOuterClass.ListConflictsRequest.class,
      responseType = oscal.services.v1.GovernanceServiceOuterClass.ListConflictsResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.ListConflictsRequest,
      oscal.services.v1.GovernanceServiceOuterClass.ListConflictsResponse> getListConflictsMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.ListConflictsRequest, oscal.services.v1.GovernanceServiceOuterClass.ListConflictsResponse> getListConflictsMethod;
    if ((getListConflictsMethod = GovernanceServiceGrpc.getListConflictsMethod) == null) {
      synchronized (GovernanceServiceGrpc.class) {
        if ((getListConflictsMethod = GovernanceServiceGrpc.getListConflictsMethod) == null) {
          GovernanceServiceGrpc.getListConflictsMethod = getListConflictsMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.GovernanceServiceOuterClass.ListConflictsRequest, oscal.services.v1.GovernanceServiceOuterClass.ListConflictsResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "ListConflicts"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.ListConflictsRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.ListConflictsResponse.getDefaultInstance()))
              .setSchemaDescriptor(new GovernanceServiceMethodDescriptorSupplier("ListConflicts"))
              .build();
        }
      }
    }
    return getListConflictsMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.ResolveConflictRequest,
      oscal.services.v1.GovernanceServiceOuterClass.ResolveConflictResponse> getResolveConflictMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "ResolveConflict",
      requestType = oscal.services.v1.GovernanceServiceOuterClass.ResolveConflictRequest.class,
      responseType = oscal.services.v1.GovernanceServiceOuterClass.ResolveConflictResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.ResolveConflictRequest,
      oscal.services.v1.GovernanceServiceOuterClass.ResolveConflictResponse> getResolveConflictMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.ResolveConflictRequest, oscal.services.v1.GovernanceServiceOuterClass.ResolveConflictResponse> getResolveConflictMethod;
    if ((getResolveConflictMethod = GovernanceServiceGrpc.getResolveConflictMethod) == null) {
      synchronized (GovernanceServiceGrpc.class) {
        if ((getResolveConflictMethod = GovernanceServiceGrpc.getResolveConflictMethod) == null) {
          GovernanceServiceGrpc.getResolveConflictMethod = getResolveConflictMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.GovernanceServiceOuterClass.ResolveConflictRequest, oscal.services.v1.GovernanceServiceOuterClass.ResolveConflictResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "ResolveConflict"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.ResolveConflictRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.ResolveConflictResponse.getDefaultInstance()))
              .setSchemaDescriptor(new GovernanceServiceMethodDescriptorSupplier("ResolveConflict"))
              .build();
        }
      }
    }
    return getResolveConflictMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.PublishReleaseRequest,
      oscal.services.v1.GovernanceServiceOuterClass.PublishReleaseResponse> getPublishReleaseMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "PublishRelease",
      requestType = oscal.services.v1.GovernanceServiceOuterClass.PublishReleaseRequest.class,
      responseType = oscal.services.v1.GovernanceServiceOuterClass.PublishReleaseResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.PublishReleaseRequest,
      oscal.services.v1.GovernanceServiceOuterClass.PublishReleaseResponse> getPublishReleaseMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.PublishReleaseRequest, oscal.services.v1.GovernanceServiceOuterClass.PublishReleaseResponse> getPublishReleaseMethod;
    if ((getPublishReleaseMethod = GovernanceServiceGrpc.getPublishReleaseMethod) == null) {
      synchronized (GovernanceServiceGrpc.class) {
        if ((getPublishReleaseMethod = GovernanceServiceGrpc.getPublishReleaseMethod) == null) {
          GovernanceServiceGrpc.getPublishReleaseMethod = getPublishReleaseMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.GovernanceServiceOuterClass.PublishReleaseRequest, oscal.services.v1.GovernanceServiceOuterClass.PublishReleaseResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "PublishRelease"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.PublishReleaseRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.PublishReleaseResponse.getDefaultInstance()))
              .setSchemaDescriptor(new GovernanceServiceMethodDescriptorSupplier("PublishRelease"))
              .build();
        }
      }
    }
    return getPublishReleaseMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.ProposeMappingUpdateRequest,
      oscal.services.v1.GovernanceServiceOuterClass.ProposeMappingUpdateResponse> getProposeMappingUpdateMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "ProposeMappingUpdate",
      requestType = oscal.services.v1.GovernanceServiceOuterClass.ProposeMappingUpdateRequest.class,
      responseType = oscal.services.v1.GovernanceServiceOuterClass.ProposeMappingUpdateResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.ProposeMappingUpdateRequest,
      oscal.services.v1.GovernanceServiceOuterClass.ProposeMappingUpdateResponse> getProposeMappingUpdateMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.GovernanceServiceOuterClass.ProposeMappingUpdateRequest, oscal.services.v1.GovernanceServiceOuterClass.ProposeMappingUpdateResponse> getProposeMappingUpdateMethod;
    if ((getProposeMappingUpdateMethod = GovernanceServiceGrpc.getProposeMappingUpdateMethod) == null) {
      synchronized (GovernanceServiceGrpc.class) {
        if ((getProposeMappingUpdateMethod = GovernanceServiceGrpc.getProposeMappingUpdateMethod) == null) {
          GovernanceServiceGrpc.getProposeMappingUpdateMethod = getProposeMappingUpdateMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.GovernanceServiceOuterClass.ProposeMappingUpdateRequest, oscal.services.v1.GovernanceServiceOuterClass.ProposeMappingUpdateResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "ProposeMappingUpdate"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.ProposeMappingUpdateRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.GovernanceServiceOuterClass.ProposeMappingUpdateResponse.getDefaultInstance()))
              .setSchemaDescriptor(new GovernanceServiceMethodDescriptorSupplier("ProposeMappingUpdate"))
              .build();
        }
      }
    }
    return getProposeMappingUpdateMethod;
  }

  /**
   * Creates a new async stub that supports all call types for the service
   */
  public static GovernanceServiceStub newStub(io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<GovernanceServiceStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<GovernanceServiceStub>() {
        @java.lang.Override
        public GovernanceServiceStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new GovernanceServiceStub(channel, callOptions);
        }
      };
    return GovernanceServiceStub.newStub(factory, channel);
  }

  /**
   * Creates a new blocking-style stub that supports all types of calls on the service
   */
  public static GovernanceServiceBlockingV2Stub newBlockingV2Stub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<GovernanceServiceBlockingV2Stub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<GovernanceServiceBlockingV2Stub>() {
        @java.lang.Override
        public GovernanceServiceBlockingV2Stub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new GovernanceServiceBlockingV2Stub(channel, callOptions);
        }
      };
    return GovernanceServiceBlockingV2Stub.newStub(factory, channel);
  }

  /**
   * Creates a new blocking-style stub that supports unary and streaming output calls on the service
   */
  public static GovernanceServiceBlockingStub newBlockingStub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<GovernanceServiceBlockingStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<GovernanceServiceBlockingStub>() {
        @java.lang.Override
        public GovernanceServiceBlockingStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new GovernanceServiceBlockingStub(channel, callOptions);
        }
      };
    return GovernanceServiceBlockingStub.newStub(factory, channel);
  }

  /**
   * Creates a new ListenableFuture-style stub that supports unary calls on the service
   */
  public static GovernanceServiceFutureStub newFutureStub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<GovernanceServiceFutureStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<GovernanceServiceFutureStub>() {
        @java.lang.Override
        public GovernanceServiceFutureStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new GovernanceServiceFutureStub(channel, callOptions);
        }
      };
    return GovernanceServiceFutureStub.newStub(factory, channel);
  }

  /**
   * <pre>
   * GovernanceService provides advanced governance, ingestion, and semantic
   * search operations over the OSCALify knowledge graph.
   * </pre>
   */
  public interface AsyncService {

    /**
     * <pre>
     * --- Entity CRUD ---
     * </pre>
     */
    default void createEntity(oscal.services.v1.GovernanceServiceOuterClass.CreateEntityRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.CreateEntityResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getCreateEntityMethod(), responseObserver);
    }

    /**
     */
    default void getEntity(oscal.services.v1.GovernanceServiceOuterClass.GetEntityRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GetEntityResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getGetEntityMethod(), responseObserver);
    }

    /**
     */
    default void updateEntity(oscal.services.v1.GovernanceServiceOuterClass.UpdateEntityRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.UpdateEntityResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getUpdateEntityMethod(), responseObserver);
    }

    /**
     */
    default void listEntities(oscal.services.v1.GovernanceServiceOuterClass.ListEntitiesRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.ListEntitiesResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getListEntitiesMethod(), responseObserver);
    }

    /**
     * <pre>
     * --- Snapshots &amp; Releases ---
     * </pre>
     */
    default void createSnapshot(oscal.services.v1.GovernanceServiceOuterClass.CreateSnapshotRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.CreateSnapshotResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getCreateSnapshotMethod(), responseObserver);
    }

    /**
     */
    default void getSnapshot(oscal.services.v1.GovernanceServiceOuterClass.GetSnapshotRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GetSnapshotResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getGetSnapshotMethod(), responseObserver);
    }

    /**
     */
    default void listSnapshots(oscal.services.v1.GovernanceServiceOuterClass.ListSnapshotsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.ListSnapshotsResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getListSnapshotsMethod(), responseObserver);
    }

    /**
     */
    default void createRelease(oscal.services.v1.GovernanceServiceOuterClass.CreateReleaseRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.CreateReleaseResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getCreateReleaseMethod(), responseObserver);
    }

    /**
     */
    default void listReleases(oscal.services.v1.GovernanceServiceOuterClass.ListReleasesRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.ListReleasesResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getListReleasesMethod(), responseObserver);
    }

    /**
     * <pre>
     * --- Ingestion ---
     * </pre>
     */
    default void ingestRequirements(oscal.services.v1.GovernanceServiceOuterClass.IngestRequirementsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.IngestRequirementsResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getIngestRequirementsMethod(), responseObserver);
    }

    /**
     * <pre>
     * --- Semantic Search ---
     * </pre>
     */
    default void semanticSearch(oscal.services.v1.GovernanceServiceOuterClass.SemanticSearchRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.SemanticSearchResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getSemanticSearchMethod(), responseObserver);
    }

    /**
     * <pre>
     * --- OSCAL Generation ---
     * </pre>
     */
    default void generateCatalog(oscal.services.v1.GovernanceServiceOuterClass.GenerateCatalogRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GenerateCatalogResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getGenerateCatalogMethod(), responseObserver);
    }

    /**
     */
    default void generateProfile(oscal.services.v1.GovernanceServiceOuterClass.GenerateProfileRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GenerateProfileResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getGenerateProfileMethod(), responseObserver);
    }

    /**
     */
    default void generateMappings(oscal.services.v1.GovernanceServiceOuterClass.GenerateMappingsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GenerateMappingsResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getGenerateMappingsMethod(), responseObserver);
    }

    /**
     */
    default void generateSSP(oscal.services.v1.GovernanceServiceOuterClass.GenerateSSPRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GenerateSSPResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getGenerateSSPMethod(), responseObserver);
    }

    /**
     */
    default void generateComponentDefinition(oscal.services.v1.GovernanceServiceOuterClass.GenerateComponentDefinitionRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GenerateComponentDefinitionResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getGenerateComponentDefinitionMethod(), responseObserver);
    }

    /**
     */
    default void generateAssessmentPlan(oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentPlanRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentPlanResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getGenerateAssessmentPlanMethod(), responseObserver);
    }

    /**
     */
    default void generatePOAM(oscal.services.v1.GovernanceServiceOuterClass.GeneratePOAMRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GeneratePOAMResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getGeneratePOAMMethod(), responseObserver);
    }

    /**
     */
    default void generateAssessmentResults(oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentResultsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentResultsResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getGenerateAssessmentResultsMethod(), responseObserver);
    }

    /**
     * <pre>
     * --- Framework Ingestion ---
     * </pre>
     */
    default void bulkIngestFrameworks(oscal.services.v1.GovernanceServiceOuterClass.BulkIngestFrameworksRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.BulkIngestFrameworksResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getBulkIngestFrameworksMethod(), responseObserver);
    }

    /**
     */
    default void listFrameworks(oscal.services.v1.GovernanceServiceOuterClass.ListFrameworksRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.ListFrameworksResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getListFrameworksMethod(), responseObserver);
    }

    /**
     */
    default void getFramework(oscal.services.v1.GovernanceServiceOuterClass.GetFrameworkRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GetFrameworkResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getGetFrameworkMethod(), responseObserver);
    }

    /**
     * <pre>
     * --- Cross-Framework Mappings ---
     * </pre>
     */
    default void generateCrossFrameworkMappings(oscal.services.v1.GovernanceServiceOuterClass.GenerateCrossFrameworkMappingsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GenerateCrossFrameworkMappingsResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getGenerateCrossFrameworkMappingsMethod(), responseObserver);
    }

    /**
     * <pre>
     * --- Reconciler ---
     * </pre>
     */
    default void propose(oscal.services.v1.GovernanceServiceOuterClass.ProposeRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.ProposeResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getProposeMethod(), responseObserver);
    }

    /**
     */
    default void listConflicts(oscal.services.v1.GovernanceServiceOuterClass.ListConflictsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.ListConflictsResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getListConflictsMethod(), responseObserver);
    }

    /**
     */
    default void resolveConflict(oscal.services.v1.GovernanceServiceOuterClass.ResolveConflictRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.ResolveConflictResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getResolveConflictMethod(), responseObserver);
    }

    /**
     * <pre>
     * --- GitOps ---
     * </pre>
     */
    default void publishRelease(oscal.services.v1.GovernanceServiceOuterClass.PublishReleaseRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.PublishReleaseResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getPublishReleaseMethod(), responseObserver);
    }

    /**
     */
    default void proposeMappingUpdate(oscal.services.v1.GovernanceServiceOuterClass.ProposeMappingUpdateRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.ProposeMappingUpdateResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getProposeMappingUpdateMethod(), responseObserver);
    }
  }

  /**
   * Base class for the server implementation of the service GovernanceService.
   * <pre>
   * GovernanceService provides advanced governance, ingestion, and semantic
   * search operations over the OSCALify knowledge graph.
   * </pre>
   */
  public static abstract class GovernanceServiceImplBase
      implements io.grpc.BindableService, AsyncService {

    @java.lang.Override public final io.grpc.ServerServiceDefinition bindService() {
      return GovernanceServiceGrpc.bindService(this);
    }
  }

  /**
   * A stub to allow clients to do asynchronous rpc calls to service GovernanceService.
   * <pre>
   * GovernanceService provides advanced governance, ingestion, and semantic
   * search operations over the OSCALify knowledge graph.
   * </pre>
   */
  public static final class GovernanceServiceStub
      extends io.grpc.stub.AbstractAsyncStub<GovernanceServiceStub> {
    private GovernanceServiceStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected GovernanceServiceStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new GovernanceServiceStub(channel, callOptions);
    }

    /**
     * <pre>
     * --- Entity CRUD ---
     * </pre>
     */
    public void createEntity(oscal.services.v1.GovernanceServiceOuterClass.CreateEntityRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.CreateEntityResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getCreateEntityMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void getEntity(oscal.services.v1.GovernanceServiceOuterClass.GetEntityRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GetEntityResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getGetEntityMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void updateEntity(oscal.services.v1.GovernanceServiceOuterClass.UpdateEntityRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.UpdateEntityResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getUpdateEntityMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void listEntities(oscal.services.v1.GovernanceServiceOuterClass.ListEntitiesRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.ListEntitiesResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getListEntitiesMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * --- Snapshots &amp; Releases ---
     * </pre>
     */
    public void createSnapshot(oscal.services.v1.GovernanceServiceOuterClass.CreateSnapshotRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.CreateSnapshotResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getCreateSnapshotMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void getSnapshot(oscal.services.v1.GovernanceServiceOuterClass.GetSnapshotRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GetSnapshotResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getGetSnapshotMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void listSnapshots(oscal.services.v1.GovernanceServiceOuterClass.ListSnapshotsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.ListSnapshotsResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getListSnapshotsMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void createRelease(oscal.services.v1.GovernanceServiceOuterClass.CreateReleaseRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.CreateReleaseResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getCreateReleaseMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void listReleases(oscal.services.v1.GovernanceServiceOuterClass.ListReleasesRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.ListReleasesResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getListReleasesMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * --- Ingestion ---
     * </pre>
     */
    public void ingestRequirements(oscal.services.v1.GovernanceServiceOuterClass.IngestRequirementsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.IngestRequirementsResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getIngestRequirementsMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * --- Semantic Search ---
     * </pre>
     */
    public void semanticSearch(oscal.services.v1.GovernanceServiceOuterClass.SemanticSearchRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.SemanticSearchResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getSemanticSearchMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * --- OSCAL Generation ---
     * </pre>
     */
    public void generateCatalog(oscal.services.v1.GovernanceServiceOuterClass.GenerateCatalogRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GenerateCatalogResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getGenerateCatalogMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void generateProfile(oscal.services.v1.GovernanceServiceOuterClass.GenerateProfileRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GenerateProfileResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getGenerateProfileMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void generateMappings(oscal.services.v1.GovernanceServiceOuterClass.GenerateMappingsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GenerateMappingsResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getGenerateMappingsMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void generateSSP(oscal.services.v1.GovernanceServiceOuterClass.GenerateSSPRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GenerateSSPResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getGenerateSSPMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void generateComponentDefinition(oscal.services.v1.GovernanceServiceOuterClass.GenerateComponentDefinitionRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GenerateComponentDefinitionResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getGenerateComponentDefinitionMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void generateAssessmentPlan(oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentPlanRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentPlanResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getGenerateAssessmentPlanMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void generatePOAM(oscal.services.v1.GovernanceServiceOuterClass.GeneratePOAMRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GeneratePOAMResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getGeneratePOAMMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void generateAssessmentResults(oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentResultsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentResultsResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getGenerateAssessmentResultsMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * --- Framework Ingestion ---
     * </pre>
     */
    public void bulkIngestFrameworks(oscal.services.v1.GovernanceServiceOuterClass.BulkIngestFrameworksRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.BulkIngestFrameworksResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getBulkIngestFrameworksMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void listFrameworks(oscal.services.v1.GovernanceServiceOuterClass.ListFrameworksRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.ListFrameworksResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getListFrameworksMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void getFramework(oscal.services.v1.GovernanceServiceOuterClass.GetFrameworkRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GetFrameworkResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getGetFrameworkMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * --- Cross-Framework Mappings ---
     * </pre>
     */
    public void generateCrossFrameworkMappings(oscal.services.v1.GovernanceServiceOuterClass.GenerateCrossFrameworkMappingsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GenerateCrossFrameworkMappingsResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getGenerateCrossFrameworkMappingsMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * --- Reconciler ---
     * </pre>
     */
    public void propose(oscal.services.v1.GovernanceServiceOuterClass.ProposeRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.ProposeResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getProposeMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void listConflicts(oscal.services.v1.GovernanceServiceOuterClass.ListConflictsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.ListConflictsResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getListConflictsMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void resolveConflict(oscal.services.v1.GovernanceServiceOuterClass.ResolveConflictRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.ResolveConflictResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getResolveConflictMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * --- GitOps ---
     * </pre>
     */
    public void publishRelease(oscal.services.v1.GovernanceServiceOuterClass.PublishReleaseRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.PublishReleaseResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getPublishReleaseMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void proposeMappingUpdate(oscal.services.v1.GovernanceServiceOuterClass.ProposeMappingUpdateRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.ProposeMappingUpdateResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getProposeMappingUpdateMethod(), getCallOptions()), request, responseObserver);
    }
  }

  /**
   * A stub to allow clients to do synchronous rpc calls to service GovernanceService.
   * <pre>
   * GovernanceService provides advanced governance, ingestion, and semantic
   * search operations over the OSCALify knowledge graph.
   * </pre>
   */
  public static final class GovernanceServiceBlockingV2Stub
      extends io.grpc.stub.AbstractBlockingStub<GovernanceServiceBlockingV2Stub> {
    private GovernanceServiceBlockingV2Stub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected GovernanceServiceBlockingV2Stub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new GovernanceServiceBlockingV2Stub(channel, callOptions);
    }

    /**
     * <pre>
     * --- Entity CRUD ---
     * </pre>
     */
    public oscal.services.v1.GovernanceServiceOuterClass.CreateEntityResponse createEntity(oscal.services.v1.GovernanceServiceOuterClass.CreateEntityRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getCreateEntityMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.GetEntityResponse getEntity(oscal.services.v1.GovernanceServiceOuterClass.GetEntityRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getGetEntityMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.UpdateEntityResponse updateEntity(oscal.services.v1.GovernanceServiceOuterClass.UpdateEntityRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getUpdateEntityMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.ListEntitiesResponse listEntities(oscal.services.v1.GovernanceServiceOuterClass.ListEntitiesRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getListEntitiesMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * --- Snapshots &amp; Releases ---
     * </pre>
     */
    public oscal.services.v1.GovernanceServiceOuterClass.CreateSnapshotResponse createSnapshot(oscal.services.v1.GovernanceServiceOuterClass.CreateSnapshotRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getCreateSnapshotMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.GetSnapshotResponse getSnapshot(oscal.services.v1.GovernanceServiceOuterClass.GetSnapshotRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getGetSnapshotMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.ListSnapshotsResponse listSnapshots(oscal.services.v1.GovernanceServiceOuterClass.ListSnapshotsRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getListSnapshotsMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.CreateReleaseResponse createRelease(oscal.services.v1.GovernanceServiceOuterClass.CreateReleaseRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getCreateReleaseMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.ListReleasesResponse listReleases(oscal.services.v1.GovernanceServiceOuterClass.ListReleasesRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getListReleasesMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * --- Ingestion ---
     * </pre>
     */
    public oscal.services.v1.GovernanceServiceOuterClass.IngestRequirementsResponse ingestRequirements(oscal.services.v1.GovernanceServiceOuterClass.IngestRequirementsRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getIngestRequirementsMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * --- Semantic Search ---
     * </pre>
     */
    public oscal.services.v1.GovernanceServiceOuterClass.SemanticSearchResponse semanticSearch(oscal.services.v1.GovernanceServiceOuterClass.SemanticSearchRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getSemanticSearchMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * --- OSCAL Generation ---
     * </pre>
     */
    public oscal.services.v1.GovernanceServiceOuterClass.GenerateCatalogResponse generateCatalog(oscal.services.v1.GovernanceServiceOuterClass.GenerateCatalogRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getGenerateCatalogMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.GenerateProfileResponse generateProfile(oscal.services.v1.GovernanceServiceOuterClass.GenerateProfileRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getGenerateProfileMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.GenerateMappingsResponse generateMappings(oscal.services.v1.GovernanceServiceOuterClass.GenerateMappingsRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getGenerateMappingsMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.GenerateSSPResponse generateSSP(oscal.services.v1.GovernanceServiceOuterClass.GenerateSSPRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getGenerateSSPMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.GenerateComponentDefinitionResponse generateComponentDefinition(oscal.services.v1.GovernanceServiceOuterClass.GenerateComponentDefinitionRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getGenerateComponentDefinitionMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentPlanResponse generateAssessmentPlan(oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentPlanRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getGenerateAssessmentPlanMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.GeneratePOAMResponse generatePOAM(oscal.services.v1.GovernanceServiceOuterClass.GeneratePOAMRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getGeneratePOAMMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentResultsResponse generateAssessmentResults(oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentResultsRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getGenerateAssessmentResultsMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * --- Framework Ingestion ---
     * </pre>
     */
    public oscal.services.v1.GovernanceServiceOuterClass.BulkIngestFrameworksResponse bulkIngestFrameworks(oscal.services.v1.GovernanceServiceOuterClass.BulkIngestFrameworksRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getBulkIngestFrameworksMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.ListFrameworksResponse listFrameworks(oscal.services.v1.GovernanceServiceOuterClass.ListFrameworksRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getListFrameworksMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.GetFrameworkResponse getFramework(oscal.services.v1.GovernanceServiceOuterClass.GetFrameworkRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getGetFrameworkMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * --- Cross-Framework Mappings ---
     * </pre>
     */
    public oscal.services.v1.GovernanceServiceOuterClass.GenerateCrossFrameworkMappingsResponse generateCrossFrameworkMappings(oscal.services.v1.GovernanceServiceOuterClass.GenerateCrossFrameworkMappingsRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getGenerateCrossFrameworkMappingsMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * --- Reconciler ---
     * </pre>
     */
    public oscal.services.v1.GovernanceServiceOuterClass.ProposeResponse propose(oscal.services.v1.GovernanceServiceOuterClass.ProposeRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getProposeMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.ListConflictsResponse listConflicts(oscal.services.v1.GovernanceServiceOuterClass.ListConflictsRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getListConflictsMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.ResolveConflictResponse resolveConflict(oscal.services.v1.GovernanceServiceOuterClass.ResolveConflictRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getResolveConflictMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * --- GitOps ---
     * </pre>
     */
    public oscal.services.v1.GovernanceServiceOuterClass.PublishReleaseResponse publishRelease(oscal.services.v1.GovernanceServiceOuterClass.PublishReleaseRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getPublishReleaseMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.ProposeMappingUpdateResponse proposeMappingUpdate(oscal.services.v1.GovernanceServiceOuterClass.ProposeMappingUpdateRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getProposeMappingUpdateMethod(), getCallOptions(), request);
    }
  }

  /**
   * A stub to allow clients to do limited synchronous rpc calls to service GovernanceService.
   * <pre>
   * GovernanceService provides advanced governance, ingestion, and semantic
   * search operations over the OSCALify knowledge graph.
   * </pre>
   */
  public static final class GovernanceServiceBlockingStub
      extends io.grpc.stub.AbstractBlockingStub<GovernanceServiceBlockingStub> {
    private GovernanceServiceBlockingStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected GovernanceServiceBlockingStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new GovernanceServiceBlockingStub(channel, callOptions);
    }

    /**
     * <pre>
     * --- Entity CRUD ---
     * </pre>
     */
    public oscal.services.v1.GovernanceServiceOuterClass.CreateEntityResponse createEntity(oscal.services.v1.GovernanceServiceOuterClass.CreateEntityRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getCreateEntityMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.GetEntityResponse getEntity(oscal.services.v1.GovernanceServiceOuterClass.GetEntityRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getGetEntityMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.UpdateEntityResponse updateEntity(oscal.services.v1.GovernanceServiceOuterClass.UpdateEntityRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getUpdateEntityMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.ListEntitiesResponse listEntities(oscal.services.v1.GovernanceServiceOuterClass.ListEntitiesRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getListEntitiesMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * --- Snapshots &amp; Releases ---
     * </pre>
     */
    public oscal.services.v1.GovernanceServiceOuterClass.CreateSnapshotResponse createSnapshot(oscal.services.v1.GovernanceServiceOuterClass.CreateSnapshotRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getCreateSnapshotMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.GetSnapshotResponse getSnapshot(oscal.services.v1.GovernanceServiceOuterClass.GetSnapshotRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getGetSnapshotMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.ListSnapshotsResponse listSnapshots(oscal.services.v1.GovernanceServiceOuterClass.ListSnapshotsRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getListSnapshotsMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.CreateReleaseResponse createRelease(oscal.services.v1.GovernanceServiceOuterClass.CreateReleaseRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getCreateReleaseMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.ListReleasesResponse listReleases(oscal.services.v1.GovernanceServiceOuterClass.ListReleasesRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getListReleasesMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * --- Ingestion ---
     * </pre>
     */
    public oscal.services.v1.GovernanceServiceOuterClass.IngestRequirementsResponse ingestRequirements(oscal.services.v1.GovernanceServiceOuterClass.IngestRequirementsRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getIngestRequirementsMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * --- Semantic Search ---
     * </pre>
     */
    public oscal.services.v1.GovernanceServiceOuterClass.SemanticSearchResponse semanticSearch(oscal.services.v1.GovernanceServiceOuterClass.SemanticSearchRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getSemanticSearchMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * --- OSCAL Generation ---
     * </pre>
     */
    public oscal.services.v1.GovernanceServiceOuterClass.GenerateCatalogResponse generateCatalog(oscal.services.v1.GovernanceServiceOuterClass.GenerateCatalogRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getGenerateCatalogMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.GenerateProfileResponse generateProfile(oscal.services.v1.GovernanceServiceOuterClass.GenerateProfileRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getGenerateProfileMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.GenerateMappingsResponse generateMappings(oscal.services.v1.GovernanceServiceOuterClass.GenerateMappingsRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getGenerateMappingsMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.GenerateSSPResponse generateSSP(oscal.services.v1.GovernanceServiceOuterClass.GenerateSSPRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getGenerateSSPMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.GenerateComponentDefinitionResponse generateComponentDefinition(oscal.services.v1.GovernanceServiceOuterClass.GenerateComponentDefinitionRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getGenerateComponentDefinitionMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentPlanResponse generateAssessmentPlan(oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentPlanRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getGenerateAssessmentPlanMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.GeneratePOAMResponse generatePOAM(oscal.services.v1.GovernanceServiceOuterClass.GeneratePOAMRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getGeneratePOAMMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentResultsResponse generateAssessmentResults(oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentResultsRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getGenerateAssessmentResultsMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * --- Framework Ingestion ---
     * </pre>
     */
    public oscal.services.v1.GovernanceServiceOuterClass.BulkIngestFrameworksResponse bulkIngestFrameworks(oscal.services.v1.GovernanceServiceOuterClass.BulkIngestFrameworksRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getBulkIngestFrameworksMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.ListFrameworksResponse listFrameworks(oscal.services.v1.GovernanceServiceOuterClass.ListFrameworksRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getListFrameworksMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.GetFrameworkResponse getFramework(oscal.services.v1.GovernanceServiceOuterClass.GetFrameworkRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getGetFrameworkMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * --- Cross-Framework Mappings ---
     * </pre>
     */
    public oscal.services.v1.GovernanceServiceOuterClass.GenerateCrossFrameworkMappingsResponse generateCrossFrameworkMappings(oscal.services.v1.GovernanceServiceOuterClass.GenerateCrossFrameworkMappingsRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getGenerateCrossFrameworkMappingsMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * --- Reconciler ---
     * </pre>
     */
    public oscal.services.v1.GovernanceServiceOuterClass.ProposeResponse propose(oscal.services.v1.GovernanceServiceOuterClass.ProposeRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getProposeMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.ListConflictsResponse listConflicts(oscal.services.v1.GovernanceServiceOuterClass.ListConflictsRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getListConflictsMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.ResolveConflictResponse resolveConflict(oscal.services.v1.GovernanceServiceOuterClass.ResolveConflictRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getResolveConflictMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * --- GitOps ---
     * </pre>
     */
    public oscal.services.v1.GovernanceServiceOuterClass.PublishReleaseResponse publishRelease(oscal.services.v1.GovernanceServiceOuterClass.PublishReleaseRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getPublishReleaseMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.GovernanceServiceOuterClass.ProposeMappingUpdateResponse proposeMappingUpdate(oscal.services.v1.GovernanceServiceOuterClass.ProposeMappingUpdateRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getProposeMappingUpdateMethod(), getCallOptions(), request);
    }
  }

  /**
   * A stub to allow clients to do ListenableFuture-style rpc calls to service GovernanceService.
   * <pre>
   * GovernanceService provides advanced governance, ingestion, and semantic
   * search operations over the OSCALify knowledge graph.
   * </pre>
   */
  public static final class GovernanceServiceFutureStub
      extends io.grpc.stub.AbstractFutureStub<GovernanceServiceFutureStub> {
    private GovernanceServiceFutureStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected GovernanceServiceFutureStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new GovernanceServiceFutureStub(channel, callOptions);
    }

    /**
     * <pre>
     * --- Entity CRUD ---
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.GovernanceServiceOuterClass.CreateEntityResponse> createEntity(
        oscal.services.v1.GovernanceServiceOuterClass.CreateEntityRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getCreateEntityMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.GovernanceServiceOuterClass.GetEntityResponse> getEntity(
        oscal.services.v1.GovernanceServiceOuterClass.GetEntityRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getGetEntityMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.GovernanceServiceOuterClass.UpdateEntityResponse> updateEntity(
        oscal.services.v1.GovernanceServiceOuterClass.UpdateEntityRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getUpdateEntityMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.GovernanceServiceOuterClass.ListEntitiesResponse> listEntities(
        oscal.services.v1.GovernanceServiceOuterClass.ListEntitiesRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getListEntitiesMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * --- Snapshots &amp; Releases ---
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.GovernanceServiceOuterClass.CreateSnapshotResponse> createSnapshot(
        oscal.services.v1.GovernanceServiceOuterClass.CreateSnapshotRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getCreateSnapshotMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.GovernanceServiceOuterClass.GetSnapshotResponse> getSnapshot(
        oscal.services.v1.GovernanceServiceOuterClass.GetSnapshotRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getGetSnapshotMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.GovernanceServiceOuterClass.ListSnapshotsResponse> listSnapshots(
        oscal.services.v1.GovernanceServiceOuterClass.ListSnapshotsRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getListSnapshotsMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.GovernanceServiceOuterClass.CreateReleaseResponse> createRelease(
        oscal.services.v1.GovernanceServiceOuterClass.CreateReleaseRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getCreateReleaseMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.GovernanceServiceOuterClass.ListReleasesResponse> listReleases(
        oscal.services.v1.GovernanceServiceOuterClass.ListReleasesRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getListReleasesMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * --- Ingestion ---
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.GovernanceServiceOuterClass.IngestRequirementsResponse> ingestRequirements(
        oscal.services.v1.GovernanceServiceOuterClass.IngestRequirementsRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getIngestRequirementsMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * --- Semantic Search ---
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.GovernanceServiceOuterClass.SemanticSearchResponse> semanticSearch(
        oscal.services.v1.GovernanceServiceOuterClass.SemanticSearchRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getSemanticSearchMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * --- OSCAL Generation ---
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.GovernanceServiceOuterClass.GenerateCatalogResponse> generateCatalog(
        oscal.services.v1.GovernanceServiceOuterClass.GenerateCatalogRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getGenerateCatalogMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.GovernanceServiceOuterClass.GenerateProfileResponse> generateProfile(
        oscal.services.v1.GovernanceServiceOuterClass.GenerateProfileRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getGenerateProfileMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.GovernanceServiceOuterClass.GenerateMappingsResponse> generateMappings(
        oscal.services.v1.GovernanceServiceOuterClass.GenerateMappingsRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getGenerateMappingsMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.GovernanceServiceOuterClass.GenerateSSPResponse> generateSSP(
        oscal.services.v1.GovernanceServiceOuterClass.GenerateSSPRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getGenerateSSPMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.GovernanceServiceOuterClass.GenerateComponentDefinitionResponse> generateComponentDefinition(
        oscal.services.v1.GovernanceServiceOuterClass.GenerateComponentDefinitionRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getGenerateComponentDefinitionMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentPlanResponse> generateAssessmentPlan(
        oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentPlanRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getGenerateAssessmentPlanMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.GovernanceServiceOuterClass.GeneratePOAMResponse> generatePOAM(
        oscal.services.v1.GovernanceServiceOuterClass.GeneratePOAMRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getGeneratePOAMMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentResultsResponse> generateAssessmentResults(
        oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentResultsRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getGenerateAssessmentResultsMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * --- Framework Ingestion ---
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.GovernanceServiceOuterClass.BulkIngestFrameworksResponse> bulkIngestFrameworks(
        oscal.services.v1.GovernanceServiceOuterClass.BulkIngestFrameworksRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getBulkIngestFrameworksMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.GovernanceServiceOuterClass.ListFrameworksResponse> listFrameworks(
        oscal.services.v1.GovernanceServiceOuterClass.ListFrameworksRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getListFrameworksMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.GovernanceServiceOuterClass.GetFrameworkResponse> getFramework(
        oscal.services.v1.GovernanceServiceOuterClass.GetFrameworkRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getGetFrameworkMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * --- Cross-Framework Mappings ---
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.GovernanceServiceOuterClass.GenerateCrossFrameworkMappingsResponse> generateCrossFrameworkMappings(
        oscal.services.v1.GovernanceServiceOuterClass.GenerateCrossFrameworkMappingsRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getGenerateCrossFrameworkMappingsMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * --- Reconciler ---
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.GovernanceServiceOuterClass.ProposeResponse> propose(
        oscal.services.v1.GovernanceServiceOuterClass.ProposeRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getProposeMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.GovernanceServiceOuterClass.ListConflictsResponse> listConflicts(
        oscal.services.v1.GovernanceServiceOuterClass.ListConflictsRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getListConflictsMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.GovernanceServiceOuterClass.ResolveConflictResponse> resolveConflict(
        oscal.services.v1.GovernanceServiceOuterClass.ResolveConflictRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getResolveConflictMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * --- GitOps ---
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.GovernanceServiceOuterClass.PublishReleaseResponse> publishRelease(
        oscal.services.v1.GovernanceServiceOuterClass.PublishReleaseRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getPublishReleaseMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.GovernanceServiceOuterClass.ProposeMappingUpdateResponse> proposeMappingUpdate(
        oscal.services.v1.GovernanceServiceOuterClass.ProposeMappingUpdateRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getProposeMappingUpdateMethod(), getCallOptions()), request);
    }
  }

  private static final int METHODID_CREATE_ENTITY = 0;
  private static final int METHODID_GET_ENTITY = 1;
  private static final int METHODID_UPDATE_ENTITY = 2;
  private static final int METHODID_LIST_ENTITIES = 3;
  private static final int METHODID_CREATE_SNAPSHOT = 4;
  private static final int METHODID_GET_SNAPSHOT = 5;
  private static final int METHODID_LIST_SNAPSHOTS = 6;
  private static final int METHODID_CREATE_RELEASE = 7;
  private static final int METHODID_LIST_RELEASES = 8;
  private static final int METHODID_INGEST_REQUIREMENTS = 9;
  private static final int METHODID_SEMANTIC_SEARCH = 10;
  private static final int METHODID_GENERATE_CATALOG = 11;
  private static final int METHODID_GENERATE_PROFILE = 12;
  private static final int METHODID_GENERATE_MAPPINGS = 13;
  private static final int METHODID_GENERATE_SSP = 14;
  private static final int METHODID_GENERATE_COMPONENT_DEFINITION = 15;
  private static final int METHODID_GENERATE_ASSESSMENT_PLAN = 16;
  private static final int METHODID_GENERATE_POAM = 17;
  private static final int METHODID_GENERATE_ASSESSMENT_RESULTS = 18;
  private static final int METHODID_BULK_INGEST_FRAMEWORKS = 19;
  private static final int METHODID_LIST_FRAMEWORKS = 20;
  private static final int METHODID_GET_FRAMEWORK = 21;
  private static final int METHODID_GENERATE_CROSS_FRAMEWORK_MAPPINGS = 22;
  private static final int METHODID_PROPOSE = 23;
  private static final int METHODID_LIST_CONFLICTS = 24;
  private static final int METHODID_RESOLVE_CONFLICT = 25;
  private static final int METHODID_PUBLISH_RELEASE = 26;
  private static final int METHODID_PROPOSE_MAPPING_UPDATE = 27;

  private static final class MethodHandlers<Req, Resp> implements
      io.grpc.stub.ServerCalls.UnaryMethod<Req, Resp>,
      io.grpc.stub.ServerCalls.ServerStreamingMethod<Req, Resp>,
      io.grpc.stub.ServerCalls.ClientStreamingMethod<Req, Resp>,
      io.grpc.stub.ServerCalls.BidiStreamingMethod<Req, Resp> {
    private final AsyncService serviceImpl;
    private final int methodId;

    MethodHandlers(AsyncService serviceImpl, int methodId) {
      this.serviceImpl = serviceImpl;
      this.methodId = methodId;
    }

    @java.lang.Override
    @java.lang.SuppressWarnings("unchecked")
    public void invoke(Req request, io.grpc.stub.StreamObserver<Resp> responseObserver) {
      switch (methodId) {
        case METHODID_CREATE_ENTITY:
          serviceImpl.createEntity((oscal.services.v1.GovernanceServiceOuterClass.CreateEntityRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.CreateEntityResponse>) responseObserver);
          break;
        case METHODID_GET_ENTITY:
          serviceImpl.getEntity((oscal.services.v1.GovernanceServiceOuterClass.GetEntityRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GetEntityResponse>) responseObserver);
          break;
        case METHODID_UPDATE_ENTITY:
          serviceImpl.updateEntity((oscal.services.v1.GovernanceServiceOuterClass.UpdateEntityRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.UpdateEntityResponse>) responseObserver);
          break;
        case METHODID_LIST_ENTITIES:
          serviceImpl.listEntities((oscal.services.v1.GovernanceServiceOuterClass.ListEntitiesRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.ListEntitiesResponse>) responseObserver);
          break;
        case METHODID_CREATE_SNAPSHOT:
          serviceImpl.createSnapshot((oscal.services.v1.GovernanceServiceOuterClass.CreateSnapshotRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.CreateSnapshotResponse>) responseObserver);
          break;
        case METHODID_GET_SNAPSHOT:
          serviceImpl.getSnapshot((oscal.services.v1.GovernanceServiceOuterClass.GetSnapshotRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GetSnapshotResponse>) responseObserver);
          break;
        case METHODID_LIST_SNAPSHOTS:
          serviceImpl.listSnapshots((oscal.services.v1.GovernanceServiceOuterClass.ListSnapshotsRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.ListSnapshotsResponse>) responseObserver);
          break;
        case METHODID_CREATE_RELEASE:
          serviceImpl.createRelease((oscal.services.v1.GovernanceServiceOuterClass.CreateReleaseRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.CreateReleaseResponse>) responseObserver);
          break;
        case METHODID_LIST_RELEASES:
          serviceImpl.listReleases((oscal.services.v1.GovernanceServiceOuterClass.ListReleasesRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.ListReleasesResponse>) responseObserver);
          break;
        case METHODID_INGEST_REQUIREMENTS:
          serviceImpl.ingestRequirements((oscal.services.v1.GovernanceServiceOuterClass.IngestRequirementsRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.IngestRequirementsResponse>) responseObserver);
          break;
        case METHODID_SEMANTIC_SEARCH:
          serviceImpl.semanticSearch((oscal.services.v1.GovernanceServiceOuterClass.SemanticSearchRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.SemanticSearchResponse>) responseObserver);
          break;
        case METHODID_GENERATE_CATALOG:
          serviceImpl.generateCatalog((oscal.services.v1.GovernanceServiceOuterClass.GenerateCatalogRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GenerateCatalogResponse>) responseObserver);
          break;
        case METHODID_GENERATE_PROFILE:
          serviceImpl.generateProfile((oscal.services.v1.GovernanceServiceOuterClass.GenerateProfileRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GenerateProfileResponse>) responseObserver);
          break;
        case METHODID_GENERATE_MAPPINGS:
          serviceImpl.generateMappings((oscal.services.v1.GovernanceServiceOuterClass.GenerateMappingsRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GenerateMappingsResponse>) responseObserver);
          break;
        case METHODID_GENERATE_SSP:
          serviceImpl.generateSSP((oscal.services.v1.GovernanceServiceOuterClass.GenerateSSPRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GenerateSSPResponse>) responseObserver);
          break;
        case METHODID_GENERATE_COMPONENT_DEFINITION:
          serviceImpl.generateComponentDefinition((oscal.services.v1.GovernanceServiceOuterClass.GenerateComponentDefinitionRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GenerateComponentDefinitionResponse>) responseObserver);
          break;
        case METHODID_GENERATE_ASSESSMENT_PLAN:
          serviceImpl.generateAssessmentPlan((oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentPlanRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentPlanResponse>) responseObserver);
          break;
        case METHODID_GENERATE_POAM:
          serviceImpl.generatePOAM((oscal.services.v1.GovernanceServiceOuterClass.GeneratePOAMRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GeneratePOAMResponse>) responseObserver);
          break;
        case METHODID_GENERATE_ASSESSMENT_RESULTS:
          serviceImpl.generateAssessmentResults((oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentResultsRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentResultsResponse>) responseObserver);
          break;
        case METHODID_BULK_INGEST_FRAMEWORKS:
          serviceImpl.bulkIngestFrameworks((oscal.services.v1.GovernanceServiceOuterClass.BulkIngestFrameworksRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.BulkIngestFrameworksResponse>) responseObserver);
          break;
        case METHODID_LIST_FRAMEWORKS:
          serviceImpl.listFrameworks((oscal.services.v1.GovernanceServiceOuterClass.ListFrameworksRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.ListFrameworksResponse>) responseObserver);
          break;
        case METHODID_GET_FRAMEWORK:
          serviceImpl.getFramework((oscal.services.v1.GovernanceServiceOuterClass.GetFrameworkRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GetFrameworkResponse>) responseObserver);
          break;
        case METHODID_GENERATE_CROSS_FRAMEWORK_MAPPINGS:
          serviceImpl.generateCrossFrameworkMappings((oscal.services.v1.GovernanceServiceOuterClass.GenerateCrossFrameworkMappingsRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.GenerateCrossFrameworkMappingsResponse>) responseObserver);
          break;
        case METHODID_PROPOSE:
          serviceImpl.propose((oscal.services.v1.GovernanceServiceOuterClass.ProposeRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.ProposeResponse>) responseObserver);
          break;
        case METHODID_LIST_CONFLICTS:
          serviceImpl.listConflicts((oscal.services.v1.GovernanceServiceOuterClass.ListConflictsRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.ListConflictsResponse>) responseObserver);
          break;
        case METHODID_RESOLVE_CONFLICT:
          serviceImpl.resolveConflict((oscal.services.v1.GovernanceServiceOuterClass.ResolveConflictRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.ResolveConflictResponse>) responseObserver);
          break;
        case METHODID_PUBLISH_RELEASE:
          serviceImpl.publishRelease((oscal.services.v1.GovernanceServiceOuterClass.PublishReleaseRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.PublishReleaseResponse>) responseObserver);
          break;
        case METHODID_PROPOSE_MAPPING_UPDATE:
          serviceImpl.proposeMappingUpdate((oscal.services.v1.GovernanceServiceOuterClass.ProposeMappingUpdateRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.GovernanceServiceOuterClass.ProposeMappingUpdateResponse>) responseObserver);
          break;
        default:
          throw new AssertionError();
      }
    }

    @java.lang.Override
    @java.lang.SuppressWarnings("unchecked")
    public io.grpc.stub.StreamObserver<Req> invoke(
        io.grpc.stub.StreamObserver<Resp> responseObserver) {
      switch (methodId) {
        default:
          throw new AssertionError();
      }
    }
  }

  public static final io.grpc.ServerServiceDefinition bindService(AsyncService service) {
    return io.grpc.ServerServiceDefinition.builder(getServiceDescriptor())
        .addMethod(
          getCreateEntityMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.GovernanceServiceOuterClass.CreateEntityRequest,
              oscal.services.v1.GovernanceServiceOuterClass.CreateEntityResponse>(
                service, METHODID_CREATE_ENTITY)))
        .addMethod(
          getGetEntityMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.GovernanceServiceOuterClass.GetEntityRequest,
              oscal.services.v1.GovernanceServiceOuterClass.GetEntityResponse>(
                service, METHODID_GET_ENTITY)))
        .addMethod(
          getUpdateEntityMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.GovernanceServiceOuterClass.UpdateEntityRequest,
              oscal.services.v1.GovernanceServiceOuterClass.UpdateEntityResponse>(
                service, METHODID_UPDATE_ENTITY)))
        .addMethod(
          getListEntitiesMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.GovernanceServiceOuterClass.ListEntitiesRequest,
              oscal.services.v1.GovernanceServiceOuterClass.ListEntitiesResponse>(
                service, METHODID_LIST_ENTITIES)))
        .addMethod(
          getCreateSnapshotMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.GovernanceServiceOuterClass.CreateSnapshotRequest,
              oscal.services.v1.GovernanceServiceOuterClass.CreateSnapshotResponse>(
                service, METHODID_CREATE_SNAPSHOT)))
        .addMethod(
          getGetSnapshotMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.GovernanceServiceOuterClass.GetSnapshotRequest,
              oscal.services.v1.GovernanceServiceOuterClass.GetSnapshotResponse>(
                service, METHODID_GET_SNAPSHOT)))
        .addMethod(
          getListSnapshotsMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.GovernanceServiceOuterClass.ListSnapshotsRequest,
              oscal.services.v1.GovernanceServiceOuterClass.ListSnapshotsResponse>(
                service, METHODID_LIST_SNAPSHOTS)))
        .addMethod(
          getCreateReleaseMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.GovernanceServiceOuterClass.CreateReleaseRequest,
              oscal.services.v1.GovernanceServiceOuterClass.CreateReleaseResponse>(
                service, METHODID_CREATE_RELEASE)))
        .addMethod(
          getListReleasesMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.GovernanceServiceOuterClass.ListReleasesRequest,
              oscal.services.v1.GovernanceServiceOuterClass.ListReleasesResponse>(
                service, METHODID_LIST_RELEASES)))
        .addMethod(
          getIngestRequirementsMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.GovernanceServiceOuterClass.IngestRequirementsRequest,
              oscal.services.v1.GovernanceServiceOuterClass.IngestRequirementsResponse>(
                service, METHODID_INGEST_REQUIREMENTS)))
        .addMethod(
          getSemanticSearchMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.GovernanceServiceOuterClass.SemanticSearchRequest,
              oscal.services.v1.GovernanceServiceOuterClass.SemanticSearchResponse>(
                service, METHODID_SEMANTIC_SEARCH)))
        .addMethod(
          getGenerateCatalogMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.GovernanceServiceOuterClass.GenerateCatalogRequest,
              oscal.services.v1.GovernanceServiceOuterClass.GenerateCatalogResponse>(
                service, METHODID_GENERATE_CATALOG)))
        .addMethod(
          getGenerateProfileMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.GovernanceServiceOuterClass.GenerateProfileRequest,
              oscal.services.v1.GovernanceServiceOuterClass.GenerateProfileResponse>(
                service, METHODID_GENERATE_PROFILE)))
        .addMethod(
          getGenerateMappingsMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.GovernanceServiceOuterClass.GenerateMappingsRequest,
              oscal.services.v1.GovernanceServiceOuterClass.GenerateMappingsResponse>(
                service, METHODID_GENERATE_MAPPINGS)))
        .addMethod(
          getGenerateSSPMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.GovernanceServiceOuterClass.GenerateSSPRequest,
              oscal.services.v1.GovernanceServiceOuterClass.GenerateSSPResponse>(
                service, METHODID_GENERATE_SSP)))
        .addMethod(
          getGenerateComponentDefinitionMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.GovernanceServiceOuterClass.GenerateComponentDefinitionRequest,
              oscal.services.v1.GovernanceServiceOuterClass.GenerateComponentDefinitionResponse>(
                service, METHODID_GENERATE_COMPONENT_DEFINITION)))
        .addMethod(
          getGenerateAssessmentPlanMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentPlanRequest,
              oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentPlanResponse>(
                service, METHODID_GENERATE_ASSESSMENT_PLAN)))
        .addMethod(
          getGeneratePOAMMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.GovernanceServiceOuterClass.GeneratePOAMRequest,
              oscal.services.v1.GovernanceServiceOuterClass.GeneratePOAMResponse>(
                service, METHODID_GENERATE_POAM)))
        .addMethod(
          getGenerateAssessmentResultsMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentResultsRequest,
              oscal.services.v1.GovernanceServiceOuterClass.GenerateAssessmentResultsResponse>(
                service, METHODID_GENERATE_ASSESSMENT_RESULTS)))
        .addMethod(
          getBulkIngestFrameworksMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.GovernanceServiceOuterClass.BulkIngestFrameworksRequest,
              oscal.services.v1.GovernanceServiceOuterClass.BulkIngestFrameworksResponse>(
                service, METHODID_BULK_INGEST_FRAMEWORKS)))
        .addMethod(
          getListFrameworksMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.GovernanceServiceOuterClass.ListFrameworksRequest,
              oscal.services.v1.GovernanceServiceOuterClass.ListFrameworksResponse>(
                service, METHODID_LIST_FRAMEWORKS)))
        .addMethod(
          getGetFrameworkMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.GovernanceServiceOuterClass.GetFrameworkRequest,
              oscal.services.v1.GovernanceServiceOuterClass.GetFrameworkResponse>(
                service, METHODID_GET_FRAMEWORK)))
        .addMethod(
          getGenerateCrossFrameworkMappingsMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.GovernanceServiceOuterClass.GenerateCrossFrameworkMappingsRequest,
              oscal.services.v1.GovernanceServiceOuterClass.GenerateCrossFrameworkMappingsResponse>(
                service, METHODID_GENERATE_CROSS_FRAMEWORK_MAPPINGS)))
        .addMethod(
          getProposeMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.GovernanceServiceOuterClass.ProposeRequest,
              oscal.services.v1.GovernanceServiceOuterClass.ProposeResponse>(
                service, METHODID_PROPOSE)))
        .addMethod(
          getListConflictsMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.GovernanceServiceOuterClass.ListConflictsRequest,
              oscal.services.v1.GovernanceServiceOuterClass.ListConflictsResponse>(
                service, METHODID_LIST_CONFLICTS)))
        .addMethod(
          getResolveConflictMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.GovernanceServiceOuterClass.ResolveConflictRequest,
              oscal.services.v1.GovernanceServiceOuterClass.ResolveConflictResponse>(
                service, METHODID_RESOLVE_CONFLICT)))
        .addMethod(
          getPublishReleaseMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.GovernanceServiceOuterClass.PublishReleaseRequest,
              oscal.services.v1.GovernanceServiceOuterClass.PublishReleaseResponse>(
                service, METHODID_PUBLISH_RELEASE)))
        .addMethod(
          getProposeMappingUpdateMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.GovernanceServiceOuterClass.ProposeMappingUpdateRequest,
              oscal.services.v1.GovernanceServiceOuterClass.ProposeMappingUpdateResponse>(
                service, METHODID_PROPOSE_MAPPING_UPDATE)))
        .build();
  }

  private static abstract class GovernanceServiceBaseDescriptorSupplier
      implements io.grpc.protobuf.ProtoFileDescriptorSupplier, io.grpc.protobuf.ProtoServiceDescriptorSupplier {
    GovernanceServiceBaseDescriptorSupplier() {}

    @java.lang.Override
    public com.google.protobuf.Descriptors.FileDescriptor getFileDescriptor() {
      return oscal.services.v1.GovernanceServiceOuterClass.getDescriptor();
    }

    @java.lang.Override
    public com.google.protobuf.Descriptors.ServiceDescriptor getServiceDescriptor() {
      return getFileDescriptor().findServiceByName("GovernanceService");
    }
  }

  private static final class GovernanceServiceFileDescriptorSupplier
      extends GovernanceServiceBaseDescriptorSupplier {
    GovernanceServiceFileDescriptorSupplier() {}
  }

  private static final class GovernanceServiceMethodDescriptorSupplier
      extends GovernanceServiceBaseDescriptorSupplier
      implements io.grpc.protobuf.ProtoMethodDescriptorSupplier {
    private final java.lang.String methodName;

    GovernanceServiceMethodDescriptorSupplier(java.lang.String methodName) {
      this.methodName = methodName;
    }

    @java.lang.Override
    public com.google.protobuf.Descriptors.MethodDescriptor getMethodDescriptor() {
      return getServiceDescriptor().findMethodByName(methodName);
    }
  }

  private static volatile io.grpc.ServiceDescriptor serviceDescriptor;

  public static io.grpc.ServiceDescriptor getServiceDescriptor() {
    io.grpc.ServiceDescriptor result = serviceDescriptor;
    if (result == null) {
      synchronized (GovernanceServiceGrpc.class) {
        result = serviceDescriptor;
        if (result == null) {
          serviceDescriptor = result = io.grpc.ServiceDescriptor.newBuilder(SERVICE_NAME)
              .setSchemaDescriptor(new GovernanceServiceFileDescriptorSupplier())
              .addMethod(getCreateEntityMethod())
              .addMethod(getGetEntityMethod())
              .addMethod(getUpdateEntityMethod())
              .addMethod(getListEntitiesMethod())
              .addMethod(getCreateSnapshotMethod())
              .addMethod(getGetSnapshotMethod())
              .addMethod(getListSnapshotsMethod())
              .addMethod(getCreateReleaseMethod())
              .addMethod(getListReleasesMethod())
              .addMethod(getIngestRequirementsMethod())
              .addMethod(getSemanticSearchMethod())
              .addMethod(getGenerateCatalogMethod())
              .addMethod(getGenerateProfileMethod())
              .addMethod(getGenerateMappingsMethod())
              .addMethod(getGenerateSSPMethod())
              .addMethod(getGenerateComponentDefinitionMethod())
              .addMethod(getGenerateAssessmentPlanMethod())
              .addMethod(getGeneratePOAMMethod())
              .addMethod(getGenerateAssessmentResultsMethod())
              .addMethod(getBulkIngestFrameworksMethod())
              .addMethod(getListFrameworksMethod())
              .addMethod(getGetFrameworkMethod())
              .addMethod(getGenerateCrossFrameworkMappingsMethod())
              .addMethod(getProposeMethod())
              .addMethod(getListConflictsMethod())
              .addMethod(getResolveConflictMethod())
              .addMethod(getPublishReleaseMethod())
              .addMethod(getProposeMappingUpdateMethod())
              .build();
        }
      }
    }
    return result;
  }
}
