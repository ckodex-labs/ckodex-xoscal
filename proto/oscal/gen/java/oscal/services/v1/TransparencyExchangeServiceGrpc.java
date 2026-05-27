package oscal.services.v1;

import static io.grpc.MethodDescriptor.generateFullMethodName;

/**
 * <pre>
 * TransparencyExchangeService provides content-addressed attestation claims
 * and xBOM evidence exchange for xOSCAL.
 * </pre>
 */
@io.grpc.stub.annotations.GrpcGenerated
public final class TransparencyExchangeServiceGrpc {

  private TransparencyExchangeServiceGrpc() {}

  public static final java.lang.String SERVICE_NAME = "oscal.services.v1.TransparencyExchangeService";

  // Static method descriptors that strictly reflect the proto.
  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.TransparencyExchangeServiceOuterClass.CreateClaimRequest,
      oscal.services.v1.TransparencyExchangeServiceOuterClass.CreateClaimResponse> getCreateClaimMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "CreateClaim",
      requestType = oscal.services.v1.TransparencyExchangeServiceOuterClass.CreateClaimRequest.class,
      responseType = oscal.services.v1.TransparencyExchangeServiceOuterClass.CreateClaimResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.TransparencyExchangeServiceOuterClass.CreateClaimRequest,
      oscal.services.v1.TransparencyExchangeServiceOuterClass.CreateClaimResponse> getCreateClaimMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.TransparencyExchangeServiceOuterClass.CreateClaimRequest, oscal.services.v1.TransparencyExchangeServiceOuterClass.CreateClaimResponse> getCreateClaimMethod;
    if ((getCreateClaimMethod = TransparencyExchangeServiceGrpc.getCreateClaimMethod) == null) {
      synchronized (TransparencyExchangeServiceGrpc.class) {
        if ((getCreateClaimMethod = TransparencyExchangeServiceGrpc.getCreateClaimMethod) == null) {
          TransparencyExchangeServiceGrpc.getCreateClaimMethod = getCreateClaimMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.TransparencyExchangeServiceOuterClass.CreateClaimRequest, oscal.services.v1.TransparencyExchangeServiceOuterClass.CreateClaimResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "CreateClaim"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyExchangeServiceOuterClass.CreateClaimRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyExchangeServiceOuterClass.CreateClaimResponse.getDefaultInstance()))
              .setSchemaDescriptor(new TransparencyExchangeServiceMethodDescriptorSupplier("CreateClaim"))
              .build();
        }
      }
    }
    return getCreateClaimMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.TransparencyExchangeServiceOuterClass.GetClaimRequest,
      oscal.services.v1.TransparencyExchangeServiceOuterClass.GetClaimResponse> getGetClaimMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GetClaim",
      requestType = oscal.services.v1.TransparencyExchangeServiceOuterClass.GetClaimRequest.class,
      responseType = oscal.services.v1.TransparencyExchangeServiceOuterClass.GetClaimResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.TransparencyExchangeServiceOuterClass.GetClaimRequest,
      oscal.services.v1.TransparencyExchangeServiceOuterClass.GetClaimResponse> getGetClaimMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.TransparencyExchangeServiceOuterClass.GetClaimRequest, oscal.services.v1.TransparencyExchangeServiceOuterClass.GetClaimResponse> getGetClaimMethod;
    if ((getGetClaimMethod = TransparencyExchangeServiceGrpc.getGetClaimMethod) == null) {
      synchronized (TransparencyExchangeServiceGrpc.class) {
        if ((getGetClaimMethod = TransparencyExchangeServiceGrpc.getGetClaimMethod) == null) {
          TransparencyExchangeServiceGrpc.getGetClaimMethod = getGetClaimMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.TransparencyExchangeServiceOuterClass.GetClaimRequest, oscal.services.v1.TransparencyExchangeServiceOuterClass.GetClaimResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GetClaim"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyExchangeServiceOuterClass.GetClaimRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyExchangeServiceOuterClass.GetClaimResponse.getDefaultInstance()))
              .setSchemaDescriptor(new TransparencyExchangeServiceMethodDescriptorSupplier("GetClaim"))
              .build();
        }
      }
    }
    return getGetClaimMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.TransparencyExchangeServiceOuterClass.ListClaimsRequest,
      oscal.services.v1.TransparencyExchangeServiceOuterClass.ListClaimsResponse> getListClaimsMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "ListClaims",
      requestType = oscal.services.v1.TransparencyExchangeServiceOuterClass.ListClaimsRequest.class,
      responseType = oscal.services.v1.TransparencyExchangeServiceOuterClass.ListClaimsResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.TransparencyExchangeServiceOuterClass.ListClaimsRequest,
      oscal.services.v1.TransparencyExchangeServiceOuterClass.ListClaimsResponse> getListClaimsMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.TransparencyExchangeServiceOuterClass.ListClaimsRequest, oscal.services.v1.TransparencyExchangeServiceOuterClass.ListClaimsResponse> getListClaimsMethod;
    if ((getListClaimsMethod = TransparencyExchangeServiceGrpc.getListClaimsMethod) == null) {
      synchronized (TransparencyExchangeServiceGrpc.class) {
        if ((getListClaimsMethod = TransparencyExchangeServiceGrpc.getListClaimsMethod) == null) {
          TransparencyExchangeServiceGrpc.getListClaimsMethod = getListClaimsMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.TransparencyExchangeServiceOuterClass.ListClaimsRequest, oscal.services.v1.TransparencyExchangeServiceOuterClass.ListClaimsResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "ListClaims"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyExchangeServiceOuterClass.ListClaimsRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyExchangeServiceOuterClass.ListClaimsResponse.getDefaultInstance()))
              .setSchemaDescriptor(new TransparencyExchangeServiceMethodDescriptorSupplier("ListClaims"))
              .build();
        }
      }
    }
    return getListClaimsMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyClaimRequest,
      oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyClaimResponse> getVerifyClaimMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "VerifyClaim",
      requestType = oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyClaimRequest.class,
      responseType = oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyClaimResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyClaimRequest,
      oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyClaimResponse> getVerifyClaimMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyClaimRequest, oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyClaimResponse> getVerifyClaimMethod;
    if ((getVerifyClaimMethod = TransparencyExchangeServiceGrpc.getVerifyClaimMethod) == null) {
      synchronized (TransparencyExchangeServiceGrpc.class) {
        if ((getVerifyClaimMethod = TransparencyExchangeServiceGrpc.getVerifyClaimMethod) == null) {
          TransparencyExchangeServiceGrpc.getVerifyClaimMethod = getVerifyClaimMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyClaimRequest, oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyClaimResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "VerifyClaim"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyClaimRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyClaimResponse.getDefaultInstance()))
              .setSchemaDescriptor(new TransparencyExchangeServiceMethodDescriptorSupplier("VerifyClaim"))
              .build();
        }
      }
    }
    return getVerifyClaimMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.TransparencyExchangeServiceOuterClass.UploadEvidenceRequest,
      oscal.services.v1.TransparencyExchangeServiceOuterClass.UploadEvidenceResponse> getUploadEvidenceMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "UploadEvidence",
      requestType = oscal.services.v1.TransparencyExchangeServiceOuterClass.UploadEvidenceRequest.class,
      responseType = oscal.services.v1.TransparencyExchangeServiceOuterClass.UploadEvidenceResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.TransparencyExchangeServiceOuterClass.UploadEvidenceRequest,
      oscal.services.v1.TransparencyExchangeServiceOuterClass.UploadEvidenceResponse> getUploadEvidenceMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.TransparencyExchangeServiceOuterClass.UploadEvidenceRequest, oscal.services.v1.TransparencyExchangeServiceOuterClass.UploadEvidenceResponse> getUploadEvidenceMethod;
    if ((getUploadEvidenceMethod = TransparencyExchangeServiceGrpc.getUploadEvidenceMethod) == null) {
      synchronized (TransparencyExchangeServiceGrpc.class) {
        if ((getUploadEvidenceMethod = TransparencyExchangeServiceGrpc.getUploadEvidenceMethod) == null) {
          TransparencyExchangeServiceGrpc.getUploadEvidenceMethod = getUploadEvidenceMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.TransparencyExchangeServiceOuterClass.UploadEvidenceRequest, oscal.services.v1.TransparencyExchangeServiceOuterClass.UploadEvidenceResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "UploadEvidence"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyExchangeServiceOuterClass.UploadEvidenceRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyExchangeServiceOuterClass.UploadEvidenceResponse.getDefaultInstance()))
              .setSchemaDescriptor(new TransparencyExchangeServiceMethodDescriptorSupplier("UploadEvidence"))
              .build();
        }
      }
    }
    return getUploadEvidenceMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.TransparencyExchangeServiceOuterClass.GetEvidenceRequest,
      oscal.services.v1.TransparencyExchangeServiceOuterClass.GetEvidenceResponse> getGetEvidenceMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GetEvidence",
      requestType = oscal.services.v1.TransparencyExchangeServiceOuterClass.GetEvidenceRequest.class,
      responseType = oscal.services.v1.TransparencyExchangeServiceOuterClass.GetEvidenceResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.TransparencyExchangeServiceOuterClass.GetEvidenceRequest,
      oscal.services.v1.TransparencyExchangeServiceOuterClass.GetEvidenceResponse> getGetEvidenceMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.TransparencyExchangeServiceOuterClass.GetEvidenceRequest, oscal.services.v1.TransparencyExchangeServiceOuterClass.GetEvidenceResponse> getGetEvidenceMethod;
    if ((getGetEvidenceMethod = TransparencyExchangeServiceGrpc.getGetEvidenceMethod) == null) {
      synchronized (TransparencyExchangeServiceGrpc.class) {
        if ((getGetEvidenceMethod = TransparencyExchangeServiceGrpc.getGetEvidenceMethod) == null) {
          TransparencyExchangeServiceGrpc.getGetEvidenceMethod = getGetEvidenceMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.TransparencyExchangeServiceOuterClass.GetEvidenceRequest, oscal.services.v1.TransparencyExchangeServiceOuterClass.GetEvidenceResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GetEvidence"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyExchangeServiceOuterClass.GetEvidenceRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyExchangeServiceOuterClass.GetEvidenceResponse.getDefaultInstance()))
              .setSchemaDescriptor(new TransparencyExchangeServiceMethodDescriptorSupplier("GetEvidence"))
              .build();
        }
      }
    }
    return getGetEvidenceMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyEvidenceRequest,
      oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyEvidenceResponse> getVerifyEvidenceMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "VerifyEvidence",
      requestType = oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyEvidenceRequest.class,
      responseType = oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyEvidenceResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyEvidenceRequest,
      oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyEvidenceResponse> getVerifyEvidenceMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyEvidenceRequest, oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyEvidenceResponse> getVerifyEvidenceMethod;
    if ((getVerifyEvidenceMethod = TransparencyExchangeServiceGrpc.getVerifyEvidenceMethod) == null) {
      synchronized (TransparencyExchangeServiceGrpc.class) {
        if ((getVerifyEvidenceMethod = TransparencyExchangeServiceGrpc.getVerifyEvidenceMethod) == null) {
          TransparencyExchangeServiceGrpc.getVerifyEvidenceMethod = getVerifyEvidenceMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyEvidenceRequest, oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyEvidenceResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "VerifyEvidence"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyEvidenceRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyEvidenceResponse.getDefaultInstance()))
              .setSchemaDescriptor(new TransparencyExchangeServiceMethodDescriptorSupplier("VerifyEvidence"))
              .build();
        }
      }
    }
    return getVerifyEvidenceMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.TransparencyExchangeServiceOuterClass.SyncClaimsRequest,
      oscal.services.v1.TransparencyExchangeServiceOuterClass.SyncClaimsResponse> getSyncClaimsMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "SyncClaims",
      requestType = oscal.services.v1.TransparencyExchangeServiceOuterClass.SyncClaimsRequest.class,
      responseType = oscal.services.v1.TransparencyExchangeServiceOuterClass.SyncClaimsResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.TransparencyExchangeServiceOuterClass.SyncClaimsRequest,
      oscal.services.v1.TransparencyExchangeServiceOuterClass.SyncClaimsResponse> getSyncClaimsMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.TransparencyExchangeServiceOuterClass.SyncClaimsRequest, oscal.services.v1.TransparencyExchangeServiceOuterClass.SyncClaimsResponse> getSyncClaimsMethod;
    if ((getSyncClaimsMethod = TransparencyExchangeServiceGrpc.getSyncClaimsMethod) == null) {
      synchronized (TransparencyExchangeServiceGrpc.class) {
        if ((getSyncClaimsMethod = TransparencyExchangeServiceGrpc.getSyncClaimsMethod) == null) {
          TransparencyExchangeServiceGrpc.getSyncClaimsMethod = getSyncClaimsMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.TransparencyExchangeServiceOuterClass.SyncClaimsRequest, oscal.services.v1.TransparencyExchangeServiceOuterClass.SyncClaimsResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "SyncClaims"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyExchangeServiceOuterClass.SyncClaimsRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyExchangeServiceOuterClass.SyncClaimsResponse.getDefaultInstance()))
              .setSchemaDescriptor(new TransparencyExchangeServiceMethodDescriptorSupplier("SyncClaims"))
              .build();
        }
      }
    }
    return getSyncClaimsMethod;
  }

  /**
   * Creates a new async stub that supports all call types for the service
   */
  public static TransparencyExchangeServiceStub newStub(io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<TransparencyExchangeServiceStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<TransparencyExchangeServiceStub>() {
        @java.lang.Override
        public TransparencyExchangeServiceStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new TransparencyExchangeServiceStub(channel, callOptions);
        }
      };
    return TransparencyExchangeServiceStub.newStub(factory, channel);
  }

  /**
   * Creates a new blocking-style stub that supports all types of calls on the service
   */
  public static TransparencyExchangeServiceBlockingV2Stub newBlockingV2Stub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<TransparencyExchangeServiceBlockingV2Stub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<TransparencyExchangeServiceBlockingV2Stub>() {
        @java.lang.Override
        public TransparencyExchangeServiceBlockingV2Stub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new TransparencyExchangeServiceBlockingV2Stub(channel, callOptions);
        }
      };
    return TransparencyExchangeServiceBlockingV2Stub.newStub(factory, channel);
  }

  /**
   * Creates a new blocking-style stub that supports unary and streaming output calls on the service
   */
  public static TransparencyExchangeServiceBlockingStub newBlockingStub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<TransparencyExchangeServiceBlockingStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<TransparencyExchangeServiceBlockingStub>() {
        @java.lang.Override
        public TransparencyExchangeServiceBlockingStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new TransparencyExchangeServiceBlockingStub(channel, callOptions);
        }
      };
    return TransparencyExchangeServiceBlockingStub.newStub(factory, channel);
  }

  /**
   * Creates a new ListenableFuture-style stub that supports unary calls on the service
   */
  public static TransparencyExchangeServiceFutureStub newFutureStub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<TransparencyExchangeServiceFutureStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<TransparencyExchangeServiceFutureStub>() {
        @java.lang.Override
        public TransparencyExchangeServiceFutureStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new TransparencyExchangeServiceFutureStub(channel, callOptions);
        }
      };
    return TransparencyExchangeServiceFutureStub.newStub(factory, channel);
  }

  /**
   * <pre>
   * TransparencyExchangeService provides content-addressed attestation claims
   * and xBOM evidence exchange for xOSCAL.
   * </pre>
   */
  public interface AsyncService {

    /**
     */
    default void createClaim(oscal.services.v1.TransparencyExchangeServiceOuterClass.CreateClaimRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyExchangeServiceOuterClass.CreateClaimResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getCreateClaimMethod(), responseObserver);
    }

    /**
     */
    default void getClaim(oscal.services.v1.TransparencyExchangeServiceOuterClass.GetClaimRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyExchangeServiceOuterClass.GetClaimResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getGetClaimMethod(), responseObserver);
    }

    /**
     */
    default void listClaims(oscal.services.v1.TransparencyExchangeServiceOuterClass.ListClaimsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyExchangeServiceOuterClass.ListClaimsResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getListClaimsMethod(), responseObserver);
    }

    /**
     */
    default void verifyClaim(oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyClaimRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyClaimResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getVerifyClaimMethod(), responseObserver);
    }

    /**
     */
    default void uploadEvidence(oscal.services.v1.TransparencyExchangeServiceOuterClass.UploadEvidenceRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyExchangeServiceOuterClass.UploadEvidenceResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getUploadEvidenceMethod(), responseObserver);
    }

    /**
     */
    default void getEvidence(oscal.services.v1.TransparencyExchangeServiceOuterClass.GetEvidenceRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyExchangeServiceOuterClass.GetEvidenceResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getGetEvidenceMethod(), responseObserver);
    }

    /**
     */
    default void verifyEvidence(oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyEvidenceRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyEvidenceResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getVerifyEvidenceMethod(), responseObserver);
    }

    /**
     */
    default void syncClaims(oscal.services.v1.TransparencyExchangeServiceOuterClass.SyncClaimsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyExchangeServiceOuterClass.SyncClaimsResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getSyncClaimsMethod(), responseObserver);
    }
  }

  /**
   * Base class for the server implementation of the service TransparencyExchangeService.
   * <pre>
   * TransparencyExchangeService provides content-addressed attestation claims
   * and xBOM evidence exchange for xOSCAL.
   * </pre>
   */
  public static abstract class TransparencyExchangeServiceImplBase
      implements io.grpc.BindableService, AsyncService {

    @java.lang.Override public final io.grpc.ServerServiceDefinition bindService() {
      return TransparencyExchangeServiceGrpc.bindService(this);
    }
  }

  /**
   * A stub to allow clients to do asynchronous rpc calls to service TransparencyExchangeService.
   * <pre>
   * TransparencyExchangeService provides content-addressed attestation claims
   * and xBOM evidence exchange for xOSCAL.
   * </pre>
   */
  public static final class TransparencyExchangeServiceStub
      extends io.grpc.stub.AbstractAsyncStub<TransparencyExchangeServiceStub> {
    private TransparencyExchangeServiceStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected TransparencyExchangeServiceStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new TransparencyExchangeServiceStub(channel, callOptions);
    }

    /**
     */
    public void createClaim(oscal.services.v1.TransparencyExchangeServiceOuterClass.CreateClaimRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyExchangeServiceOuterClass.CreateClaimResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getCreateClaimMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void getClaim(oscal.services.v1.TransparencyExchangeServiceOuterClass.GetClaimRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyExchangeServiceOuterClass.GetClaimResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getGetClaimMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void listClaims(oscal.services.v1.TransparencyExchangeServiceOuterClass.ListClaimsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyExchangeServiceOuterClass.ListClaimsResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getListClaimsMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void verifyClaim(oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyClaimRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyClaimResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getVerifyClaimMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void uploadEvidence(oscal.services.v1.TransparencyExchangeServiceOuterClass.UploadEvidenceRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyExchangeServiceOuterClass.UploadEvidenceResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getUploadEvidenceMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void getEvidence(oscal.services.v1.TransparencyExchangeServiceOuterClass.GetEvidenceRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyExchangeServiceOuterClass.GetEvidenceResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getGetEvidenceMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void verifyEvidence(oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyEvidenceRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyEvidenceResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getVerifyEvidenceMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void syncClaims(oscal.services.v1.TransparencyExchangeServiceOuterClass.SyncClaimsRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyExchangeServiceOuterClass.SyncClaimsResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getSyncClaimsMethod(), getCallOptions()), request, responseObserver);
    }
  }

  /**
   * A stub to allow clients to do synchronous rpc calls to service TransparencyExchangeService.
   * <pre>
   * TransparencyExchangeService provides content-addressed attestation claims
   * and xBOM evidence exchange for xOSCAL.
   * </pre>
   */
  public static final class TransparencyExchangeServiceBlockingV2Stub
      extends io.grpc.stub.AbstractBlockingStub<TransparencyExchangeServiceBlockingV2Stub> {
    private TransparencyExchangeServiceBlockingV2Stub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected TransparencyExchangeServiceBlockingV2Stub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new TransparencyExchangeServiceBlockingV2Stub(channel, callOptions);
    }

    /**
     */
    public oscal.services.v1.TransparencyExchangeServiceOuterClass.CreateClaimResponse createClaim(oscal.services.v1.TransparencyExchangeServiceOuterClass.CreateClaimRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getCreateClaimMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyExchangeServiceOuterClass.GetClaimResponse getClaim(oscal.services.v1.TransparencyExchangeServiceOuterClass.GetClaimRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getGetClaimMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyExchangeServiceOuterClass.ListClaimsResponse listClaims(oscal.services.v1.TransparencyExchangeServiceOuterClass.ListClaimsRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getListClaimsMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyClaimResponse verifyClaim(oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyClaimRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getVerifyClaimMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyExchangeServiceOuterClass.UploadEvidenceResponse uploadEvidence(oscal.services.v1.TransparencyExchangeServiceOuterClass.UploadEvidenceRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getUploadEvidenceMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyExchangeServiceOuterClass.GetEvidenceResponse getEvidence(oscal.services.v1.TransparencyExchangeServiceOuterClass.GetEvidenceRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getGetEvidenceMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyEvidenceResponse verifyEvidence(oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyEvidenceRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getVerifyEvidenceMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyExchangeServiceOuterClass.SyncClaimsResponse syncClaims(oscal.services.v1.TransparencyExchangeServiceOuterClass.SyncClaimsRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getSyncClaimsMethod(), getCallOptions(), request);
    }
  }

  /**
   * A stub to allow clients to do limited synchronous rpc calls to service TransparencyExchangeService.
   * <pre>
   * TransparencyExchangeService provides content-addressed attestation claims
   * and xBOM evidence exchange for xOSCAL.
   * </pre>
   */
  public static final class TransparencyExchangeServiceBlockingStub
      extends io.grpc.stub.AbstractBlockingStub<TransparencyExchangeServiceBlockingStub> {
    private TransparencyExchangeServiceBlockingStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected TransparencyExchangeServiceBlockingStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new TransparencyExchangeServiceBlockingStub(channel, callOptions);
    }

    /**
     */
    public oscal.services.v1.TransparencyExchangeServiceOuterClass.CreateClaimResponse createClaim(oscal.services.v1.TransparencyExchangeServiceOuterClass.CreateClaimRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getCreateClaimMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyExchangeServiceOuterClass.GetClaimResponse getClaim(oscal.services.v1.TransparencyExchangeServiceOuterClass.GetClaimRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getGetClaimMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyExchangeServiceOuterClass.ListClaimsResponse listClaims(oscal.services.v1.TransparencyExchangeServiceOuterClass.ListClaimsRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getListClaimsMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyClaimResponse verifyClaim(oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyClaimRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getVerifyClaimMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyExchangeServiceOuterClass.UploadEvidenceResponse uploadEvidence(oscal.services.v1.TransparencyExchangeServiceOuterClass.UploadEvidenceRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getUploadEvidenceMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyExchangeServiceOuterClass.GetEvidenceResponse getEvidence(oscal.services.v1.TransparencyExchangeServiceOuterClass.GetEvidenceRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getGetEvidenceMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyEvidenceResponse verifyEvidence(oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyEvidenceRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getVerifyEvidenceMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyExchangeServiceOuterClass.SyncClaimsResponse syncClaims(oscal.services.v1.TransparencyExchangeServiceOuterClass.SyncClaimsRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getSyncClaimsMethod(), getCallOptions(), request);
    }
  }

  /**
   * A stub to allow clients to do ListenableFuture-style rpc calls to service TransparencyExchangeService.
   * <pre>
   * TransparencyExchangeService provides content-addressed attestation claims
   * and xBOM evidence exchange for xOSCAL.
   * </pre>
   */
  public static final class TransparencyExchangeServiceFutureStub
      extends io.grpc.stub.AbstractFutureStub<TransparencyExchangeServiceFutureStub> {
    private TransparencyExchangeServiceFutureStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected TransparencyExchangeServiceFutureStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new TransparencyExchangeServiceFutureStub(channel, callOptions);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.TransparencyExchangeServiceOuterClass.CreateClaimResponse> createClaim(
        oscal.services.v1.TransparencyExchangeServiceOuterClass.CreateClaimRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getCreateClaimMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.TransparencyExchangeServiceOuterClass.GetClaimResponse> getClaim(
        oscal.services.v1.TransparencyExchangeServiceOuterClass.GetClaimRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getGetClaimMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.TransparencyExchangeServiceOuterClass.ListClaimsResponse> listClaims(
        oscal.services.v1.TransparencyExchangeServiceOuterClass.ListClaimsRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getListClaimsMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyClaimResponse> verifyClaim(
        oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyClaimRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getVerifyClaimMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.TransparencyExchangeServiceOuterClass.UploadEvidenceResponse> uploadEvidence(
        oscal.services.v1.TransparencyExchangeServiceOuterClass.UploadEvidenceRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getUploadEvidenceMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.TransparencyExchangeServiceOuterClass.GetEvidenceResponse> getEvidence(
        oscal.services.v1.TransparencyExchangeServiceOuterClass.GetEvidenceRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getGetEvidenceMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyEvidenceResponse> verifyEvidence(
        oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyEvidenceRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getVerifyEvidenceMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.TransparencyExchangeServiceOuterClass.SyncClaimsResponse> syncClaims(
        oscal.services.v1.TransparencyExchangeServiceOuterClass.SyncClaimsRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getSyncClaimsMethod(), getCallOptions()), request);
    }
  }

  private static final int METHODID_CREATE_CLAIM = 0;
  private static final int METHODID_GET_CLAIM = 1;
  private static final int METHODID_LIST_CLAIMS = 2;
  private static final int METHODID_VERIFY_CLAIM = 3;
  private static final int METHODID_UPLOAD_EVIDENCE = 4;
  private static final int METHODID_GET_EVIDENCE = 5;
  private static final int METHODID_VERIFY_EVIDENCE = 6;
  private static final int METHODID_SYNC_CLAIMS = 7;

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
        case METHODID_CREATE_CLAIM:
          serviceImpl.createClaim((oscal.services.v1.TransparencyExchangeServiceOuterClass.CreateClaimRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyExchangeServiceOuterClass.CreateClaimResponse>) responseObserver);
          break;
        case METHODID_GET_CLAIM:
          serviceImpl.getClaim((oscal.services.v1.TransparencyExchangeServiceOuterClass.GetClaimRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyExchangeServiceOuterClass.GetClaimResponse>) responseObserver);
          break;
        case METHODID_LIST_CLAIMS:
          serviceImpl.listClaims((oscal.services.v1.TransparencyExchangeServiceOuterClass.ListClaimsRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyExchangeServiceOuterClass.ListClaimsResponse>) responseObserver);
          break;
        case METHODID_VERIFY_CLAIM:
          serviceImpl.verifyClaim((oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyClaimRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyClaimResponse>) responseObserver);
          break;
        case METHODID_UPLOAD_EVIDENCE:
          serviceImpl.uploadEvidence((oscal.services.v1.TransparencyExchangeServiceOuterClass.UploadEvidenceRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyExchangeServiceOuterClass.UploadEvidenceResponse>) responseObserver);
          break;
        case METHODID_GET_EVIDENCE:
          serviceImpl.getEvidence((oscal.services.v1.TransparencyExchangeServiceOuterClass.GetEvidenceRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyExchangeServiceOuterClass.GetEvidenceResponse>) responseObserver);
          break;
        case METHODID_VERIFY_EVIDENCE:
          serviceImpl.verifyEvidence((oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyEvidenceRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyEvidenceResponse>) responseObserver);
          break;
        case METHODID_SYNC_CLAIMS:
          serviceImpl.syncClaims((oscal.services.v1.TransparencyExchangeServiceOuterClass.SyncClaimsRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyExchangeServiceOuterClass.SyncClaimsResponse>) responseObserver);
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
          getCreateClaimMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.TransparencyExchangeServiceOuterClass.CreateClaimRequest,
              oscal.services.v1.TransparencyExchangeServiceOuterClass.CreateClaimResponse>(
                service, METHODID_CREATE_CLAIM)))
        .addMethod(
          getGetClaimMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.TransparencyExchangeServiceOuterClass.GetClaimRequest,
              oscal.services.v1.TransparencyExchangeServiceOuterClass.GetClaimResponse>(
                service, METHODID_GET_CLAIM)))
        .addMethod(
          getListClaimsMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.TransparencyExchangeServiceOuterClass.ListClaimsRequest,
              oscal.services.v1.TransparencyExchangeServiceOuterClass.ListClaimsResponse>(
                service, METHODID_LIST_CLAIMS)))
        .addMethod(
          getVerifyClaimMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyClaimRequest,
              oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyClaimResponse>(
                service, METHODID_VERIFY_CLAIM)))
        .addMethod(
          getUploadEvidenceMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.TransparencyExchangeServiceOuterClass.UploadEvidenceRequest,
              oscal.services.v1.TransparencyExchangeServiceOuterClass.UploadEvidenceResponse>(
                service, METHODID_UPLOAD_EVIDENCE)))
        .addMethod(
          getGetEvidenceMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.TransparencyExchangeServiceOuterClass.GetEvidenceRequest,
              oscal.services.v1.TransparencyExchangeServiceOuterClass.GetEvidenceResponse>(
                service, METHODID_GET_EVIDENCE)))
        .addMethod(
          getVerifyEvidenceMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyEvidenceRequest,
              oscal.services.v1.TransparencyExchangeServiceOuterClass.VerifyEvidenceResponse>(
                service, METHODID_VERIFY_EVIDENCE)))
        .addMethod(
          getSyncClaimsMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.TransparencyExchangeServiceOuterClass.SyncClaimsRequest,
              oscal.services.v1.TransparencyExchangeServiceOuterClass.SyncClaimsResponse>(
                service, METHODID_SYNC_CLAIMS)))
        .build();
  }

  private static abstract class TransparencyExchangeServiceBaseDescriptorSupplier
      implements io.grpc.protobuf.ProtoFileDescriptorSupplier, io.grpc.protobuf.ProtoServiceDescriptorSupplier {
    TransparencyExchangeServiceBaseDescriptorSupplier() {}

    @java.lang.Override
    public com.google.protobuf.Descriptors.FileDescriptor getFileDescriptor() {
      return oscal.services.v1.TransparencyExchangeServiceOuterClass.getDescriptor();
    }

    @java.lang.Override
    public com.google.protobuf.Descriptors.ServiceDescriptor getServiceDescriptor() {
      return getFileDescriptor().findServiceByName("TransparencyExchangeService");
    }
  }

  private static final class TransparencyExchangeServiceFileDescriptorSupplier
      extends TransparencyExchangeServiceBaseDescriptorSupplier {
    TransparencyExchangeServiceFileDescriptorSupplier() {}
  }

  private static final class TransparencyExchangeServiceMethodDescriptorSupplier
      extends TransparencyExchangeServiceBaseDescriptorSupplier
      implements io.grpc.protobuf.ProtoMethodDescriptorSupplier {
    private final java.lang.String methodName;

    TransparencyExchangeServiceMethodDescriptorSupplier(java.lang.String methodName) {
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
      synchronized (TransparencyExchangeServiceGrpc.class) {
        result = serviceDescriptor;
        if (result == null) {
          serviceDescriptor = result = io.grpc.ServiceDescriptor.newBuilder(SERVICE_NAME)
              .setSchemaDescriptor(new TransparencyExchangeServiceFileDescriptorSupplier())
              .addMethod(getCreateClaimMethod())
              .addMethod(getGetClaimMethod())
              .addMethod(getListClaimsMethod())
              .addMethod(getVerifyClaimMethod())
              .addMethod(getUploadEvidenceMethod())
              .addMethod(getGetEvidenceMethod())
              .addMethod(getVerifyEvidenceMethod())
              .addMethod(getSyncClaimsMethod())
              .build();
        }
      }
    }
    return result;
  }
}
