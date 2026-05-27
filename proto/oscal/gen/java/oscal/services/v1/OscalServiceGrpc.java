package oscal.services.v1;

import static io.grpc.MethodDescriptor.generateFullMethodName;

/**
 * <pre>
 * OSCAL Service provides CRUD operations for all OSCAL models
 * </pre>
 */
@io.grpc.stub.annotations.GrpcGenerated
public final class OscalServiceGrpc {

  private OscalServiceGrpc() {}

  public static final java.lang.String SERVICE_NAME = "oscal.services.v1.OscalService";

  // Static method descriptors that strictly reflect the proto.
  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.GetCatalogRequest,
      oscal.services.v1.OscalServiceOuterClass.GetCatalogResponse> getGetCatalogMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GetCatalog",
      requestType = oscal.services.v1.OscalServiceOuterClass.GetCatalogRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.GetCatalogResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.GetCatalogRequest,
      oscal.services.v1.OscalServiceOuterClass.GetCatalogResponse> getGetCatalogMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.GetCatalogRequest, oscal.services.v1.OscalServiceOuterClass.GetCatalogResponse> getGetCatalogMethod;
    if ((getGetCatalogMethod = OscalServiceGrpc.getGetCatalogMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getGetCatalogMethod = OscalServiceGrpc.getGetCatalogMethod) == null) {
          OscalServiceGrpc.getGetCatalogMethod = getGetCatalogMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.GetCatalogRequest, oscal.services.v1.OscalServiceOuterClass.GetCatalogResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GetCatalog"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.GetCatalogRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.GetCatalogResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("GetCatalog"))
              .build();
        }
      }
    }
    return getGetCatalogMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.ListCatalogsRequest,
      oscal.services.v1.OscalServiceOuterClass.ListCatalogsResponse> getListCatalogsMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "ListCatalogs",
      requestType = oscal.services.v1.OscalServiceOuterClass.ListCatalogsRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.ListCatalogsResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.ListCatalogsRequest,
      oscal.services.v1.OscalServiceOuterClass.ListCatalogsResponse> getListCatalogsMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.ListCatalogsRequest, oscal.services.v1.OscalServiceOuterClass.ListCatalogsResponse> getListCatalogsMethod;
    if ((getListCatalogsMethod = OscalServiceGrpc.getListCatalogsMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getListCatalogsMethod = OscalServiceGrpc.getListCatalogsMethod) == null) {
          OscalServiceGrpc.getListCatalogsMethod = getListCatalogsMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.ListCatalogsRequest, oscal.services.v1.OscalServiceOuterClass.ListCatalogsResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "ListCatalogs"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.ListCatalogsRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.ListCatalogsResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("ListCatalogs"))
              .build();
        }
      }
    }
    return getListCatalogsMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.CreateCatalogRequest,
      oscal.services.v1.OscalServiceOuterClass.CreateCatalogResponse> getCreateCatalogMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "CreateCatalog",
      requestType = oscal.services.v1.OscalServiceOuterClass.CreateCatalogRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.CreateCatalogResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.CreateCatalogRequest,
      oscal.services.v1.OscalServiceOuterClass.CreateCatalogResponse> getCreateCatalogMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.CreateCatalogRequest, oscal.services.v1.OscalServiceOuterClass.CreateCatalogResponse> getCreateCatalogMethod;
    if ((getCreateCatalogMethod = OscalServiceGrpc.getCreateCatalogMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getCreateCatalogMethod = OscalServiceGrpc.getCreateCatalogMethod) == null) {
          OscalServiceGrpc.getCreateCatalogMethod = getCreateCatalogMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.CreateCatalogRequest, oscal.services.v1.OscalServiceOuterClass.CreateCatalogResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "CreateCatalog"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.CreateCatalogRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.CreateCatalogResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("CreateCatalog"))
              .build();
        }
      }
    }
    return getCreateCatalogMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.UpdateCatalogRequest,
      oscal.services.v1.OscalServiceOuterClass.UpdateCatalogResponse> getUpdateCatalogMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "UpdateCatalog",
      requestType = oscal.services.v1.OscalServiceOuterClass.UpdateCatalogRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.UpdateCatalogResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.UpdateCatalogRequest,
      oscal.services.v1.OscalServiceOuterClass.UpdateCatalogResponse> getUpdateCatalogMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.UpdateCatalogRequest, oscal.services.v1.OscalServiceOuterClass.UpdateCatalogResponse> getUpdateCatalogMethod;
    if ((getUpdateCatalogMethod = OscalServiceGrpc.getUpdateCatalogMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getUpdateCatalogMethod = OscalServiceGrpc.getUpdateCatalogMethod) == null) {
          OscalServiceGrpc.getUpdateCatalogMethod = getUpdateCatalogMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.UpdateCatalogRequest, oscal.services.v1.OscalServiceOuterClass.UpdateCatalogResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "UpdateCatalog"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.UpdateCatalogRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.UpdateCatalogResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("UpdateCatalog"))
              .build();
        }
      }
    }
    return getUpdateCatalogMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.DeleteCatalogRequest,
      oscal.services.v1.OscalServiceOuterClass.DeleteCatalogResponse> getDeleteCatalogMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "DeleteCatalog",
      requestType = oscal.services.v1.OscalServiceOuterClass.DeleteCatalogRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.DeleteCatalogResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.DeleteCatalogRequest,
      oscal.services.v1.OscalServiceOuterClass.DeleteCatalogResponse> getDeleteCatalogMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.DeleteCatalogRequest, oscal.services.v1.OscalServiceOuterClass.DeleteCatalogResponse> getDeleteCatalogMethod;
    if ((getDeleteCatalogMethod = OscalServiceGrpc.getDeleteCatalogMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getDeleteCatalogMethod = OscalServiceGrpc.getDeleteCatalogMethod) == null) {
          OscalServiceGrpc.getDeleteCatalogMethod = getDeleteCatalogMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.DeleteCatalogRequest, oscal.services.v1.OscalServiceOuterClass.DeleteCatalogResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "DeleteCatalog"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.DeleteCatalogRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.DeleteCatalogResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("DeleteCatalog"))
              .build();
        }
      }
    }
    return getDeleteCatalogMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.GetProfileRequest,
      oscal.services.v1.OscalServiceOuterClass.GetProfileResponse> getGetProfileMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GetProfile",
      requestType = oscal.services.v1.OscalServiceOuterClass.GetProfileRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.GetProfileResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.GetProfileRequest,
      oscal.services.v1.OscalServiceOuterClass.GetProfileResponse> getGetProfileMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.GetProfileRequest, oscal.services.v1.OscalServiceOuterClass.GetProfileResponse> getGetProfileMethod;
    if ((getGetProfileMethod = OscalServiceGrpc.getGetProfileMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getGetProfileMethod = OscalServiceGrpc.getGetProfileMethod) == null) {
          OscalServiceGrpc.getGetProfileMethod = getGetProfileMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.GetProfileRequest, oscal.services.v1.OscalServiceOuterClass.GetProfileResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GetProfile"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.GetProfileRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.GetProfileResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("GetProfile"))
              .build();
        }
      }
    }
    return getGetProfileMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.ListProfilesRequest,
      oscal.services.v1.OscalServiceOuterClass.ListProfilesResponse> getListProfilesMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "ListProfiles",
      requestType = oscal.services.v1.OscalServiceOuterClass.ListProfilesRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.ListProfilesResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.ListProfilesRequest,
      oscal.services.v1.OscalServiceOuterClass.ListProfilesResponse> getListProfilesMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.ListProfilesRequest, oscal.services.v1.OscalServiceOuterClass.ListProfilesResponse> getListProfilesMethod;
    if ((getListProfilesMethod = OscalServiceGrpc.getListProfilesMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getListProfilesMethod = OscalServiceGrpc.getListProfilesMethod) == null) {
          OscalServiceGrpc.getListProfilesMethod = getListProfilesMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.ListProfilesRequest, oscal.services.v1.OscalServiceOuterClass.ListProfilesResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "ListProfiles"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.ListProfilesRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.ListProfilesResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("ListProfiles"))
              .build();
        }
      }
    }
    return getListProfilesMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.CreateProfileRequest,
      oscal.services.v1.OscalServiceOuterClass.CreateProfileResponse> getCreateProfileMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "CreateProfile",
      requestType = oscal.services.v1.OscalServiceOuterClass.CreateProfileRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.CreateProfileResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.CreateProfileRequest,
      oscal.services.v1.OscalServiceOuterClass.CreateProfileResponse> getCreateProfileMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.CreateProfileRequest, oscal.services.v1.OscalServiceOuterClass.CreateProfileResponse> getCreateProfileMethod;
    if ((getCreateProfileMethod = OscalServiceGrpc.getCreateProfileMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getCreateProfileMethod = OscalServiceGrpc.getCreateProfileMethod) == null) {
          OscalServiceGrpc.getCreateProfileMethod = getCreateProfileMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.CreateProfileRequest, oscal.services.v1.OscalServiceOuterClass.CreateProfileResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "CreateProfile"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.CreateProfileRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.CreateProfileResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("CreateProfile"))
              .build();
        }
      }
    }
    return getCreateProfileMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.UpdateProfileRequest,
      oscal.services.v1.OscalServiceOuterClass.UpdateProfileResponse> getUpdateProfileMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "UpdateProfile",
      requestType = oscal.services.v1.OscalServiceOuterClass.UpdateProfileRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.UpdateProfileResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.UpdateProfileRequest,
      oscal.services.v1.OscalServiceOuterClass.UpdateProfileResponse> getUpdateProfileMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.UpdateProfileRequest, oscal.services.v1.OscalServiceOuterClass.UpdateProfileResponse> getUpdateProfileMethod;
    if ((getUpdateProfileMethod = OscalServiceGrpc.getUpdateProfileMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getUpdateProfileMethod = OscalServiceGrpc.getUpdateProfileMethod) == null) {
          OscalServiceGrpc.getUpdateProfileMethod = getUpdateProfileMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.UpdateProfileRequest, oscal.services.v1.OscalServiceOuterClass.UpdateProfileResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "UpdateProfile"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.UpdateProfileRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.UpdateProfileResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("UpdateProfile"))
              .build();
        }
      }
    }
    return getUpdateProfileMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.DeleteProfileRequest,
      oscal.services.v1.OscalServiceOuterClass.DeleteProfileResponse> getDeleteProfileMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "DeleteProfile",
      requestType = oscal.services.v1.OscalServiceOuterClass.DeleteProfileRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.DeleteProfileResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.DeleteProfileRequest,
      oscal.services.v1.OscalServiceOuterClass.DeleteProfileResponse> getDeleteProfileMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.DeleteProfileRequest, oscal.services.v1.OscalServiceOuterClass.DeleteProfileResponse> getDeleteProfileMethod;
    if ((getDeleteProfileMethod = OscalServiceGrpc.getDeleteProfileMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getDeleteProfileMethod = OscalServiceGrpc.getDeleteProfileMethod) == null) {
          OscalServiceGrpc.getDeleteProfileMethod = getDeleteProfileMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.DeleteProfileRequest, oscal.services.v1.OscalServiceOuterClass.DeleteProfileResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "DeleteProfile"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.DeleteProfileRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.DeleteProfileResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("DeleteProfile"))
              .build();
        }
      }
    }
    return getDeleteProfileMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.GetComponentDefinitionRequest,
      oscal.services.v1.OscalServiceOuterClass.GetComponentDefinitionResponse> getGetComponentDefinitionMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GetComponentDefinition",
      requestType = oscal.services.v1.OscalServiceOuterClass.GetComponentDefinitionRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.GetComponentDefinitionResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.GetComponentDefinitionRequest,
      oscal.services.v1.OscalServiceOuterClass.GetComponentDefinitionResponse> getGetComponentDefinitionMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.GetComponentDefinitionRequest, oscal.services.v1.OscalServiceOuterClass.GetComponentDefinitionResponse> getGetComponentDefinitionMethod;
    if ((getGetComponentDefinitionMethod = OscalServiceGrpc.getGetComponentDefinitionMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getGetComponentDefinitionMethod = OscalServiceGrpc.getGetComponentDefinitionMethod) == null) {
          OscalServiceGrpc.getGetComponentDefinitionMethod = getGetComponentDefinitionMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.GetComponentDefinitionRequest, oscal.services.v1.OscalServiceOuterClass.GetComponentDefinitionResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GetComponentDefinition"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.GetComponentDefinitionRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.GetComponentDefinitionResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("GetComponentDefinition"))
              .build();
        }
      }
    }
    return getGetComponentDefinitionMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.ListComponentDefinitionsRequest,
      oscal.services.v1.OscalServiceOuterClass.ListComponentDefinitionsResponse> getListComponentDefinitionsMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "ListComponentDefinitions",
      requestType = oscal.services.v1.OscalServiceOuterClass.ListComponentDefinitionsRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.ListComponentDefinitionsResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.ListComponentDefinitionsRequest,
      oscal.services.v1.OscalServiceOuterClass.ListComponentDefinitionsResponse> getListComponentDefinitionsMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.ListComponentDefinitionsRequest, oscal.services.v1.OscalServiceOuterClass.ListComponentDefinitionsResponse> getListComponentDefinitionsMethod;
    if ((getListComponentDefinitionsMethod = OscalServiceGrpc.getListComponentDefinitionsMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getListComponentDefinitionsMethod = OscalServiceGrpc.getListComponentDefinitionsMethod) == null) {
          OscalServiceGrpc.getListComponentDefinitionsMethod = getListComponentDefinitionsMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.ListComponentDefinitionsRequest, oscal.services.v1.OscalServiceOuterClass.ListComponentDefinitionsResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "ListComponentDefinitions"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.ListComponentDefinitionsRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.ListComponentDefinitionsResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("ListComponentDefinitions"))
              .build();
        }
      }
    }
    return getListComponentDefinitionsMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.CreateComponentDefinitionRequest,
      oscal.services.v1.OscalServiceOuterClass.CreateComponentDefinitionResponse> getCreateComponentDefinitionMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "CreateComponentDefinition",
      requestType = oscal.services.v1.OscalServiceOuterClass.CreateComponentDefinitionRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.CreateComponentDefinitionResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.CreateComponentDefinitionRequest,
      oscal.services.v1.OscalServiceOuterClass.CreateComponentDefinitionResponse> getCreateComponentDefinitionMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.CreateComponentDefinitionRequest, oscal.services.v1.OscalServiceOuterClass.CreateComponentDefinitionResponse> getCreateComponentDefinitionMethod;
    if ((getCreateComponentDefinitionMethod = OscalServiceGrpc.getCreateComponentDefinitionMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getCreateComponentDefinitionMethod = OscalServiceGrpc.getCreateComponentDefinitionMethod) == null) {
          OscalServiceGrpc.getCreateComponentDefinitionMethod = getCreateComponentDefinitionMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.CreateComponentDefinitionRequest, oscal.services.v1.OscalServiceOuterClass.CreateComponentDefinitionResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "CreateComponentDefinition"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.CreateComponentDefinitionRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.CreateComponentDefinitionResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("CreateComponentDefinition"))
              .build();
        }
      }
    }
    return getCreateComponentDefinitionMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.UpdateComponentDefinitionRequest,
      oscal.services.v1.OscalServiceOuterClass.UpdateComponentDefinitionResponse> getUpdateComponentDefinitionMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "UpdateComponentDefinition",
      requestType = oscal.services.v1.OscalServiceOuterClass.UpdateComponentDefinitionRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.UpdateComponentDefinitionResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.UpdateComponentDefinitionRequest,
      oscal.services.v1.OscalServiceOuterClass.UpdateComponentDefinitionResponse> getUpdateComponentDefinitionMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.UpdateComponentDefinitionRequest, oscal.services.v1.OscalServiceOuterClass.UpdateComponentDefinitionResponse> getUpdateComponentDefinitionMethod;
    if ((getUpdateComponentDefinitionMethod = OscalServiceGrpc.getUpdateComponentDefinitionMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getUpdateComponentDefinitionMethod = OscalServiceGrpc.getUpdateComponentDefinitionMethod) == null) {
          OscalServiceGrpc.getUpdateComponentDefinitionMethod = getUpdateComponentDefinitionMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.UpdateComponentDefinitionRequest, oscal.services.v1.OscalServiceOuterClass.UpdateComponentDefinitionResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "UpdateComponentDefinition"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.UpdateComponentDefinitionRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.UpdateComponentDefinitionResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("UpdateComponentDefinition"))
              .build();
        }
      }
    }
    return getUpdateComponentDefinitionMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.DeleteComponentDefinitionRequest,
      oscal.services.v1.OscalServiceOuterClass.DeleteComponentDefinitionResponse> getDeleteComponentDefinitionMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "DeleteComponentDefinition",
      requestType = oscal.services.v1.OscalServiceOuterClass.DeleteComponentDefinitionRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.DeleteComponentDefinitionResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.DeleteComponentDefinitionRequest,
      oscal.services.v1.OscalServiceOuterClass.DeleteComponentDefinitionResponse> getDeleteComponentDefinitionMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.DeleteComponentDefinitionRequest, oscal.services.v1.OscalServiceOuterClass.DeleteComponentDefinitionResponse> getDeleteComponentDefinitionMethod;
    if ((getDeleteComponentDefinitionMethod = OscalServiceGrpc.getDeleteComponentDefinitionMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getDeleteComponentDefinitionMethod = OscalServiceGrpc.getDeleteComponentDefinitionMethod) == null) {
          OscalServiceGrpc.getDeleteComponentDefinitionMethod = getDeleteComponentDefinitionMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.DeleteComponentDefinitionRequest, oscal.services.v1.OscalServiceOuterClass.DeleteComponentDefinitionResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "DeleteComponentDefinition"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.DeleteComponentDefinitionRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.DeleteComponentDefinitionResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("DeleteComponentDefinition"))
              .build();
        }
      }
    }
    return getDeleteComponentDefinitionMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.GetSspRequest,
      oscal.services.v1.OscalServiceOuterClass.GetSspResponse> getGetSspMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GetSsp",
      requestType = oscal.services.v1.OscalServiceOuterClass.GetSspRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.GetSspResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.GetSspRequest,
      oscal.services.v1.OscalServiceOuterClass.GetSspResponse> getGetSspMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.GetSspRequest, oscal.services.v1.OscalServiceOuterClass.GetSspResponse> getGetSspMethod;
    if ((getGetSspMethod = OscalServiceGrpc.getGetSspMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getGetSspMethod = OscalServiceGrpc.getGetSspMethod) == null) {
          OscalServiceGrpc.getGetSspMethod = getGetSspMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.GetSspRequest, oscal.services.v1.OscalServiceOuterClass.GetSspResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GetSsp"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.GetSspRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.GetSspResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("GetSsp"))
              .build();
        }
      }
    }
    return getGetSspMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.ListSspsRequest,
      oscal.services.v1.OscalServiceOuterClass.ListSspsResponse> getListSspsMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "ListSsps",
      requestType = oscal.services.v1.OscalServiceOuterClass.ListSspsRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.ListSspsResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.ListSspsRequest,
      oscal.services.v1.OscalServiceOuterClass.ListSspsResponse> getListSspsMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.ListSspsRequest, oscal.services.v1.OscalServiceOuterClass.ListSspsResponse> getListSspsMethod;
    if ((getListSspsMethod = OscalServiceGrpc.getListSspsMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getListSspsMethod = OscalServiceGrpc.getListSspsMethod) == null) {
          OscalServiceGrpc.getListSspsMethod = getListSspsMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.ListSspsRequest, oscal.services.v1.OscalServiceOuterClass.ListSspsResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "ListSsps"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.ListSspsRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.ListSspsResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("ListSsps"))
              .build();
        }
      }
    }
    return getListSspsMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.CreateSspRequest,
      oscal.services.v1.OscalServiceOuterClass.CreateSspResponse> getCreateSspMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "CreateSsp",
      requestType = oscal.services.v1.OscalServiceOuterClass.CreateSspRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.CreateSspResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.CreateSspRequest,
      oscal.services.v1.OscalServiceOuterClass.CreateSspResponse> getCreateSspMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.CreateSspRequest, oscal.services.v1.OscalServiceOuterClass.CreateSspResponse> getCreateSspMethod;
    if ((getCreateSspMethod = OscalServiceGrpc.getCreateSspMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getCreateSspMethod = OscalServiceGrpc.getCreateSspMethod) == null) {
          OscalServiceGrpc.getCreateSspMethod = getCreateSspMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.CreateSspRequest, oscal.services.v1.OscalServiceOuterClass.CreateSspResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "CreateSsp"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.CreateSspRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.CreateSspResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("CreateSsp"))
              .build();
        }
      }
    }
    return getCreateSspMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.UpdateSspRequest,
      oscal.services.v1.OscalServiceOuterClass.UpdateSspResponse> getUpdateSspMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "UpdateSsp",
      requestType = oscal.services.v1.OscalServiceOuterClass.UpdateSspRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.UpdateSspResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.UpdateSspRequest,
      oscal.services.v1.OscalServiceOuterClass.UpdateSspResponse> getUpdateSspMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.UpdateSspRequest, oscal.services.v1.OscalServiceOuterClass.UpdateSspResponse> getUpdateSspMethod;
    if ((getUpdateSspMethod = OscalServiceGrpc.getUpdateSspMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getUpdateSspMethod = OscalServiceGrpc.getUpdateSspMethod) == null) {
          OscalServiceGrpc.getUpdateSspMethod = getUpdateSspMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.UpdateSspRequest, oscal.services.v1.OscalServiceOuterClass.UpdateSspResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "UpdateSsp"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.UpdateSspRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.UpdateSspResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("UpdateSsp"))
              .build();
        }
      }
    }
    return getUpdateSspMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.DeleteSspRequest,
      oscal.services.v1.OscalServiceOuterClass.DeleteSspResponse> getDeleteSspMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "DeleteSsp",
      requestType = oscal.services.v1.OscalServiceOuterClass.DeleteSspRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.DeleteSspResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.DeleteSspRequest,
      oscal.services.v1.OscalServiceOuterClass.DeleteSspResponse> getDeleteSspMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.DeleteSspRequest, oscal.services.v1.OscalServiceOuterClass.DeleteSspResponse> getDeleteSspMethod;
    if ((getDeleteSspMethod = OscalServiceGrpc.getDeleteSspMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getDeleteSspMethod = OscalServiceGrpc.getDeleteSspMethod) == null) {
          OscalServiceGrpc.getDeleteSspMethod = getDeleteSspMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.DeleteSspRequest, oscal.services.v1.OscalServiceOuterClass.DeleteSspResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "DeleteSsp"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.DeleteSspRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.DeleteSspResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("DeleteSsp"))
              .build();
        }
      }
    }
    return getDeleteSspMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.GetAssessmentPlanRequest,
      oscal.services.v1.OscalServiceOuterClass.GetAssessmentPlanResponse> getGetAssessmentPlanMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GetAssessmentPlan",
      requestType = oscal.services.v1.OscalServiceOuterClass.GetAssessmentPlanRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.GetAssessmentPlanResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.GetAssessmentPlanRequest,
      oscal.services.v1.OscalServiceOuterClass.GetAssessmentPlanResponse> getGetAssessmentPlanMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.GetAssessmentPlanRequest, oscal.services.v1.OscalServiceOuterClass.GetAssessmentPlanResponse> getGetAssessmentPlanMethod;
    if ((getGetAssessmentPlanMethod = OscalServiceGrpc.getGetAssessmentPlanMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getGetAssessmentPlanMethod = OscalServiceGrpc.getGetAssessmentPlanMethod) == null) {
          OscalServiceGrpc.getGetAssessmentPlanMethod = getGetAssessmentPlanMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.GetAssessmentPlanRequest, oscal.services.v1.OscalServiceOuterClass.GetAssessmentPlanResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GetAssessmentPlan"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.GetAssessmentPlanRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.GetAssessmentPlanResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("GetAssessmentPlan"))
              .build();
        }
      }
    }
    return getGetAssessmentPlanMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.ListAssessmentPlansRequest,
      oscal.services.v1.OscalServiceOuterClass.ListAssessmentPlansResponse> getListAssessmentPlansMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "ListAssessmentPlans",
      requestType = oscal.services.v1.OscalServiceOuterClass.ListAssessmentPlansRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.ListAssessmentPlansResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.ListAssessmentPlansRequest,
      oscal.services.v1.OscalServiceOuterClass.ListAssessmentPlansResponse> getListAssessmentPlansMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.ListAssessmentPlansRequest, oscal.services.v1.OscalServiceOuterClass.ListAssessmentPlansResponse> getListAssessmentPlansMethod;
    if ((getListAssessmentPlansMethod = OscalServiceGrpc.getListAssessmentPlansMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getListAssessmentPlansMethod = OscalServiceGrpc.getListAssessmentPlansMethod) == null) {
          OscalServiceGrpc.getListAssessmentPlansMethod = getListAssessmentPlansMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.ListAssessmentPlansRequest, oscal.services.v1.OscalServiceOuterClass.ListAssessmentPlansResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "ListAssessmentPlans"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.ListAssessmentPlansRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.ListAssessmentPlansResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("ListAssessmentPlans"))
              .build();
        }
      }
    }
    return getListAssessmentPlansMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.CreateAssessmentPlanRequest,
      oscal.services.v1.OscalServiceOuterClass.CreateAssessmentPlanResponse> getCreateAssessmentPlanMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "CreateAssessmentPlan",
      requestType = oscal.services.v1.OscalServiceOuterClass.CreateAssessmentPlanRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.CreateAssessmentPlanResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.CreateAssessmentPlanRequest,
      oscal.services.v1.OscalServiceOuterClass.CreateAssessmentPlanResponse> getCreateAssessmentPlanMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.CreateAssessmentPlanRequest, oscal.services.v1.OscalServiceOuterClass.CreateAssessmentPlanResponse> getCreateAssessmentPlanMethod;
    if ((getCreateAssessmentPlanMethod = OscalServiceGrpc.getCreateAssessmentPlanMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getCreateAssessmentPlanMethod = OscalServiceGrpc.getCreateAssessmentPlanMethod) == null) {
          OscalServiceGrpc.getCreateAssessmentPlanMethod = getCreateAssessmentPlanMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.CreateAssessmentPlanRequest, oscal.services.v1.OscalServiceOuterClass.CreateAssessmentPlanResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "CreateAssessmentPlan"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.CreateAssessmentPlanRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.CreateAssessmentPlanResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("CreateAssessmentPlan"))
              .build();
        }
      }
    }
    return getCreateAssessmentPlanMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentPlanRequest,
      oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentPlanResponse> getUpdateAssessmentPlanMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "UpdateAssessmentPlan",
      requestType = oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentPlanRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentPlanResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentPlanRequest,
      oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentPlanResponse> getUpdateAssessmentPlanMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentPlanRequest, oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentPlanResponse> getUpdateAssessmentPlanMethod;
    if ((getUpdateAssessmentPlanMethod = OscalServiceGrpc.getUpdateAssessmentPlanMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getUpdateAssessmentPlanMethod = OscalServiceGrpc.getUpdateAssessmentPlanMethod) == null) {
          OscalServiceGrpc.getUpdateAssessmentPlanMethod = getUpdateAssessmentPlanMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentPlanRequest, oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentPlanResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "UpdateAssessmentPlan"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentPlanRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentPlanResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("UpdateAssessmentPlan"))
              .build();
        }
      }
    }
    return getUpdateAssessmentPlanMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentPlanRequest,
      oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentPlanResponse> getDeleteAssessmentPlanMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "DeleteAssessmentPlan",
      requestType = oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentPlanRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentPlanResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentPlanRequest,
      oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentPlanResponse> getDeleteAssessmentPlanMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentPlanRequest, oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentPlanResponse> getDeleteAssessmentPlanMethod;
    if ((getDeleteAssessmentPlanMethod = OscalServiceGrpc.getDeleteAssessmentPlanMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getDeleteAssessmentPlanMethod = OscalServiceGrpc.getDeleteAssessmentPlanMethod) == null) {
          OscalServiceGrpc.getDeleteAssessmentPlanMethod = getDeleteAssessmentPlanMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentPlanRequest, oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentPlanResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "DeleteAssessmentPlan"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentPlanRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentPlanResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("DeleteAssessmentPlan"))
              .build();
        }
      }
    }
    return getDeleteAssessmentPlanMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.GetAssessmentResultsRequest,
      oscal.services.v1.OscalServiceOuterClass.GetAssessmentResultsResponse> getGetAssessmentResultsMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GetAssessmentResults",
      requestType = oscal.services.v1.OscalServiceOuterClass.GetAssessmentResultsRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.GetAssessmentResultsResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.GetAssessmentResultsRequest,
      oscal.services.v1.OscalServiceOuterClass.GetAssessmentResultsResponse> getGetAssessmentResultsMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.GetAssessmentResultsRequest, oscal.services.v1.OscalServiceOuterClass.GetAssessmentResultsResponse> getGetAssessmentResultsMethod;
    if ((getGetAssessmentResultsMethod = OscalServiceGrpc.getGetAssessmentResultsMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getGetAssessmentResultsMethod = OscalServiceGrpc.getGetAssessmentResultsMethod) == null) {
          OscalServiceGrpc.getGetAssessmentResultsMethod = getGetAssessmentResultsMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.GetAssessmentResultsRequest, oscal.services.v1.OscalServiceOuterClass.GetAssessmentResultsResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GetAssessmentResults"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.GetAssessmentResultsRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.GetAssessmentResultsResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("GetAssessmentResults"))
              .build();
        }
      }
    }
    return getGetAssessmentResultsMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.ListAssessmentResultsRequest,
      oscal.services.v1.OscalServiceOuterClass.ListAssessmentResultsResponse> getListAssessmentResultsMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "ListAssessmentResults",
      requestType = oscal.services.v1.OscalServiceOuterClass.ListAssessmentResultsRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.ListAssessmentResultsResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.ListAssessmentResultsRequest,
      oscal.services.v1.OscalServiceOuterClass.ListAssessmentResultsResponse> getListAssessmentResultsMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.ListAssessmentResultsRequest, oscal.services.v1.OscalServiceOuterClass.ListAssessmentResultsResponse> getListAssessmentResultsMethod;
    if ((getListAssessmentResultsMethod = OscalServiceGrpc.getListAssessmentResultsMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getListAssessmentResultsMethod = OscalServiceGrpc.getListAssessmentResultsMethod) == null) {
          OscalServiceGrpc.getListAssessmentResultsMethod = getListAssessmentResultsMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.ListAssessmentResultsRequest, oscal.services.v1.OscalServiceOuterClass.ListAssessmentResultsResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "ListAssessmentResults"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.ListAssessmentResultsRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.ListAssessmentResultsResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("ListAssessmentResults"))
              .build();
        }
      }
    }
    return getListAssessmentResultsMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.CreateAssessmentResultsRequest,
      oscal.services.v1.OscalServiceOuterClass.CreateAssessmentResultsResponse> getCreateAssessmentResultsMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "CreateAssessmentResults",
      requestType = oscal.services.v1.OscalServiceOuterClass.CreateAssessmentResultsRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.CreateAssessmentResultsResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.CreateAssessmentResultsRequest,
      oscal.services.v1.OscalServiceOuterClass.CreateAssessmentResultsResponse> getCreateAssessmentResultsMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.CreateAssessmentResultsRequest, oscal.services.v1.OscalServiceOuterClass.CreateAssessmentResultsResponse> getCreateAssessmentResultsMethod;
    if ((getCreateAssessmentResultsMethod = OscalServiceGrpc.getCreateAssessmentResultsMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getCreateAssessmentResultsMethod = OscalServiceGrpc.getCreateAssessmentResultsMethod) == null) {
          OscalServiceGrpc.getCreateAssessmentResultsMethod = getCreateAssessmentResultsMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.CreateAssessmentResultsRequest, oscal.services.v1.OscalServiceOuterClass.CreateAssessmentResultsResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "CreateAssessmentResults"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.CreateAssessmentResultsRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.CreateAssessmentResultsResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("CreateAssessmentResults"))
              .build();
        }
      }
    }
    return getCreateAssessmentResultsMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentResultsRequest,
      oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentResultsResponse> getUpdateAssessmentResultsMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "UpdateAssessmentResults",
      requestType = oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentResultsRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentResultsResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentResultsRequest,
      oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentResultsResponse> getUpdateAssessmentResultsMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentResultsRequest, oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentResultsResponse> getUpdateAssessmentResultsMethod;
    if ((getUpdateAssessmentResultsMethod = OscalServiceGrpc.getUpdateAssessmentResultsMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getUpdateAssessmentResultsMethod = OscalServiceGrpc.getUpdateAssessmentResultsMethod) == null) {
          OscalServiceGrpc.getUpdateAssessmentResultsMethod = getUpdateAssessmentResultsMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentResultsRequest, oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentResultsResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "UpdateAssessmentResults"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentResultsRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentResultsResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("UpdateAssessmentResults"))
              .build();
        }
      }
    }
    return getUpdateAssessmentResultsMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentResultsRequest,
      oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentResultsResponse> getDeleteAssessmentResultsMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "DeleteAssessmentResults",
      requestType = oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentResultsRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentResultsResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentResultsRequest,
      oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentResultsResponse> getDeleteAssessmentResultsMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentResultsRequest, oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentResultsResponse> getDeleteAssessmentResultsMethod;
    if ((getDeleteAssessmentResultsMethod = OscalServiceGrpc.getDeleteAssessmentResultsMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getDeleteAssessmentResultsMethod = OscalServiceGrpc.getDeleteAssessmentResultsMethod) == null) {
          OscalServiceGrpc.getDeleteAssessmentResultsMethod = getDeleteAssessmentResultsMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentResultsRequest, oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentResultsResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "DeleteAssessmentResults"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentResultsRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentResultsResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("DeleteAssessmentResults"))
              .build();
        }
      }
    }
    return getDeleteAssessmentResultsMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.GetPoamRequest,
      oscal.services.v1.OscalServiceOuterClass.GetPoamResponse> getGetPoamMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GetPoam",
      requestType = oscal.services.v1.OscalServiceOuterClass.GetPoamRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.GetPoamResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.GetPoamRequest,
      oscal.services.v1.OscalServiceOuterClass.GetPoamResponse> getGetPoamMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.GetPoamRequest, oscal.services.v1.OscalServiceOuterClass.GetPoamResponse> getGetPoamMethod;
    if ((getGetPoamMethod = OscalServiceGrpc.getGetPoamMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getGetPoamMethod = OscalServiceGrpc.getGetPoamMethod) == null) {
          OscalServiceGrpc.getGetPoamMethod = getGetPoamMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.GetPoamRequest, oscal.services.v1.OscalServiceOuterClass.GetPoamResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GetPoam"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.GetPoamRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.GetPoamResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("GetPoam"))
              .build();
        }
      }
    }
    return getGetPoamMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.ListPoamsRequest,
      oscal.services.v1.OscalServiceOuterClass.ListPoamsResponse> getListPoamsMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "ListPoams",
      requestType = oscal.services.v1.OscalServiceOuterClass.ListPoamsRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.ListPoamsResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.ListPoamsRequest,
      oscal.services.v1.OscalServiceOuterClass.ListPoamsResponse> getListPoamsMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.ListPoamsRequest, oscal.services.v1.OscalServiceOuterClass.ListPoamsResponse> getListPoamsMethod;
    if ((getListPoamsMethod = OscalServiceGrpc.getListPoamsMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getListPoamsMethod = OscalServiceGrpc.getListPoamsMethod) == null) {
          OscalServiceGrpc.getListPoamsMethod = getListPoamsMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.ListPoamsRequest, oscal.services.v1.OscalServiceOuterClass.ListPoamsResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "ListPoams"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.ListPoamsRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.ListPoamsResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("ListPoams"))
              .build();
        }
      }
    }
    return getListPoamsMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.CreatePoamRequest,
      oscal.services.v1.OscalServiceOuterClass.CreatePoamResponse> getCreatePoamMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "CreatePoam",
      requestType = oscal.services.v1.OscalServiceOuterClass.CreatePoamRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.CreatePoamResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.CreatePoamRequest,
      oscal.services.v1.OscalServiceOuterClass.CreatePoamResponse> getCreatePoamMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.CreatePoamRequest, oscal.services.v1.OscalServiceOuterClass.CreatePoamResponse> getCreatePoamMethod;
    if ((getCreatePoamMethod = OscalServiceGrpc.getCreatePoamMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getCreatePoamMethod = OscalServiceGrpc.getCreatePoamMethod) == null) {
          OscalServiceGrpc.getCreatePoamMethod = getCreatePoamMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.CreatePoamRequest, oscal.services.v1.OscalServiceOuterClass.CreatePoamResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "CreatePoam"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.CreatePoamRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.CreatePoamResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("CreatePoam"))
              .build();
        }
      }
    }
    return getCreatePoamMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.UpdatePoamRequest,
      oscal.services.v1.OscalServiceOuterClass.UpdatePoamResponse> getUpdatePoamMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "UpdatePoam",
      requestType = oscal.services.v1.OscalServiceOuterClass.UpdatePoamRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.UpdatePoamResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.UpdatePoamRequest,
      oscal.services.v1.OscalServiceOuterClass.UpdatePoamResponse> getUpdatePoamMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.UpdatePoamRequest, oscal.services.v1.OscalServiceOuterClass.UpdatePoamResponse> getUpdatePoamMethod;
    if ((getUpdatePoamMethod = OscalServiceGrpc.getUpdatePoamMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getUpdatePoamMethod = OscalServiceGrpc.getUpdatePoamMethod) == null) {
          OscalServiceGrpc.getUpdatePoamMethod = getUpdatePoamMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.UpdatePoamRequest, oscal.services.v1.OscalServiceOuterClass.UpdatePoamResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "UpdatePoam"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.UpdatePoamRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.UpdatePoamResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("UpdatePoam"))
              .build();
        }
      }
    }
    return getUpdatePoamMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.DeletePoamRequest,
      oscal.services.v1.OscalServiceOuterClass.DeletePoamResponse> getDeletePoamMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "DeletePoam",
      requestType = oscal.services.v1.OscalServiceOuterClass.DeletePoamRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.DeletePoamResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.DeletePoamRequest,
      oscal.services.v1.OscalServiceOuterClass.DeletePoamResponse> getDeletePoamMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.DeletePoamRequest, oscal.services.v1.OscalServiceOuterClass.DeletePoamResponse> getDeletePoamMethod;
    if ((getDeletePoamMethod = OscalServiceGrpc.getDeletePoamMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getDeletePoamMethod = OscalServiceGrpc.getDeletePoamMethod) == null) {
          OscalServiceGrpc.getDeletePoamMethod = getDeletePoamMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.DeletePoamRequest, oscal.services.v1.OscalServiceOuterClass.DeletePoamResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "DeletePoam"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.DeletePoamRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.DeletePoamResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("DeletePoam"))
              .build();
        }
      }
    }
    return getDeletePoamMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.GetMappingRequest,
      oscal.services.v1.OscalServiceOuterClass.GetMappingResponse> getGetMappingMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GetMapping",
      requestType = oscal.services.v1.OscalServiceOuterClass.GetMappingRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.GetMappingResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.GetMappingRequest,
      oscal.services.v1.OscalServiceOuterClass.GetMappingResponse> getGetMappingMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.GetMappingRequest, oscal.services.v1.OscalServiceOuterClass.GetMappingResponse> getGetMappingMethod;
    if ((getGetMappingMethod = OscalServiceGrpc.getGetMappingMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getGetMappingMethod = OscalServiceGrpc.getGetMappingMethod) == null) {
          OscalServiceGrpc.getGetMappingMethod = getGetMappingMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.GetMappingRequest, oscal.services.v1.OscalServiceOuterClass.GetMappingResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GetMapping"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.GetMappingRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.GetMappingResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("GetMapping"))
              .build();
        }
      }
    }
    return getGetMappingMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.ListMappingsRequest,
      oscal.services.v1.OscalServiceOuterClass.ListMappingsResponse> getListMappingsMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "ListMappings",
      requestType = oscal.services.v1.OscalServiceOuterClass.ListMappingsRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.ListMappingsResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.ListMappingsRequest,
      oscal.services.v1.OscalServiceOuterClass.ListMappingsResponse> getListMappingsMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.ListMappingsRequest, oscal.services.v1.OscalServiceOuterClass.ListMappingsResponse> getListMappingsMethod;
    if ((getListMappingsMethod = OscalServiceGrpc.getListMappingsMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getListMappingsMethod = OscalServiceGrpc.getListMappingsMethod) == null) {
          OscalServiceGrpc.getListMappingsMethod = getListMappingsMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.ListMappingsRequest, oscal.services.v1.OscalServiceOuterClass.ListMappingsResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "ListMappings"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.ListMappingsRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.ListMappingsResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("ListMappings"))
              .build();
        }
      }
    }
    return getListMappingsMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.CreateMappingRequest,
      oscal.services.v1.OscalServiceOuterClass.CreateMappingResponse> getCreateMappingMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "CreateMapping",
      requestType = oscal.services.v1.OscalServiceOuterClass.CreateMappingRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.CreateMappingResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.CreateMappingRequest,
      oscal.services.v1.OscalServiceOuterClass.CreateMappingResponse> getCreateMappingMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.CreateMappingRequest, oscal.services.v1.OscalServiceOuterClass.CreateMappingResponse> getCreateMappingMethod;
    if ((getCreateMappingMethod = OscalServiceGrpc.getCreateMappingMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getCreateMappingMethod = OscalServiceGrpc.getCreateMappingMethod) == null) {
          OscalServiceGrpc.getCreateMappingMethod = getCreateMappingMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.CreateMappingRequest, oscal.services.v1.OscalServiceOuterClass.CreateMappingResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "CreateMapping"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.CreateMappingRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.CreateMappingResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("CreateMapping"))
              .build();
        }
      }
    }
    return getCreateMappingMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.UpdateMappingRequest,
      oscal.services.v1.OscalServiceOuterClass.UpdateMappingResponse> getUpdateMappingMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "UpdateMapping",
      requestType = oscal.services.v1.OscalServiceOuterClass.UpdateMappingRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.UpdateMappingResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.UpdateMappingRequest,
      oscal.services.v1.OscalServiceOuterClass.UpdateMappingResponse> getUpdateMappingMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.UpdateMappingRequest, oscal.services.v1.OscalServiceOuterClass.UpdateMappingResponse> getUpdateMappingMethod;
    if ((getUpdateMappingMethod = OscalServiceGrpc.getUpdateMappingMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getUpdateMappingMethod = OscalServiceGrpc.getUpdateMappingMethod) == null) {
          OscalServiceGrpc.getUpdateMappingMethod = getUpdateMappingMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.UpdateMappingRequest, oscal.services.v1.OscalServiceOuterClass.UpdateMappingResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "UpdateMapping"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.UpdateMappingRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.UpdateMappingResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("UpdateMapping"))
              .build();
        }
      }
    }
    return getUpdateMappingMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.DeleteMappingRequest,
      oscal.services.v1.OscalServiceOuterClass.DeleteMappingResponse> getDeleteMappingMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "DeleteMapping",
      requestType = oscal.services.v1.OscalServiceOuterClass.DeleteMappingRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.DeleteMappingResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.DeleteMappingRequest,
      oscal.services.v1.OscalServiceOuterClass.DeleteMappingResponse> getDeleteMappingMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.DeleteMappingRequest, oscal.services.v1.OscalServiceOuterClass.DeleteMappingResponse> getDeleteMappingMethod;
    if ((getDeleteMappingMethod = OscalServiceGrpc.getDeleteMappingMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getDeleteMappingMethod = OscalServiceGrpc.getDeleteMappingMethod) == null) {
          OscalServiceGrpc.getDeleteMappingMethod = getDeleteMappingMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.DeleteMappingRequest, oscal.services.v1.OscalServiceOuterClass.DeleteMappingResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "DeleteMapping"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.DeleteMappingRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.DeleteMappingResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("DeleteMapping"))
              .build();
        }
      }
    }
    return getDeleteMappingMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.SearchRequest,
      oscal.services.v1.OscalServiceOuterClass.SearchResponse> getSearchMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "Search",
      requestType = oscal.services.v1.OscalServiceOuterClass.SearchRequest.class,
      responseType = oscal.services.v1.OscalServiceOuterClass.SearchResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.SearchRequest,
      oscal.services.v1.OscalServiceOuterClass.SearchResponse> getSearchMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.OscalServiceOuterClass.SearchRequest, oscal.services.v1.OscalServiceOuterClass.SearchResponse> getSearchMethod;
    if ((getSearchMethod = OscalServiceGrpc.getSearchMethod) == null) {
      synchronized (OscalServiceGrpc.class) {
        if ((getSearchMethod = OscalServiceGrpc.getSearchMethod) == null) {
          OscalServiceGrpc.getSearchMethod = getSearchMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.OscalServiceOuterClass.SearchRequest, oscal.services.v1.OscalServiceOuterClass.SearchResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "Search"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.SearchRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.OscalServiceOuterClass.SearchResponse.getDefaultInstance()))
              .setSchemaDescriptor(new OscalServiceMethodDescriptorSupplier("Search"))
              .build();
        }
      }
    }
    return getSearchMethod;
  }

  /**
   * Creates a new async stub that supports all call types for the service
   */
  public static OscalServiceStub newStub(io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<OscalServiceStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<OscalServiceStub>() {
        @java.lang.Override
        public OscalServiceStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new OscalServiceStub(channel, callOptions);
        }
      };
    return OscalServiceStub.newStub(factory, channel);
  }

  /**
   * Creates a new blocking-style stub that supports all types of calls on the service
   */
  public static OscalServiceBlockingV2Stub newBlockingV2Stub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<OscalServiceBlockingV2Stub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<OscalServiceBlockingV2Stub>() {
        @java.lang.Override
        public OscalServiceBlockingV2Stub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new OscalServiceBlockingV2Stub(channel, callOptions);
        }
      };
    return OscalServiceBlockingV2Stub.newStub(factory, channel);
  }

  /**
   * Creates a new blocking-style stub that supports unary and streaming output calls on the service
   */
  public static OscalServiceBlockingStub newBlockingStub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<OscalServiceBlockingStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<OscalServiceBlockingStub>() {
        @java.lang.Override
        public OscalServiceBlockingStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new OscalServiceBlockingStub(channel, callOptions);
        }
      };
    return OscalServiceBlockingStub.newStub(factory, channel);
  }

  /**
   * Creates a new ListenableFuture-style stub that supports unary calls on the service
   */
  public static OscalServiceFutureStub newFutureStub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<OscalServiceFutureStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<OscalServiceFutureStub>() {
        @java.lang.Override
        public OscalServiceFutureStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new OscalServiceFutureStub(channel, callOptions);
        }
      };
    return OscalServiceFutureStub.newStub(factory, channel);
  }

  /**
   * <pre>
   * OSCAL Service provides CRUD operations for all OSCAL models
   * </pre>
   */
  public interface AsyncService {

    /**
     * <pre>
     * Catalog operations
     * </pre>
     */
    default void getCatalog(oscal.services.v1.OscalServiceOuterClass.GetCatalogRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.GetCatalogResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getGetCatalogMethod(), responseObserver);
    }

    /**
     */
    default void listCatalogs(oscal.services.v1.OscalServiceOuterClass.ListCatalogsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.ListCatalogsResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getListCatalogsMethod(), responseObserver);
    }

    /**
     */
    default void createCatalog(oscal.services.v1.OscalServiceOuterClass.CreateCatalogRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.CreateCatalogResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getCreateCatalogMethod(), responseObserver);
    }

    /**
     */
    default void updateCatalog(oscal.services.v1.OscalServiceOuterClass.UpdateCatalogRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.UpdateCatalogResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getUpdateCatalogMethod(), responseObserver);
    }

    /**
     */
    default void deleteCatalog(oscal.services.v1.OscalServiceOuterClass.DeleteCatalogRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.DeleteCatalogResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getDeleteCatalogMethod(), responseObserver);
    }

    /**
     * <pre>
     * Profile operations
     * </pre>
     */
    default void getProfile(oscal.services.v1.OscalServiceOuterClass.GetProfileRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.GetProfileResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getGetProfileMethod(), responseObserver);
    }

    /**
     */
    default void listProfiles(oscal.services.v1.OscalServiceOuterClass.ListProfilesRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.ListProfilesResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getListProfilesMethod(), responseObserver);
    }

    /**
     */
    default void createProfile(oscal.services.v1.OscalServiceOuterClass.CreateProfileRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.CreateProfileResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getCreateProfileMethod(), responseObserver);
    }

    /**
     */
    default void updateProfile(oscal.services.v1.OscalServiceOuterClass.UpdateProfileRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.UpdateProfileResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getUpdateProfileMethod(), responseObserver);
    }

    /**
     */
    default void deleteProfile(oscal.services.v1.OscalServiceOuterClass.DeleteProfileRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.DeleteProfileResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getDeleteProfileMethod(), responseObserver);
    }

    /**
     * <pre>
     * Component Definition operations
     * </pre>
     */
    default void getComponentDefinition(oscal.services.v1.OscalServiceOuterClass.GetComponentDefinitionRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.GetComponentDefinitionResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getGetComponentDefinitionMethod(), responseObserver);
    }

    /**
     */
    default void listComponentDefinitions(oscal.services.v1.OscalServiceOuterClass.ListComponentDefinitionsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.ListComponentDefinitionsResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getListComponentDefinitionsMethod(), responseObserver);
    }

    /**
     */
    default void createComponentDefinition(oscal.services.v1.OscalServiceOuterClass.CreateComponentDefinitionRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.CreateComponentDefinitionResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getCreateComponentDefinitionMethod(), responseObserver);
    }

    /**
     */
    default void updateComponentDefinition(oscal.services.v1.OscalServiceOuterClass.UpdateComponentDefinitionRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.UpdateComponentDefinitionResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getUpdateComponentDefinitionMethod(), responseObserver);
    }

    /**
     */
    default void deleteComponentDefinition(oscal.services.v1.OscalServiceOuterClass.DeleteComponentDefinitionRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.DeleteComponentDefinitionResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getDeleteComponentDefinitionMethod(), responseObserver);
    }

    /**
     * <pre>
     * SSP operations
     * </pre>
     */
    default void getSsp(oscal.services.v1.OscalServiceOuterClass.GetSspRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.GetSspResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getGetSspMethod(), responseObserver);
    }

    /**
     */
    default void listSsps(oscal.services.v1.OscalServiceOuterClass.ListSspsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.ListSspsResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getListSspsMethod(), responseObserver);
    }

    /**
     */
    default void createSsp(oscal.services.v1.OscalServiceOuterClass.CreateSspRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.CreateSspResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getCreateSspMethod(), responseObserver);
    }

    /**
     */
    default void updateSsp(oscal.services.v1.OscalServiceOuterClass.UpdateSspRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.UpdateSspResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getUpdateSspMethod(), responseObserver);
    }

    /**
     */
    default void deleteSsp(oscal.services.v1.OscalServiceOuterClass.DeleteSspRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.DeleteSspResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getDeleteSspMethod(), responseObserver);
    }

    /**
     * <pre>
     * Assessment Plan operations
     * </pre>
     */
    default void getAssessmentPlan(oscal.services.v1.OscalServiceOuterClass.GetAssessmentPlanRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.GetAssessmentPlanResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getGetAssessmentPlanMethod(), responseObserver);
    }

    /**
     */
    default void listAssessmentPlans(oscal.services.v1.OscalServiceOuterClass.ListAssessmentPlansRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.ListAssessmentPlansResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getListAssessmentPlansMethod(), responseObserver);
    }

    /**
     */
    default void createAssessmentPlan(oscal.services.v1.OscalServiceOuterClass.CreateAssessmentPlanRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.CreateAssessmentPlanResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getCreateAssessmentPlanMethod(), responseObserver);
    }

    /**
     */
    default void updateAssessmentPlan(oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentPlanRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentPlanResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getUpdateAssessmentPlanMethod(), responseObserver);
    }

    /**
     */
    default void deleteAssessmentPlan(oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentPlanRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentPlanResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getDeleteAssessmentPlanMethod(), responseObserver);
    }

    /**
     * <pre>
     * Assessment Results operations
     * </pre>
     */
    default void getAssessmentResults(oscal.services.v1.OscalServiceOuterClass.GetAssessmentResultsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.GetAssessmentResultsResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getGetAssessmentResultsMethod(), responseObserver);
    }

    /**
     */
    default void listAssessmentResults(oscal.services.v1.OscalServiceOuterClass.ListAssessmentResultsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.ListAssessmentResultsResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getListAssessmentResultsMethod(), responseObserver);
    }

    /**
     */
    default void createAssessmentResults(oscal.services.v1.OscalServiceOuterClass.CreateAssessmentResultsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.CreateAssessmentResultsResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getCreateAssessmentResultsMethod(), responseObserver);
    }

    /**
     */
    default void updateAssessmentResults(oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentResultsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentResultsResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getUpdateAssessmentResultsMethod(), responseObserver);
    }

    /**
     */
    default void deleteAssessmentResults(oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentResultsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentResultsResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getDeleteAssessmentResultsMethod(), responseObserver);
    }

    /**
     * <pre>
     * POA&amp;M operations
     * </pre>
     */
    default void getPoam(oscal.services.v1.OscalServiceOuterClass.GetPoamRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.GetPoamResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getGetPoamMethod(), responseObserver);
    }

    /**
     */
    default void listPoams(oscal.services.v1.OscalServiceOuterClass.ListPoamsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.ListPoamsResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getListPoamsMethod(), responseObserver);
    }

    /**
     */
    default void createPoam(oscal.services.v1.OscalServiceOuterClass.CreatePoamRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.CreatePoamResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getCreatePoamMethod(), responseObserver);
    }

    /**
     */
    default void updatePoam(oscal.services.v1.OscalServiceOuterClass.UpdatePoamRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.UpdatePoamResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getUpdatePoamMethod(), responseObserver);
    }

    /**
     */
    default void deletePoam(oscal.services.v1.OscalServiceOuterClass.DeletePoamRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.DeletePoamResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getDeletePoamMethod(), responseObserver);
    }

    /**
     * <pre>
     * Mapping operations
     * </pre>
     */
    default void getMapping(oscal.services.v1.OscalServiceOuterClass.GetMappingRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.GetMappingResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getGetMappingMethod(), responseObserver);
    }

    /**
     */
    default void listMappings(oscal.services.v1.OscalServiceOuterClass.ListMappingsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.ListMappingsResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getListMappingsMethod(), responseObserver);
    }

    /**
     */
    default void createMapping(oscal.services.v1.OscalServiceOuterClass.CreateMappingRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.CreateMappingResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getCreateMappingMethod(), responseObserver);
    }

    /**
     */
    default void updateMapping(oscal.services.v1.OscalServiceOuterClass.UpdateMappingRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.UpdateMappingResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getUpdateMappingMethod(), responseObserver);
    }

    /**
     */
    default void deleteMapping(oscal.services.v1.OscalServiceOuterClass.DeleteMappingRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.DeleteMappingResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getDeleteMappingMethod(), responseObserver);
    }

    /**
     * <pre>
     * Search operations
     * </pre>
     */
    default void search(oscal.services.v1.OscalServiceOuterClass.SearchRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.SearchResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getSearchMethod(), responseObserver);
    }
  }

  /**
   * Base class for the server implementation of the service OscalService.
   * <pre>
   * OSCAL Service provides CRUD operations for all OSCAL models
   * </pre>
   */
  public static abstract class OscalServiceImplBase
      implements io.grpc.BindableService, AsyncService {

    @java.lang.Override public final io.grpc.ServerServiceDefinition bindService() {
      return OscalServiceGrpc.bindService(this);
    }
  }

  /**
   * A stub to allow clients to do asynchronous rpc calls to service OscalService.
   * <pre>
   * OSCAL Service provides CRUD operations for all OSCAL models
   * </pre>
   */
  public static final class OscalServiceStub
      extends io.grpc.stub.AbstractAsyncStub<OscalServiceStub> {
    private OscalServiceStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected OscalServiceStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new OscalServiceStub(channel, callOptions);
    }

    /**
     * <pre>
     * Catalog operations
     * </pre>
     */
    public void getCatalog(oscal.services.v1.OscalServiceOuterClass.GetCatalogRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.GetCatalogResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getGetCatalogMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void listCatalogs(oscal.services.v1.OscalServiceOuterClass.ListCatalogsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.ListCatalogsResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getListCatalogsMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void createCatalog(oscal.services.v1.OscalServiceOuterClass.CreateCatalogRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.CreateCatalogResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getCreateCatalogMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void updateCatalog(oscal.services.v1.OscalServiceOuterClass.UpdateCatalogRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.UpdateCatalogResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getUpdateCatalogMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void deleteCatalog(oscal.services.v1.OscalServiceOuterClass.DeleteCatalogRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.DeleteCatalogResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getDeleteCatalogMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * Profile operations
     * </pre>
     */
    public void getProfile(oscal.services.v1.OscalServiceOuterClass.GetProfileRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.GetProfileResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getGetProfileMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void listProfiles(oscal.services.v1.OscalServiceOuterClass.ListProfilesRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.ListProfilesResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getListProfilesMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void createProfile(oscal.services.v1.OscalServiceOuterClass.CreateProfileRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.CreateProfileResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getCreateProfileMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void updateProfile(oscal.services.v1.OscalServiceOuterClass.UpdateProfileRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.UpdateProfileResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getUpdateProfileMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void deleteProfile(oscal.services.v1.OscalServiceOuterClass.DeleteProfileRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.DeleteProfileResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getDeleteProfileMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * Component Definition operations
     * </pre>
     */
    public void getComponentDefinition(oscal.services.v1.OscalServiceOuterClass.GetComponentDefinitionRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.GetComponentDefinitionResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getGetComponentDefinitionMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void listComponentDefinitions(oscal.services.v1.OscalServiceOuterClass.ListComponentDefinitionsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.ListComponentDefinitionsResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getListComponentDefinitionsMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void createComponentDefinition(oscal.services.v1.OscalServiceOuterClass.CreateComponentDefinitionRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.CreateComponentDefinitionResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getCreateComponentDefinitionMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void updateComponentDefinition(oscal.services.v1.OscalServiceOuterClass.UpdateComponentDefinitionRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.UpdateComponentDefinitionResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getUpdateComponentDefinitionMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void deleteComponentDefinition(oscal.services.v1.OscalServiceOuterClass.DeleteComponentDefinitionRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.DeleteComponentDefinitionResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getDeleteComponentDefinitionMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * SSP operations
     * </pre>
     */
    public void getSsp(oscal.services.v1.OscalServiceOuterClass.GetSspRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.GetSspResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getGetSspMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void listSsps(oscal.services.v1.OscalServiceOuterClass.ListSspsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.ListSspsResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getListSspsMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void createSsp(oscal.services.v1.OscalServiceOuterClass.CreateSspRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.CreateSspResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getCreateSspMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void updateSsp(oscal.services.v1.OscalServiceOuterClass.UpdateSspRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.UpdateSspResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getUpdateSspMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void deleteSsp(oscal.services.v1.OscalServiceOuterClass.DeleteSspRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.DeleteSspResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getDeleteSspMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * Assessment Plan operations
     * </pre>
     */
    public void getAssessmentPlan(oscal.services.v1.OscalServiceOuterClass.GetAssessmentPlanRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.GetAssessmentPlanResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getGetAssessmentPlanMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void listAssessmentPlans(oscal.services.v1.OscalServiceOuterClass.ListAssessmentPlansRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.ListAssessmentPlansResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getListAssessmentPlansMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void createAssessmentPlan(oscal.services.v1.OscalServiceOuterClass.CreateAssessmentPlanRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.CreateAssessmentPlanResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getCreateAssessmentPlanMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void updateAssessmentPlan(oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentPlanRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentPlanResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getUpdateAssessmentPlanMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void deleteAssessmentPlan(oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentPlanRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentPlanResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getDeleteAssessmentPlanMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * Assessment Results operations
     * </pre>
     */
    public void getAssessmentResults(oscal.services.v1.OscalServiceOuterClass.GetAssessmentResultsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.GetAssessmentResultsResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getGetAssessmentResultsMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void listAssessmentResults(oscal.services.v1.OscalServiceOuterClass.ListAssessmentResultsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.ListAssessmentResultsResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getListAssessmentResultsMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void createAssessmentResults(oscal.services.v1.OscalServiceOuterClass.CreateAssessmentResultsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.CreateAssessmentResultsResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getCreateAssessmentResultsMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void updateAssessmentResults(oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentResultsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentResultsResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getUpdateAssessmentResultsMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void deleteAssessmentResults(oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentResultsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentResultsResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getDeleteAssessmentResultsMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * POA&amp;M operations
     * </pre>
     */
    public void getPoam(oscal.services.v1.OscalServiceOuterClass.GetPoamRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.GetPoamResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getGetPoamMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void listPoams(oscal.services.v1.OscalServiceOuterClass.ListPoamsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.ListPoamsResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getListPoamsMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void createPoam(oscal.services.v1.OscalServiceOuterClass.CreatePoamRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.CreatePoamResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getCreatePoamMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void updatePoam(oscal.services.v1.OscalServiceOuterClass.UpdatePoamRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.UpdatePoamResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getUpdatePoamMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void deletePoam(oscal.services.v1.OscalServiceOuterClass.DeletePoamRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.DeletePoamResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getDeletePoamMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * Mapping operations
     * </pre>
     */
    public void getMapping(oscal.services.v1.OscalServiceOuterClass.GetMappingRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.GetMappingResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getGetMappingMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void listMappings(oscal.services.v1.OscalServiceOuterClass.ListMappingsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.ListMappingsResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getListMappingsMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void createMapping(oscal.services.v1.OscalServiceOuterClass.CreateMappingRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.CreateMappingResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getCreateMappingMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void updateMapping(oscal.services.v1.OscalServiceOuterClass.UpdateMappingRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.UpdateMappingResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getUpdateMappingMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void deleteMapping(oscal.services.v1.OscalServiceOuterClass.DeleteMappingRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.DeleteMappingResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getDeleteMappingMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * Search operations
     * </pre>
     */
    public void search(oscal.services.v1.OscalServiceOuterClass.SearchRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.SearchResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getSearchMethod(), getCallOptions()), request, responseObserver);
    }
  }

  /**
   * A stub to allow clients to do synchronous rpc calls to service OscalService.
   * <pre>
   * OSCAL Service provides CRUD operations for all OSCAL models
   * </pre>
   */
  public static final class OscalServiceBlockingV2Stub
      extends io.grpc.stub.AbstractBlockingStub<OscalServiceBlockingV2Stub> {
    private OscalServiceBlockingV2Stub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected OscalServiceBlockingV2Stub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new OscalServiceBlockingV2Stub(channel, callOptions);
    }

    /**
     * <pre>
     * Catalog operations
     * </pre>
     */
    public oscal.services.v1.OscalServiceOuterClass.GetCatalogResponse getCatalog(oscal.services.v1.OscalServiceOuterClass.GetCatalogRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getGetCatalogMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.ListCatalogsResponse listCatalogs(oscal.services.v1.OscalServiceOuterClass.ListCatalogsRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getListCatalogsMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.CreateCatalogResponse createCatalog(oscal.services.v1.OscalServiceOuterClass.CreateCatalogRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getCreateCatalogMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.UpdateCatalogResponse updateCatalog(oscal.services.v1.OscalServiceOuterClass.UpdateCatalogRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getUpdateCatalogMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.DeleteCatalogResponse deleteCatalog(oscal.services.v1.OscalServiceOuterClass.DeleteCatalogRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getDeleteCatalogMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * Profile operations
     * </pre>
     */
    public oscal.services.v1.OscalServiceOuterClass.GetProfileResponse getProfile(oscal.services.v1.OscalServiceOuterClass.GetProfileRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getGetProfileMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.ListProfilesResponse listProfiles(oscal.services.v1.OscalServiceOuterClass.ListProfilesRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getListProfilesMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.CreateProfileResponse createProfile(oscal.services.v1.OscalServiceOuterClass.CreateProfileRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getCreateProfileMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.UpdateProfileResponse updateProfile(oscal.services.v1.OscalServiceOuterClass.UpdateProfileRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getUpdateProfileMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.DeleteProfileResponse deleteProfile(oscal.services.v1.OscalServiceOuterClass.DeleteProfileRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getDeleteProfileMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * Component Definition operations
     * </pre>
     */
    public oscal.services.v1.OscalServiceOuterClass.GetComponentDefinitionResponse getComponentDefinition(oscal.services.v1.OscalServiceOuterClass.GetComponentDefinitionRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getGetComponentDefinitionMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.ListComponentDefinitionsResponse listComponentDefinitions(oscal.services.v1.OscalServiceOuterClass.ListComponentDefinitionsRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getListComponentDefinitionsMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.CreateComponentDefinitionResponse createComponentDefinition(oscal.services.v1.OscalServiceOuterClass.CreateComponentDefinitionRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getCreateComponentDefinitionMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.UpdateComponentDefinitionResponse updateComponentDefinition(oscal.services.v1.OscalServiceOuterClass.UpdateComponentDefinitionRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getUpdateComponentDefinitionMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.DeleteComponentDefinitionResponse deleteComponentDefinition(oscal.services.v1.OscalServiceOuterClass.DeleteComponentDefinitionRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getDeleteComponentDefinitionMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * SSP operations
     * </pre>
     */
    public oscal.services.v1.OscalServiceOuterClass.GetSspResponse getSsp(oscal.services.v1.OscalServiceOuterClass.GetSspRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getGetSspMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.ListSspsResponse listSsps(oscal.services.v1.OscalServiceOuterClass.ListSspsRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getListSspsMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.CreateSspResponse createSsp(oscal.services.v1.OscalServiceOuterClass.CreateSspRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getCreateSspMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.UpdateSspResponse updateSsp(oscal.services.v1.OscalServiceOuterClass.UpdateSspRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getUpdateSspMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.DeleteSspResponse deleteSsp(oscal.services.v1.OscalServiceOuterClass.DeleteSspRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getDeleteSspMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * Assessment Plan operations
     * </pre>
     */
    public oscal.services.v1.OscalServiceOuterClass.GetAssessmentPlanResponse getAssessmentPlan(oscal.services.v1.OscalServiceOuterClass.GetAssessmentPlanRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getGetAssessmentPlanMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.ListAssessmentPlansResponse listAssessmentPlans(oscal.services.v1.OscalServiceOuterClass.ListAssessmentPlansRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getListAssessmentPlansMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.CreateAssessmentPlanResponse createAssessmentPlan(oscal.services.v1.OscalServiceOuterClass.CreateAssessmentPlanRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getCreateAssessmentPlanMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentPlanResponse updateAssessmentPlan(oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentPlanRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getUpdateAssessmentPlanMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentPlanResponse deleteAssessmentPlan(oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentPlanRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getDeleteAssessmentPlanMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * Assessment Results operations
     * </pre>
     */
    public oscal.services.v1.OscalServiceOuterClass.GetAssessmentResultsResponse getAssessmentResults(oscal.services.v1.OscalServiceOuterClass.GetAssessmentResultsRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getGetAssessmentResultsMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.ListAssessmentResultsResponse listAssessmentResults(oscal.services.v1.OscalServiceOuterClass.ListAssessmentResultsRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getListAssessmentResultsMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.CreateAssessmentResultsResponse createAssessmentResults(oscal.services.v1.OscalServiceOuterClass.CreateAssessmentResultsRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getCreateAssessmentResultsMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentResultsResponse updateAssessmentResults(oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentResultsRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getUpdateAssessmentResultsMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentResultsResponse deleteAssessmentResults(oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentResultsRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getDeleteAssessmentResultsMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * POA&amp;M operations
     * </pre>
     */
    public oscal.services.v1.OscalServiceOuterClass.GetPoamResponse getPoam(oscal.services.v1.OscalServiceOuterClass.GetPoamRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getGetPoamMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.ListPoamsResponse listPoams(oscal.services.v1.OscalServiceOuterClass.ListPoamsRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getListPoamsMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.CreatePoamResponse createPoam(oscal.services.v1.OscalServiceOuterClass.CreatePoamRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getCreatePoamMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.UpdatePoamResponse updatePoam(oscal.services.v1.OscalServiceOuterClass.UpdatePoamRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getUpdatePoamMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.DeletePoamResponse deletePoam(oscal.services.v1.OscalServiceOuterClass.DeletePoamRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getDeletePoamMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * Mapping operations
     * </pre>
     */
    public oscal.services.v1.OscalServiceOuterClass.GetMappingResponse getMapping(oscal.services.v1.OscalServiceOuterClass.GetMappingRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getGetMappingMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.ListMappingsResponse listMappings(oscal.services.v1.OscalServiceOuterClass.ListMappingsRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getListMappingsMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.CreateMappingResponse createMapping(oscal.services.v1.OscalServiceOuterClass.CreateMappingRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getCreateMappingMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.UpdateMappingResponse updateMapping(oscal.services.v1.OscalServiceOuterClass.UpdateMappingRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getUpdateMappingMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.DeleteMappingResponse deleteMapping(oscal.services.v1.OscalServiceOuterClass.DeleteMappingRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getDeleteMappingMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * Search operations
     * </pre>
     */
    public oscal.services.v1.OscalServiceOuterClass.SearchResponse search(oscal.services.v1.OscalServiceOuterClass.SearchRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getSearchMethod(), getCallOptions(), request);
    }
  }

  /**
   * A stub to allow clients to do limited synchronous rpc calls to service OscalService.
   * <pre>
   * OSCAL Service provides CRUD operations for all OSCAL models
   * </pre>
   */
  public static final class OscalServiceBlockingStub
      extends io.grpc.stub.AbstractBlockingStub<OscalServiceBlockingStub> {
    private OscalServiceBlockingStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected OscalServiceBlockingStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new OscalServiceBlockingStub(channel, callOptions);
    }

    /**
     * <pre>
     * Catalog operations
     * </pre>
     */
    public oscal.services.v1.OscalServiceOuterClass.GetCatalogResponse getCatalog(oscal.services.v1.OscalServiceOuterClass.GetCatalogRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getGetCatalogMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.ListCatalogsResponse listCatalogs(oscal.services.v1.OscalServiceOuterClass.ListCatalogsRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getListCatalogsMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.CreateCatalogResponse createCatalog(oscal.services.v1.OscalServiceOuterClass.CreateCatalogRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getCreateCatalogMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.UpdateCatalogResponse updateCatalog(oscal.services.v1.OscalServiceOuterClass.UpdateCatalogRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getUpdateCatalogMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.DeleteCatalogResponse deleteCatalog(oscal.services.v1.OscalServiceOuterClass.DeleteCatalogRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getDeleteCatalogMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * Profile operations
     * </pre>
     */
    public oscal.services.v1.OscalServiceOuterClass.GetProfileResponse getProfile(oscal.services.v1.OscalServiceOuterClass.GetProfileRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getGetProfileMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.ListProfilesResponse listProfiles(oscal.services.v1.OscalServiceOuterClass.ListProfilesRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getListProfilesMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.CreateProfileResponse createProfile(oscal.services.v1.OscalServiceOuterClass.CreateProfileRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getCreateProfileMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.UpdateProfileResponse updateProfile(oscal.services.v1.OscalServiceOuterClass.UpdateProfileRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getUpdateProfileMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.DeleteProfileResponse deleteProfile(oscal.services.v1.OscalServiceOuterClass.DeleteProfileRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getDeleteProfileMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * Component Definition operations
     * </pre>
     */
    public oscal.services.v1.OscalServiceOuterClass.GetComponentDefinitionResponse getComponentDefinition(oscal.services.v1.OscalServiceOuterClass.GetComponentDefinitionRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getGetComponentDefinitionMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.ListComponentDefinitionsResponse listComponentDefinitions(oscal.services.v1.OscalServiceOuterClass.ListComponentDefinitionsRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getListComponentDefinitionsMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.CreateComponentDefinitionResponse createComponentDefinition(oscal.services.v1.OscalServiceOuterClass.CreateComponentDefinitionRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getCreateComponentDefinitionMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.UpdateComponentDefinitionResponse updateComponentDefinition(oscal.services.v1.OscalServiceOuterClass.UpdateComponentDefinitionRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getUpdateComponentDefinitionMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.DeleteComponentDefinitionResponse deleteComponentDefinition(oscal.services.v1.OscalServiceOuterClass.DeleteComponentDefinitionRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getDeleteComponentDefinitionMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * SSP operations
     * </pre>
     */
    public oscal.services.v1.OscalServiceOuterClass.GetSspResponse getSsp(oscal.services.v1.OscalServiceOuterClass.GetSspRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getGetSspMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.ListSspsResponse listSsps(oscal.services.v1.OscalServiceOuterClass.ListSspsRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getListSspsMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.CreateSspResponse createSsp(oscal.services.v1.OscalServiceOuterClass.CreateSspRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getCreateSspMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.UpdateSspResponse updateSsp(oscal.services.v1.OscalServiceOuterClass.UpdateSspRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getUpdateSspMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.DeleteSspResponse deleteSsp(oscal.services.v1.OscalServiceOuterClass.DeleteSspRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getDeleteSspMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * Assessment Plan operations
     * </pre>
     */
    public oscal.services.v1.OscalServiceOuterClass.GetAssessmentPlanResponse getAssessmentPlan(oscal.services.v1.OscalServiceOuterClass.GetAssessmentPlanRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getGetAssessmentPlanMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.ListAssessmentPlansResponse listAssessmentPlans(oscal.services.v1.OscalServiceOuterClass.ListAssessmentPlansRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getListAssessmentPlansMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.CreateAssessmentPlanResponse createAssessmentPlan(oscal.services.v1.OscalServiceOuterClass.CreateAssessmentPlanRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getCreateAssessmentPlanMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentPlanResponse updateAssessmentPlan(oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentPlanRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getUpdateAssessmentPlanMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentPlanResponse deleteAssessmentPlan(oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentPlanRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getDeleteAssessmentPlanMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * Assessment Results operations
     * </pre>
     */
    public oscal.services.v1.OscalServiceOuterClass.GetAssessmentResultsResponse getAssessmentResults(oscal.services.v1.OscalServiceOuterClass.GetAssessmentResultsRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getGetAssessmentResultsMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.ListAssessmentResultsResponse listAssessmentResults(oscal.services.v1.OscalServiceOuterClass.ListAssessmentResultsRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getListAssessmentResultsMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.CreateAssessmentResultsResponse createAssessmentResults(oscal.services.v1.OscalServiceOuterClass.CreateAssessmentResultsRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getCreateAssessmentResultsMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentResultsResponse updateAssessmentResults(oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentResultsRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getUpdateAssessmentResultsMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentResultsResponse deleteAssessmentResults(oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentResultsRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getDeleteAssessmentResultsMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * POA&amp;M operations
     * </pre>
     */
    public oscal.services.v1.OscalServiceOuterClass.GetPoamResponse getPoam(oscal.services.v1.OscalServiceOuterClass.GetPoamRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getGetPoamMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.ListPoamsResponse listPoams(oscal.services.v1.OscalServiceOuterClass.ListPoamsRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getListPoamsMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.CreatePoamResponse createPoam(oscal.services.v1.OscalServiceOuterClass.CreatePoamRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getCreatePoamMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.UpdatePoamResponse updatePoam(oscal.services.v1.OscalServiceOuterClass.UpdatePoamRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getUpdatePoamMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.DeletePoamResponse deletePoam(oscal.services.v1.OscalServiceOuterClass.DeletePoamRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getDeletePoamMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * Mapping operations
     * </pre>
     */
    public oscal.services.v1.OscalServiceOuterClass.GetMappingResponse getMapping(oscal.services.v1.OscalServiceOuterClass.GetMappingRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getGetMappingMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.ListMappingsResponse listMappings(oscal.services.v1.OscalServiceOuterClass.ListMappingsRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getListMappingsMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.CreateMappingResponse createMapping(oscal.services.v1.OscalServiceOuterClass.CreateMappingRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getCreateMappingMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.UpdateMappingResponse updateMapping(oscal.services.v1.OscalServiceOuterClass.UpdateMappingRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getUpdateMappingMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.OscalServiceOuterClass.DeleteMappingResponse deleteMapping(oscal.services.v1.OscalServiceOuterClass.DeleteMappingRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getDeleteMappingMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * Search operations
     * </pre>
     */
    public oscal.services.v1.OscalServiceOuterClass.SearchResponse search(oscal.services.v1.OscalServiceOuterClass.SearchRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getSearchMethod(), getCallOptions(), request);
    }
  }

  /**
   * A stub to allow clients to do ListenableFuture-style rpc calls to service OscalService.
   * <pre>
   * OSCAL Service provides CRUD operations for all OSCAL models
   * </pre>
   */
  public static final class OscalServiceFutureStub
      extends io.grpc.stub.AbstractFutureStub<OscalServiceFutureStub> {
    private OscalServiceFutureStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected OscalServiceFutureStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new OscalServiceFutureStub(channel, callOptions);
    }

    /**
     * <pre>
     * Catalog operations
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.GetCatalogResponse> getCatalog(
        oscal.services.v1.OscalServiceOuterClass.GetCatalogRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getGetCatalogMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.ListCatalogsResponse> listCatalogs(
        oscal.services.v1.OscalServiceOuterClass.ListCatalogsRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getListCatalogsMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.CreateCatalogResponse> createCatalog(
        oscal.services.v1.OscalServiceOuterClass.CreateCatalogRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getCreateCatalogMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.UpdateCatalogResponse> updateCatalog(
        oscal.services.v1.OscalServiceOuterClass.UpdateCatalogRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getUpdateCatalogMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.DeleteCatalogResponse> deleteCatalog(
        oscal.services.v1.OscalServiceOuterClass.DeleteCatalogRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getDeleteCatalogMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * Profile operations
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.GetProfileResponse> getProfile(
        oscal.services.v1.OscalServiceOuterClass.GetProfileRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getGetProfileMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.ListProfilesResponse> listProfiles(
        oscal.services.v1.OscalServiceOuterClass.ListProfilesRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getListProfilesMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.CreateProfileResponse> createProfile(
        oscal.services.v1.OscalServiceOuterClass.CreateProfileRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getCreateProfileMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.UpdateProfileResponse> updateProfile(
        oscal.services.v1.OscalServiceOuterClass.UpdateProfileRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getUpdateProfileMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.DeleteProfileResponse> deleteProfile(
        oscal.services.v1.OscalServiceOuterClass.DeleteProfileRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getDeleteProfileMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * Component Definition operations
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.GetComponentDefinitionResponse> getComponentDefinition(
        oscal.services.v1.OscalServiceOuterClass.GetComponentDefinitionRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getGetComponentDefinitionMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.ListComponentDefinitionsResponse> listComponentDefinitions(
        oscal.services.v1.OscalServiceOuterClass.ListComponentDefinitionsRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getListComponentDefinitionsMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.CreateComponentDefinitionResponse> createComponentDefinition(
        oscal.services.v1.OscalServiceOuterClass.CreateComponentDefinitionRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getCreateComponentDefinitionMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.UpdateComponentDefinitionResponse> updateComponentDefinition(
        oscal.services.v1.OscalServiceOuterClass.UpdateComponentDefinitionRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getUpdateComponentDefinitionMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.DeleteComponentDefinitionResponse> deleteComponentDefinition(
        oscal.services.v1.OscalServiceOuterClass.DeleteComponentDefinitionRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getDeleteComponentDefinitionMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * SSP operations
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.GetSspResponse> getSsp(
        oscal.services.v1.OscalServiceOuterClass.GetSspRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getGetSspMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.ListSspsResponse> listSsps(
        oscal.services.v1.OscalServiceOuterClass.ListSspsRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getListSspsMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.CreateSspResponse> createSsp(
        oscal.services.v1.OscalServiceOuterClass.CreateSspRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getCreateSspMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.UpdateSspResponse> updateSsp(
        oscal.services.v1.OscalServiceOuterClass.UpdateSspRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getUpdateSspMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.DeleteSspResponse> deleteSsp(
        oscal.services.v1.OscalServiceOuterClass.DeleteSspRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getDeleteSspMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * Assessment Plan operations
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.GetAssessmentPlanResponse> getAssessmentPlan(
        oscal.services.v1.OscalServiceOuterClass.GetAssessmentPlanRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getGetAssessmentPlanMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.ListAssessmentPlansResponse> listAssessmentPlans(
        oscal.services.v1.OscalServiceOuterClass.ListAssessmentPlansRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getListAssessmentPlansMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.CreateAssessmentPlanResponse> createAssessmentPlan(
        oscal.services.v1.OscalServiceOuterClass.CreateAssessmentPlanRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getCreateAssessmentPlanMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentPlanResponse> updateAssessmentPlan(
        oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentPlanRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getUpdateAssessmentPlanMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentPlanResponse> deleteAssessmentPlan(
        oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentPlanRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getDeleteAssessmentPlanMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * Assessment Results operations
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.GetAssessmentResultsResponse> getAssessmentResults(
        oscal.services.v1.OscalServiceOuterClass.GetAssessmentResultsRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getGetAssessmentResultsMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.ListAssessmentResultsResponse> listAssessmentResults(
        oscal.services.v1.OscalServiceOuterClass.ListAssessmentResultsRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getListAssessmentResultsMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.CreateAssessmentResultsResponse> createAssessmentResults(
        oscal.services.v1.OscalServiceOuterClass.CreateAssessmentResultsRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getCreateAssessmentResultsMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentResultsResponse> updateAssessmentResults(
        oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentResultsRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getUpdateAssessmentResultsMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentResultsResponse> deleteAssessmentResults(
        oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentResultsRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getDeleteAssessmentResultsMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * POA&amp;M operations
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.GetPoamResponse> getPoam(
        oscal.services.v1.OscalServiceOuterClass.GetPoamRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getGetPoamMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.ListPoamsResponse> listPoams(
        oscal.services.v1.OscalServiceOuterClass.ListPoamsRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getListPoamsMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.CreatePoamResponse> createPoam(
        oscal.services.v1.OscalServiceOuterClass.CreatePoamRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getCreatePoamMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.UpdatePoamResponse> updatePoam(
        oscal.services.v1.OscalServiceOuterClass.UpdatePoamRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getUpdatePoamMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.DeletePoamResponse> deletePoam(
        oscal.services.v1.OscalServiceOuterClass.DeletePoamRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getDeletePoamMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * Mapping operations
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.GetMappingResponse> getMapping(
        oscal.services.v1.OscalServiceOuterClass.GetMappingRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getGetMappingMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.ListMappingsResponse> listMappings(
        oscal.services.v1.OscalServiceOuterClass.ListMappingsRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getListMappingsMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.CreateMappingResponse> createMapping(
        oscal.services.v1.OscalServiceOuterClass.CreateMappingRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getCreateMappingMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.UpdateMappingResponse> updateMapping(
        oscal.services.v1.OscalServiceOuterClass.UpdateMappingRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getUpdateMappingMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.DeleteMappingResponse> deleteMapping(
        oscal.services.v1.OscalServiceOuterClass.DeleteMappingRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getDeleteMappingMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * Search operations
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.OscalServiceOuterClass.SearchResponse> search(
        oscal.services.v1.OscalServiceOuterClass.SearchRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getSearchMethod(), getCallOptions()), request);
    }
  }

  private static final int METHODID_GET_CATALOG = 0;
  private static final int METHODID_LIST_CATALOGS = 1;
  private static final int METHODID_CREATE_CATALOG = 2;
  private static final int METHODID_UPDATE_CATALOG = 3;
  private static final int METHODID_DELETE_CATALOG = 4;
  private static final int METHODID_GET_PROFILE = 5;
  private static final int METHODID_LIST_PROFILES = 6;
  private static final int METHODID_CREATE_PROFILE = 7;
  private static final int METHODID_UPDATE_PROFILE = 8;
  private static final int METHODID_DELETE_PROFILE = 9;
  private static final int METHODID_GET_COMPONENT_DEFINITION = 10;
  private static final int METHODID_LIST_COMPONENT_DEFINITIONS = 11;
  private static final int METHODID_CREATE_COMPONENT_DEFINITION = 12;
  private static final int METHODID_UPDATE_COMPONENT_DEFINITION = 13;
  private static final int METHODID_DELETE_COMPONENT_DEFINITION = 14;
  private static final int METHODID_GET_SSP = 15;
  private static final int METHODID_LIST_SSPS = 16;
  private static final int METHODID_CREATE_SSP = 17;
  private static final int METHODID_UPDATE_SSP = 18;
  private static final int METHODID_DELETE_SSP = 19;
  private static final int METHODID_GET_ASSESSMENT_PLAN = 20;
  private static final int METHODID_LIST_ASSESSMENT_PLANS = 21;
  private static final int METHODID_CREATE_ASSESSMENT_PLAN = 22;
  private static final int METHODID_UPDATE_ASSESSMENT_PLAN = 23;
  private static final int METHODID_DELETE_ASSESSMENT_PLAN = 24;
  private static final int METHODID_GET_ASSESSMENT_RESULTS = 25;
  private static final int METHODID_LIST_ASSESSMENT_RESULTS = 26;
  private static final int METHODID_CREATE_ASSESSMENT_RESULTS = 27;
  private static final int METHODID_UPDATE_ASSESSMENT_RESULTS = 28;
  private static final int METHODID_DELETE_ASSESSMENT_RESULTS = 29;
  private static final int METHODID_GET_POAM = 30;
  private static final int METHODID_LIST_POAMS = 31;
  private static final int METHODID_CREATE_POAM = 32;
  private static final int METHODID_UPDATE_POAM = 33;
  private static final int METHODID_DELETE_POAM = 34;
  private static final int METHODID_GET_MAPPING = 35;
  private static final int METHODID_LIST_MAPPINGS = 36;
  private static final int METHODID_CREATE_MAPPING = 37;
  private static final int METHODID_UPDATE_MAPPING = 38;
  private static final int METHODID_DELETE_MAPPING = 39;
  private static final int METHODID_SEARCH = 40;

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
        case METHODID_GET_CATALOG:
          serviceImpl.getCatalog((oscal.services.v1.OscalServiceOuterClass.GetCatalogRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.GetCatalogResponse>) responseObserver);
          break;
        case METHODID_LIST_CATALOGS:
          serviceImpl.listCatalogs((oscal.services.v1.OscalServiceOuterClass.ListCatalogsRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.ListCatalogsResponse>) responseObserver);
          break;
        case METHODID_CREATE_CATALOG:
          serviceImpl.createCatalog((oscal.services.v1.OscalServiceOuterClass.CreateCatalogRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.CreateCatalogResponse>) responseObserver);
          break;
        case METHODID_UPDATE_CATALOG:
          serviceImpl.updateCatalog((oscal.services.v1.OscalServiceOuterClass.UpdateCatalogRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.UpdateCatalogResponse>) responseObserver);
          break;
        case METHODID_DELETE_CATALOG:
          serviceImpl.deleteCatalog((oscal.services.v1.OscalServiceOuterClass.DeleteCatalogRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.DeleteCatalogResponse>) responseObserver);
          break;
        case METHODID_GET_PROFILE:
          serviceImpl.getProfile((oscal.services.v1.OscalServiceOuterClass.GetProfileRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.GetProfileResponse>) responseObserver);
          break;
        case METHODID_LIST_PROFILES:
          serviceImpl.listProfiles((oscal.services.v1.OscalServiceOuterClass.ListProfilesRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.ListProfilesResponse>) responseObserver);
          break;
        case METHODID_CREATE_PROFILE:
          serviceImpl.createProfile((oscal.services.v1.OscalServiceOuterClass.CreateProfileRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.CreateProfileResponse>) responseObserver);
          break;
        case METHODID_UPDATE_PROFILE:
          serviceImpl.updateProfile((oscal.services.v1.OscalServiceOuterClass.UpdateProfileRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.UpdateProfileResponse>) responseObserver);
          break;
        case METHODID_DELETE_PROFILE:
          serviceImpl.deleteProfile((oscal.services.v1.OscalServiceOuterClass.DeleteProfileRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.DeleteProfileResponse>) responseObserver);
          break;
        case METHODID_GET_COMPONENT_DEFINITION:
          serviceImpl.getComponentDefinition((oscal.services.v1.OscalServiceOuterClass.GetComponentDefinitionRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.GetComponentDefinitionResponse>) responseObserver);
          break;
        case METHODID_LIST_COMPONENT_DEFINITIONS:
          serviceImpl.listComponentDefinitions((oscal.services.v1.OscalServiceOuterClass.ListComponentDefinitionsRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.ListComponentDefinitionsResponse>) responseObserver);
          break;
        case METHODID_CREATE_COMPONENT_DEFINITION:
          serviceImpl.createComponentDefinition((oscal.services.v1.OscalServiceOuterClass.CreateComponentDefinitionRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.CreateComponentDefinitionResponse>) responseObserver);
          break;
        case METHODID_UPDATE_COMPONENT_DEFINITION:
          serviceImpl.updateComponentDefinition((oscal.services.v1.OscalServiceOuterClass.UpdateComponentDefinitionRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.UpdateComponentDefinitionResponse>) responseObserver);
          break;
        case METHODID_DELETE_COMPONENT_DEFINITION:
          serviceImpl.deleteComponentDefinition((oscal.services.v1.OscalServiceOuterClass.DeleteComponentDefinitionRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.DeleteComponentDefinitionResponse>) responseObserver);
          break;
        case METHODID_GET_SSP:
          serviceImpl.getSsp((oscal.services.v1.OscalServiceOuterClass.GetSspRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.GetSspResponse>) responseObserver);
          break;
        case METHODID_LIST_SSPS:
          serviceImpl.listSsps((oscal.services.v1.OscalServiceOuterClass.ListSspsRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.ListSspsResponse>) responseObserver);
          break;
        case METHODID_CREATE_SSP:
          serviceImpl.createSsp((oscal.services.v1.OscalServiceOuterClass.CreateSspRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.CreateSspResponse>) responseObserver);
          break;
        case METHODID_UPDATE_SSP:
          serviceImpl.updateSsp((oscal.services.v1.OscalServiceOuterClass.UpdateSspRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.UpdateSspResponse>) responseObserver);
          break;
        case METHODID_DELETE_SSP:
          serviceImpl.deleteSsp((oscal.services.v1.OscalServiceOuterClass.DeleteSspRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.DeleteSspResponse>) responseObserver);
          break;
        case METHODID_GET_ASSESSMENT_PLAN:
          serviceImpl.getAssessmentPlan((oscal.services.v1.OscalServiceOuterClass.GetAssessmentPlanRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.GetAssessmentPlanResponse>) responseObserver);
          break;
        case METHODID_LIST_ASSESSMENT_PLANS:
          serviceImpl.listAssessmentPlans((oscal.services.v1.OscalServiceOuterClass.ListAssessmentPlansRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.ListAssessmentPlansResponse>) responseObserver);
          break;
        case METHODID_CREATE_ASSESSMENT_PLAN:
          serviceImpl.createAssessmentPlan((oscal.services.v1.OscalServiceOuterClass.CreateAssessmentPlanRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.CreateAssessmentPlanResponse>) responseObserver);
          break;
        case METHODID_UPDATE_ASSESSMENT_PLAN:
          serviceImpl.updateAssessmentPlan((oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentPlanRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentPlanResponse>) responseObserver);
          break;
        case METHODID_DELETE_ASSESSMENT_PLAN:
          serviceImpl.deleteAssessmentPlan((oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentPlanRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentPlanResponse>) responseObserver);
          break;
        case METHODID_GET_ASSESSMENT_RESULTS:
          serviceImpl.getAssessmentResults((oscal.services.v1.OscalServiceOuterClass.GetAssessmentResultsRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.GetAssessmentResultsResponse>) responseObserver);
          break;
        case METHODID_LIST_ASSESSMENT_RESULTS:
          serviceImpl.listAssessmentResults((oscal.services.v1.OscalServiceOuterClass.ListAssessmentResultsRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.ListAssessmentResultsResponse>) responseObserver);
          break;
        case METHODID_CREATE_ASSESSMENT_RESULTS:
          serviceImpl.createAssessmentResults((oscal.services.v1.OscalServiceOuterClass.CreateAssessmentResultsRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.CreateAssessmentResultsResponse>) responseObserver);
          break;
        case METHODID_UPDATE_ASSESSMENT_RESULTS:
          serviceImpl.updateAssessmentResults((oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentResultsRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentResultsResponse>) responseObserver);
          break;
        case METHODID_DELETE_ASSESSMENT_RESULTS:
          serviceImpl.deleteAssessmentResults((oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentResultsRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentResultsResponse>) responseObserver);
          break;
        case METHODID_GET_POAM:
          serviceImpl.getPoam((oscal.services.v1.OscalServiceOuterClass.GetPoamRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.GetPoamResponse>) responseObserver);
          break;
        case METHODID_LIST_POAMS:
          serviceImpl.listPoams((oscal.services.v1.OscalServiceOuterClass.ListPoamsRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.ListPoamsResponse>) responseObserver);
          break;
        case METHODID_CREATE_POAM:
          serviceImpl.createPoam((oscal.services.v1.OscalServiceOuterClass.CreatePoamRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.CreatePoamResponse>) responseObserver);
          break;
        case METHODID_UPDATE_POAM:
          serviceImpl.updatePoam((oscal.services.v1.OscalServiceOuterClass.UpdatePoamRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.UpdatePoamResponse>) responseObserver);
          break;
        case METHODID_DELETE_POAM:
          serviceImpl.deletePoam((oscal.services.v1.OscalServiceOuterClass.DeletePoamRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.DeletePoamResponse>) responseObserver);
          break;
        case METHODID_GET_MAPPING:
          serviceImpl.getMapping((oscal.services.v1.OscalServiceOuterClass.GetMappingRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.GetMappingResponse>) responseObserver);
          break;
        case METHODID_LIST_MAPPINGS:
          serviceImpl.listMappings((oscal.services.v1.OscalServiceOuterClass.ListMappingsRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.ListMappingsResponse>) responseObserver);
          break;
        case METHODID_CREATE_MAPPING:
          serviceImpl.createMapping((oscal.services.v1.OscalServiceOuterClass.CreateMappingRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.CreateMappingResponse>) responseObserver);
          break;
        case METHODID_UPDATE_MAPPING:
          serviceImpl.updateMapping((oscal.services.v1.OscalServiceOuterClass.UpdateMappingRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.UpdateMappingResponse>) responseObserver);
          break;
        case METHODID_DELETE_MAPPING:
          serviceImpl.deleteMapping((oscal.services.v1.OscalServiceOuterClass.DeleteMappingRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.DeleteMappingResponse>) responseObserver);
          break;
        case METHODID_SEARCH:
          serviceImpl.search((oscal.services.v1.OscalServiceOuterClass.SearchRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.OscalServiceOuterClass.SearchResponse>) responseObserver);
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
          getGetCatalogMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.GetCatalogRequest,
              oscal.services.v1.OscalServiceOuterClass.GetCatalogResponse>(
                service, METHODID_GET_CATALOG)))
        .addMethod(
          getListCatalogsMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.ListCatalogsRequest,
              oscal.services.v1.OscalServiceOuterClass.ListCatalogsResponse>(
                service, METHODID_LIST_CATALOGS)))
        .addMethod(
          getCreateCatalogMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.CreateCatalogRequest,
              oscal.services.v1.OscalServiceOuterClass.CreateCatalogResponse>(
                service, METHODID_CREATE_CATALOG)))
        .addMethod(
          getUpdateCatalogMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.UpdateCatalogRequest,
              oscal.services.v1.OscalServiceOuterClass.UpdateCatalogResponse>(
                service, METHODID_UPDATE_CATALOG)))
        .addMethod(
          getDeleteCatalogMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.DeleteCatalogRequest,
              oscal.services.v1.OscalServiceOuterClass.DeleteCatalogResponse>(
                service, METHODID_DELETE_CATALOG)))
        .addMethod(
          getGetProfileMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.GetProfileRequest,
              oscal.services.v1.OscalServiceOuterClass.GetProfileResponse>(
                service, METHODID_GET_PROFILE)))
        .addMethod(
          getListProfilesMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.ListProfilesRequest,
              oscal.services.v1.OscalServiceOuterClass.ListProfilesResponse>(
                service, METHODID_LIST_PROFILES)))
        .addMethod(
          getCreateProfileMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.CreateProfileRequest,
              oscal.services.v1.OscalServiceOuterClass.CreateProfileResponse>(
                service, METHODID_CREATE_PROFILE)))
        .addMethod(
          getUpdateProfileMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.UpdateProfileRequest,
              oscal.services.v1.OscalServiceOuterClass.UpdateProfileResponse>(
                service, METHODID_UPDATE_PROFILE)))
        .addMethod(
          getDeleteProfileMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.DeleteProfileRequest,
              oscal.services.v1.OscalServiceOuterClass.DeleteProfileResponse>(
                service, METHODID_DELETE_PROFILE)))
        .addMethod(
          getGetComponentDefinitionMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.GetComponentDefinitionRequest,
              oscal.services.v1.OscalServiceOuterClass.GetComponentDefinitionResponse>(
                service, METHODID_GET_COMPONENT_DEFINITION)))
        .addMethod(
          getListComponentDefinitionsMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.ListComponentDefinitionsRequest,
              oscal.services.v1.OscalServiceOuterClass.ListComponentDefinitionsResponse>(
                service, METHODID_LIST_COMPONENT_DEFINITIONS)))
        .addMethod(
          getCreateComponentDefinitionMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.CreateComponentDefinitionRequest,
              oscal.services.v1.OscalServiceOuterClass.CreateComponentDefinitionResponse>(
                service, METHODID_CREATE_COMPONENT_DEFINITION)))
        .addMethod(
          getUpdateComponentDefinitionMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.UpdateComponentDefinitionRequest,
              oscal.services.v1.OscalServiceOuterClass.UpdateComponentDefinitionResponse>(
                service, METHODID_UPDATE_COMPONENT_DEFINITION)))
        .addMethod(
          getDeleteComponentDefinitionMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.DeleteComponentDefinitionRequest,
              oscal.services.v1.OscalServiceOuterClass.DeleteComponentDefinitionResponse>(
                service, METHODID_DELETE_COMPONENT_DEFINITION)))
        .addMethod(
          getGetSspMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.GetSspRequest,
              oscal.services.v1.OscalServiceOuterClass.GetSspResponse>(
                service, METHODID_GET_SSP)))
        .addMethod(
          getListSspsMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.ListSspsRequest,
              oscal.services.v1.OscalServiceOuterClass.ListSspsResponse>(
                service, METHODID_LIST_SSPS)))
        .addMethod(
          getCreateSspMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.CreateSspRequest,
              oscal.services.v1.OscalServiceOuterClass.CreateSspResponse>(
                service, METHODID_CREATE_SSP)))
        .addMethod(
          getUpdateSspMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.UpdateSspRequest,
              oscal.services.v1.OscalServiceOuterClass.UpdateSspResponse>(
                service, METHODID_UPDATE_SSP)))
        .addMethod(
          getDeleteSspMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.DeleteSspRequest,
              oscal.services.v1.OscalServiceOuterClass.DeleteSspResponse>(
                service, METHODID_DELETE_SSP)))
        .addMethod(
          getGetAssessmentPlanMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.GetAssessmentPlanRequest,
              oscal.services.v1.OscalServiceOuterClass.GetAssessmentPlanResponse>(
                service, METHODID_GET_ASSESSMENT_PLAN)))
        .addMethod(
          getListAssessmentPlansMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.ListAssessmentPlansRequest,
              oscal.services.v1.OscalServiceOuterClass.ListAssessmentPlansResponse>(
                service, METHODID_LIST_ASSESSMENT_PLANS)))
        .addMethod(
          getCreateAssessmentPlanMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.CreateAssessmentPlanRequest,
              oscal.services.v1.OscalServiceOuterClass.CreateAssessmentPlanResponse>(
                service, METHODID_CREATE_ASSESSMENT_PLAN)))
        .addMethod(
          getUpdateAssessmentPlanMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentPlanRequest,
              oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentPlanResponse>(
                service, METHODID_UPDATE_ASSESSMENT_PLAN)))
        .addMethod(
          getDeleteAssessmentPlanMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentPlanRequest,
              oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentPlanResponse>(
                service, METHODID_DELETE_ASSESSMENT_PLAN)))
        .addMethod(
          getGetAssessmentResultsMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.GetAssessmentResultsRequest,
              oscal.services.v1.OscalServiceOuterClass.GetAssessmentResultsResponse>(
                service, METHODID_GET_ASSESSMENT_RESULTS)))
        .addMethod(
          getListAssessmentResultsMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.ListAssessmentResultsRequest,
              oscal.services.v1.OscalServiceOuterClass.ListAssessmentResultsResponse>(
                service, METHODID_LIST_ASSESSMENT_RESULTS)))
        .addMethod(
          getCreateAssessmentResultsMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.CreateAssessmentResultsRequest,
              oscal.services.v1.OscalServiceOuterClass.CreateAssessmentResultsResponse>(
                service, METHODID_CREATE_ASSESSMENT_RESULTS)))
        .addMethod(
          getUpdateAssessmentResultsMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentResultsRequest,
              oscal.services.v1.OscalServiceOuterClass.UpdateAssessmentResultsResponse>(
                service, METHODID_UPDATE_ASSESSMENT_RESULTS)))
        .addMethod(
          getDeleteAssessmentResultsMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentResultsRequest,
              oscal.services.v1.OscalServiceOuterClass.DeleteAssessmentResultsResponse>(
                service, METHODID_DELETE_ASSESSMENT_RESULTS)))
        .addMethod(
          getGetPoamMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.GetPoamRequest,
              oscal.services.v1.OscalServiceOuterClass.GetPoamResponse>(
                service, METHODID_GET_POAM)))
        .addMethod(
          getListPoamsMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.ListPoamsRequest,
              oscal.services.v1.OscalServiceOuterClass.ListPoamsResponse>(
                service, METHODID_LIST_POAMS)))
        .addMethod(
          getCreatePoamMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.CreatePoamRequest,
              oscal.services.v1.OscalServiceOuterClass.CreatePoamResponse>(
                service, METHODID_CREATE_POAM)))
        .addMethod(
          getUpdatePoamMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.UpdatePoamRequest,
              oscal.services.v1.OscalServiceOuterClass.UpdatePoamResponse>(
                service, METHODID_UPDATE_POAM)))
        .addMethod(
          getDeletePoamMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.DeletePoamRequest,
              oscal.services.v1.OscalServiceOuterClass.DeletePoamResponse>(
                service, METHODID_DELETE_POAM)))
        .addMethod(
          getGetMappingMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.GetMappingRequest,
              oscal.services.v1.OscalServiceOuterClass.GetMappingResponse>(
                service, METHODID_GET_MAPPING)))
        .addMethod(
          getListMappingsMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.ListMappingsRequest,
              oscal.services.v1.OscalServiceOuterClass.ListMappingsResponse>(
                service, METHODID_LIST_MAPPINGS)))
        .addMethod(
          getCreateMappingMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.CreateMappingRequest,
              oscal.services.v1.OscalServiceOuterClass.CreateMappingResponse>(
                service, METHODID_CREATE_MAPPING)))
        .addMethod(
          getUpdateMappingMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.UpdateMappingRequest,
              oscal.services.v1.OscalServiceOuterClass.UpdateMappingResponse>(
                service, METHODID_UPDATE_MAPPING)))
        .addMethod(
          getDeleteMappingMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.DeleteMappingRequest,
              oscal.services.v1.OscalServiceOuterClass.DeleteMappingResponse>(
                service, METHODID_DELETE_MAPPING)))
        .addMethod(
          getSearchMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.OscalServiceOuterClass.SearchRequest,
              oscal.services.v1.OscalServiceOuterClass.SearchResponse>(
                service, METHODID_SEARCH)))
        .build();
  }

  private static abstract class OscalServiceBaseDescriptorSupplier
      implements io.grpc.protobuf.ProtoFileDescriptorSupplier, io.grpc.protobuf.ProtoServiceDescriptorSupplier {
    OscalServiceBaseDescriptorSupplier() {}

    @java.lang.Override
    public com.google.protobuf.Descriptors.FileDescriptor getFileDescriptor() {
      return oscal.services.v1.OscalServiceOuterClass.getDescriptor();
    }

    @java.lang.Override
    public com.google.protobuf.Descriptors.ServiceDescriptor getServiceDescriptor() {
      return getFileDescriptor().findServiceByName("OscalService");
    }
  }

  private static final class OscalServiceFileDescriptorSupplier
      extends OscalServiceBaseDescriptorSupplier {
    OscalServiceFileDescriptorSupplier() {}
  }

  private static final class OscalServiceMethodDescriptorSupplier
      extends OscalServiceBaseDescriptorSupplier
      implements io.grpc.protobuf.ProtoMethodDescriptorSupplier {
    private final java.lang.String methodName;

    OscalServiceMethodDescriptorSupplier(java.lang.String methodName) {
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
      synchronized (OscalServiceGrpc.class) {
        result = serviceDescriptor;
        if (result == null) {
          serviceDescriptor = result = io.grpc.ServiceDescriptor.newBuilder(SERVICE_NAME)
              .setSchemaDescriptor(new OscalServiceFileDescriptorSupplier())
              .addMethod(getGetCatalogMethod())
              .addMethod(getListCatalogsMethod())
              .addMethod(getCreateCatalogMethod())
              .addMethod(getUpdateCatalogMethod())
              .addMethod(getDeleteCatalogMethod())
              .addMethod(getGetProfileMethod())
              .addMethod(getListProfilesMethod())
              .addMethod(getCreateProfileMethod())
              .addMethod(getUpdateProfileMethod())
              .addMethod(getDeleteProfileMethod())
              .addMethod(getGetComponentDefinitionMethod())
              .addMethod(getListComponentDefinitionsMethod())
              .addMethod(getCreateComponentDefinitionMethod())
              .addMethod(getUpdateComponentDefinitionMethod())
              .addMethod(getDeleteComponentDefinitionMethod())
              .addMethod(getGetSspMethod())
              .addMethod(getListSspsMethod())
              .addMethod(getCreateSspMethod())
              .addMethod(getUpdateSspMethod())
              .addMethod(getDeleteSspMethod())
              .addMethod(getGetAssessmentPlanMethod())
              .addMethod(getListAssessmentPlansMethod())
              .addMethod(getCreateAssessmentPlanMethod())
              .addMethod(getUpdateAssessmentPlanMethod())
              .addMethod(getDeleteAssessmentPlanMethod())
              .addMethod(getGetAssessmentResultsMethod())
              .addMethod(getListAssessmentResultsMethod())
              .addMethod(getCreateAssessmentResultsMethod())
              .addMethod(getUpdateAssessmentResultsMethod())
              .addMethod(getDeleteAssessmentResultsMethod())
              .addMethod(getGetPoamMethod())
              .addMethod(getListPoamsMethod())
              .addMethod(getCreatePoamMethod())
              .addMethod(getUpdatePoamMethod())
              .addMethod(getDeletePoamMethod())
              .addMethod(getGetMappingMethod())
              .addMethod(getListMappingsMethod())
              .addMethod(getCreateMappingMethod())
              .addMethod(getUpdateMappingMethod())
              .addMethod(getDeleteMappingMethod())
              .addMethod(getSearchMethod())
              .build();
        }
      }
    }
    return result;
  }
}
