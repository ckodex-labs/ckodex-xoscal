package oscal.services.v1;

import static io.grpc.MethodDescriptor.generateFullMethodName;

/**
 * <pre>
 * TransparencyGraphService provides graph traversal over projected claims
 * with structured proof-state verification for xOSCAL.
 * </pre>
 */
@io.grpc.stub.annotations.GrpcGenerated
public final class TransparencyGraphServiceGrpc {

  private TransparencyGraphServiceGrpc() {}

  public static final java.lang.String SERVICE_NAME = "oscal.services.v1.TransparencyGraphService";

  // Static method descriptors that strictly reflect the proto.
  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.ProjectEdgeRequest,
      oscal.services.v1.TransparencyGraphServiceOuterClass.ProjectEdgeResponse> getProjectEdgeMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "ProjectEdge",
      requestType = oscal.services.v1.TransparencyGraphServiceOuterClass.ProjectEdgeRequest.class,
      responseType = oscal.services.v1.TransparencyGraphServiceOuterClass.ProjectEdgeResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.ProjectEdgeRequest,
      oscal.services.v1.TransparencyGraphServiceOuterClass.ProjectEdgeResponse> getProjectEdgeMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.ProjectEdgeRequest, oscal.services.v1.TransparencyGraphServiceOuterClass.ProjectEdgeResponse> getProjectEdgeMethod;
    if ((getProjectEdgeMethod = TransparencyGraphServiceGrpc.getProjectEdgeMethod) == null) {
      synchronized (TransparencyGraphServiceGrpc.class) {
        if ((getProjectEdgeMethod = TransparencyGraphServiceGrpc.getProjectEdgeMethod) == null) {
          TransparencyGraphServiceGrpc.getProjectEdgeMethod = getProjectEdgeMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.TransparencyGraphServiceOuterClass.ProjectEdgeRequest, oscal.services.v1.TransparencyGraphServiceOuterClass.ProjectEdgeResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "ProjectEdge"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyGraphServiceOuterClass.ProjectEdgeRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyGraphServiceOuterClass.ProjectEdgeResponse.getDefaultInstance()))
              .setSchemaDescriptor(new TransparencyGraphServiceMethodDescriptorSupplier("ProjectEdge"))
              .build();
        }
      }
    }
    return getProjectEdgeMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.GetEdgeRequest,
      oscal.services.v1.TransparencyGraphServiceOuterClass.GetEdgeResponse> getGetEdgeMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GetEdge",
      requestType = oscal.services.v1.TransparencyGraphServiceOuterClass.GetEdgeRequest.class,
      responseType = oscal.services.v1.TransparencyGraphServiceOuterClass.GetEdgeResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.GetEdgeRequest,
      oscal.services.v1.TransparencyGraphServiceOuterClass.GetEdgeResponse> getGetEdgeMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.GetEdgeRequest, oscal.services.v1.TransparencyGraphServiceOuterClass.GetEdgeResponse> getGetEdgeMethod;
    if ((getGetEdgeMethod = TransparencyGraphServiceGrpc.getGetEdgeMethod) == null) {
      synchronized (TransparencyGraphServiceGrpc.class) {
        if ((getGetEdgeMethod = TransparencyGraphServiceGrpc.getGetEdgeMethod) == null) {
          TransparencyGraphServiceGrpc.getGetEdgeMethod = getGetEdgeMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.TransparencyGraphServiceOuterClass.GetEdgeRequest, oscal.services.v1.TransparencyGraphServiceOuterClass.GetEdgeResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GetEdge"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyGraphServiceOuterClass.GetEdgeRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyGraphServiceOuterClass.GetEdgeResponse.getDefaultInstance()))
              .setSchemaDescriptor(new TransparencyGraphServiceMethodDescriptorSupplier("GetEdge"))
              .build();
        }
      }
    }
    return getGetEdgeMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.ListEdgesRequest,
      oscal.services.v1.TransparencyGraphServiceOuterClass.ListEdgesResponse> getListEdgesMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "ListEdges",
      requestType = oscal.services.v1.TransparencyGraphServiceOuterClass.ListEdgesRequest.class,
      responseType = oscal.services.v1.TransparencyGraphServiceOuterClass.ListEdgesResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.ListEdgesRequest,
      oscal.services.v1.TransparencyGraphServiceOuterClass.ListEdgesResponse> getListEdgesMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.ListEdgesRequest, oscal.services.v1.TransparencyGraphServiceOuterClass.ListEdgesResponse> getListEdgesMethod;
    if ((getListEdgesMethod = TransparencyGraphServiceGrpc.getListEdgesMethod) == null) {
      synchronized (TransparencyGraphServiceGrpc.class) {
        if ((getListEdgesMethod = TransparencyGraphServiceGrpc.getListEdgesMethod) == null) {
          TransparencyGraphServiceGrpc.getListEdgesMethod = getListEdgesMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.TransparencyGraphServiceOuterClass.ListEdgesRequest, oscal.services.v1.TransparencyGraphServiceOuterClass.ListEdgesResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "ListEdges"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyGraphServiceOuterClass.ListEdgesRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyGraphServiceOuterClass.ListEdgesResponse.getDefaultInstance()))
              .setSchemaDescriptor(new TransparencyGraphServiceMethodDescriptorSupplier("ListEdges"))
              .build();
        }
      }
    }
    return getListEdgesMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.DeleteEdgeRequest,
      oscal.services.v1.TransparencyGraphServiceOuterClass.DeleteEdgeResponse> getDeleteEdgeMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "DeleteEdge",
      requestType = oscal.services.v1.TransparencyGraphServiceOuterClass.DeleteEdgeRequest.class,
      responseType = oscal.services.v1.TransparencyGraphServiceOuterClass.DeleteEdgeResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.DeleteEdgeRequest,
      oscal.services.v1.TransparencyGraphServiceOuterClass.DeleteEdgeResponse> getDeleteEdgeMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.DeleteEdgeRequest, oscal.services.v1.TransparencyGraphServiceOuterClass.DeleteEdgeResponse> getDeleteEdgeMethod;
    if ((getDeleteEdgeMethod = TransparencyGraphServiceGrpc.getDeleteEdgeMethod) == null) {
      synchronized (TransparencyGraphServiceGrpc.class) {
        if ((getDeleteEdgeMethod = TransparencyGraphServiceGrpc.getDeleteEdgeMethod) == null) {
          TransparencyGraphServiceGrpc.getDeleteEdgeMethod = getDeleteEdgeMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.TransparencyGraphServiceOuterClass.DeleteEdgeRequest, oscal.services.v1.TransparencyGraphServiceOuterClass.DeleteEdgeResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "DeleteEdge"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyGraphServiceOuterClass.DeleteEdgeRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyGraphServiceOuterClass.DeleteEdgeResponse.getDefaultInstance()))
              .setSchemaDescriptor(new TransparencyGraphServiceMethodDescriptorSupplier("DeleteEdge"))
              .build();
        }
      }
    }
    return getDeleteEdgeMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.GetNodeRequest,
      oscal.services.v1.TransparencyGraphServiceOuterClass.GetNodeResponse> getGetNodeMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GetNode",
      requestType = oscal.services.v1.TransparencyGraphServiceOuterClass.GetNodeRequest.class,
      responseType = oscal.services.v1.TransparencyGraphServiceOuterClass.GetNodeResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.GetNodeRequest,
      oscal.services.v1.TransparencyGraphServiceOuterClass.GetNodeResponse> getGetNodeMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.GetNodeRequest, oscal.services.v1.TransparencyGraphServiceOuterClass.GetNodeResponse> getGetNodeMethod;
    if ((getGetNodeMethod = TransparencyGraphServiceGrpc.getGetNodeMethod) == null) {
      synchronized (TransparencyGraphServiceGrpc.class) {
        if ((getGetNodeMethod = TransparencyGraphServiceGrpc.getGetNodeMethod) == null) {
          TransparencyGraphServiceGrpc.getGetNodeMethod = getGetNodeMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.TransparencyGraphServiceOuterClass.GetNodeRequest, oscal.services.v1.TransparencyGraphServiceOuterClass.GetNodeResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GetNode"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyGraphServiceOuterClass.GetNodeRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyGraphServiceOuterClass.GetNodeResponse.getDefaultInstance()))
              .setSchemaDescriptor(new TransparencyGraphServiceMethodDescriptorSupplier("GetNode"))
              .build();
        }
      }
    }
    return getGetNodeMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.ListNodesRequest,
      oscal.services.v1.TransparencyGraphServiceOuterClass.ListNodesResponse> getListNodesMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "ListNodes",
      requestType = oscal.services.v1.TransparencyGraphServiceOuterClass.ListNodesRequest.class,
      responseType = oscal.services.v1.TransparencyGraphServiceOuterClass.ListNodesResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.ListNodesRequest,
      oscal.services.v1.TransparencyGraphServiceOuterClass.ListNodesResponse> getListNodesMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.ListNodesRequest, oscal.services.v1.TransparencyGraphServiceOuterClass.ListNodesResponse> getListNodesMethod;
    if ((getListNodesMethod = TransparencyGraphServiceGrpc.getListNodesMethod) == null) {
      synchronized (TransparencyGraphServiceGrpc.class) {
        if ((getListNodesMethod = TransparencyGraphServiceGrpc.getListNodesMethod) == null) {
          TransparencyGraphServiceGrpc.getListNodesMethod = getListNodesMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.TransparencyGraphServiceOuterClass.ListNodesRequest, oscal.services.v1.TransparencyGraphServiceOuterClass.ListNodesResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "ListNodes"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyGraphServiceOuterClass.ListNodesRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyGraphServiceOuterClass.ListNodesResponse.getDefaultInstance()))
              .setSchemaDescriptor(new TransparencyGraphServiceMethodDescriptorSupplier("ListNodes"))
              .build();
        }
      }
    }
    return getListNodesMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.TraverseRequest,
      oscal.services.v1.TransparencyGraphServiceOuterClass.TraverseResponse> getTraverseMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "Traverse",
      requestType = oscal.services.v1.TransparencyGraphServiceOuterClass.TraverseRequest.class,
      responseType = oscal.services.v1.TransparencyGraphServiceOuterClass.TraverseResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.TraverseRequest,
      oscal.services.v1.TransparencyGraphServiceOuterClass.TraverseResponse> getTraverseMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.TraverseRequest, oscal.services.v1.TransparencyGraphServiceOuterClass.TraverseResponse> getTraverseMethod;
    if ((getTraverseMethod = TransparencyGraphServiceGrpc.getTraverseMethod) == null) {
      synchronized (TransparencyGraphServiceGrpc.class) {
        if ((getTraverseMethod = TransparencyGraphServiceGrpc.getTraverseMethod) == null) {
          TransparencyGraphServiceGrpc.getTraverseMethod = getTraverseMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.TransparencyGraphServiceOuterClass.TraverseRequest, oscal.services.v1.TransparencyGraphServiceOuterClass.TraverseResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "Traverse"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyGraphServiceOuterClass.TraverseRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyGraphServiceOuterClass.TraverseResponse.getDefaultInstance()))
              .setSchemaDescriptor(new TransparencyGraphServiceMethodDescriptorSupplier("Traverse"))
              .build();
        }
      }
    }
    return getTraverseMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.ShortestPathRequest,
      oscal.services.v1.TransparencyGraphServiceOuterClass.ShortestPathResponse> getShortestPathMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "ShortestPath",
      requestType = oscal.services.v1.TransparencyGraphServiceOuterClass.ShortestPathRequest.class,
      responseType = oscal.services.v1.TransparencyGraphServiceOuterClass.ShortestPathResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.ShortestPathRequest,
      oscal.services.v1.TransparencyGraphServiceOuterClass.ShortestPathResponse> getShortestPathMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.ShortestPathRequest, oscal.services.v1.TransparencyGraphServiceOuterClass.ShortestPathResponse> getShortestPathMethod;
    if ((getShortestPathMethod = TransparencyGraphServiceGrpc.getShortestPathMethod) == null) {
      synchronized (TransparencyGraphServiceGrpc.class) {
        if ((getShortestPathMethod = TransparencyGraphServiceGrpc.getShortestPathMethod) == null) {
          TransparencyGraphServiceGrpc.getShortestPathMethod = getShortestPathMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.TransparencyGraphServiceOuterClass.ShortestPathRequest, oscal.services.v1.TransparencyGraphServiceOuterClass.ShortestPathResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "ShortestPath"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyGraphServiceOuterClass.ShortestPathRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyGraphServiceOuterClass.ShortestPathResponse.getDefaultInstance()))
              .setSchemaDescriptor(new TransparencyGraphServiceMethodDescriptorSupplier("ShortestPath"))
              .build();
        }
      }
    }
    return getShortestPathMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.ImpactRadiusRequest,
      oscal.services.v1.TransparencyGraphServiceOuterClass.ImpactRadiusResponse> getImpactRadiusMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "ImpactRadius",
      requestType = oscal.services.v1.TransparencyGraphServiceOuterClass.ImpactRadiusRequest.class,
      responseType = oscal.services.v1.TransparencyGraphServiceOuterClass.ImpactRadiusResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.ImpactRadiusRequest,
      oscal.services.v1.TransparencyGraphServiceOuterClass.ImpactRadiusResponse> getImpactRadiusMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.ImpactRadiusRequest, oscal.services.v1.TransparencyGraphServiceOuterClass.ImpactRadiusResponse> getImpactRadiusMethod;
    if ((getImpactRadiusMethod = TransparencyGraphServiceGrpc.getImpactRadiusMethod) == null) {
      synchronized (TransparencyGraphServiceGrpc.class) {
        if ((getImpactRadiusMethod = TransparencyGraphServiceGrpc.getImpactRadiusMethod) == null) {
          TransparencyGraphServiceGrpc.getImpactRadiusMethod = getImpactRadiusMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.TransparencyGraphServiceOuterClass.ImpactRadiusRequest, oscal.services.v1.TransparencyGraphServiceOuterClass.ImpactRadiusResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "ImpactRadius"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyGraphServiceOuterClass.ImpactRadiusRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyGraphServiceOuterClass.ImpactRadiusResponse.getDefaultInstance()))
              .setSchemaDescriptor(new TransparencyGraphServiceMethodDescriptorSupplier("ImpactRadius"))
              .build();
        }
      }
    }
    return getImpactRadiusMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.ExplainClaimRequest,
      oscal.services.v1.TransparencyGraphServiceOuterClass.ExplainClaimResponse> getExplainClaimMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "ExplainClaim",
      requestType = oscal.services.v1.TransparencyGraphServiceOuterClass.ExplainClaimRequest.class,
      responseType = oscal.services.v1.TransparencyGraphServiceOuterClass.ExplainClaimResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.ExplainClaimRequest,
      oscal.services.v1.TransparencyGraphServiceOuterClass.ExplainClaimResponse> getExplainClaimMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.ExplainClaimRequest, oscal.services.v1.TransparencyGraphServiceOuterClass.ExplainClaimResponse> getExplainClaimMethod;
    if ((getExplainClaimMethod = TransparencyGraphServiceGrpc.getExplainClaimMethod) == null) {
      synchronized (TransparencyGraphServiceGrpc.class) {
        if ((getExplainClaimMethod = TransparencyGraphServiceGrpc.getExplainClaimMethod) == null) {
          TransparencyGraphServiceGrpc.getExplainClaimMethod = getExplainClaimMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.TransparencyGraphServiceOuterClass.ExplainClaimRequest, oscal.services.v1.TransparencyGraphServiceOuterClass.ExplainClaimResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "ExplainClaim"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyGraphServiceOuterClass.ExplainClaimRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyGraphServiceOuterClass.ExplainClaimResponse.getDefaultInstance()))
              .setSchemaDescriptor(new TransparencyGraphServiceMethodDescriptorSupplier("ExplainClaim"))
              .build();
        }
      }
    }
    return getExplainClaimMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.ComputeTrustStateRequest,
      oscal.services.v1.TransparencyGraphServiceOuterClass.ComputeTrustStateResponse> getComputeTrustStateMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "ComputeTrustState",
      requestType = oscal.services.v1.TransparencyGraphServiceOuterClass.ComputeTrustStateRequest.class,
      responseType = oscal.services.v1.TransparencyGraphServiceOuterClass.ComputeTrustStateResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.ComputeTrustStateRequest,
      oscal.services.v1.TransparencyGraphServiceOuterClass.ComputeTrustStateResponse> getComputeTrustStateMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.ComputeTrustStateRequest, oscal.services.v1.TransparencyGraphServiceOuterClass.ComputeTrustStateResponse> getComputeTrustStateMethod;
    if ((getComputeTrustStateMethod = TransparencyGraphServiceGrpc.getComputeTrustStateMethod) == null) {
      synchronized (TransparencyGraphServiceGrpc.class) {
        if ((getComputeTrustStateMethod = TransparencyGraphServiceGrpc.getComputeTrustStateMethod) == null) {
          TransparencyGraphServiceGrpc.getComputeTrustStateMethod = getComputeTrustStateMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.TransparencyGraphServiceOuterClass.ComputeTrustStateRequest, oscal.services.v1.TransparencyGraphServiceOuterClass.ComputeTrustStateResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "ComputeTrustState"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyGraphServiceOuterClass.ComputeTrustStateRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyGraphServiceOuterClass.ComputeTrustStateResponse.getDefaultInstance()))
              .setSchemaDescriptor(new TransparencyGraphServiceMethodDescriptorSupplier("ComputeTrustState"))
              .build();
        }
      }
    }
    return getComputeTrustStateMethod;
  }

  private static volatile io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.VerifyClosureRequest,
      oscal.services.v1.TransparencyGraphServiceOuterClass.VerifyClosureResponse> getVerifyClosureMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "VerifyClosure",
      requestType = oscal.services.v1.TransparencyGraphServiceOuterClass.VerifyClosureRequest.class,
      responseType = oscal.services.v1.TransparencyGraphServiceOuterClass.VerifyClosureResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.VerifyClosureRequest,
      oscal.services.v1.TransparencyGraphServiceOuterClass.VerifyClosureResponse> getVerifyClosureMethod() {
    io.grpc.MethodDescriptor<oscal.services.v1.TransparencyGraphServiceOuterClass.VerifyClosureRequest, oscal.services.v1.TransparencyGraphServiceOuterClass.VerifyClosureResponse> getVerifyClosureMethod;
    if ((getVerifyClosureMethod = TransparencyGraphServiceGrpc.getVerifyClosureMethod) == null) {
      synchronized (TransparencyGraphServiceGrpc.class) {
        if ((getVerifyClosureMethod = TransparencyGraphServiceGrpc.getVerifyClosureMethod) == null) {
          TransparencyGraphServiceGrpc.getVerifyClosureMethod = getVerifyClosureMethod =
              io.grpc.MethodDescriptor.<oscal.services.v1.TransparencyGraphServiceOuterClass.VerifyClosureRequest, oscal.services.v1.TransparencyGraphServiceOuterClass.VerifyClosureResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "VerifyClosure"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyGraphServiceOuterClass.VerifyClosureRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  oscal.services.v1.TransparencyGraphServiceOuterClass.VerifyClosureResponse.getDefaultInstance()))
              .setSchemaDescriptor(new TransparencyGraphServiceMethodDescriptorSupplier("VerifyClosure"))
              .build();
        }
      }
    }
    return getVerifyClosureMethod;
  }

  /**
   * Creates a new async stub that supports all call types for the service
   */
  public static TransparencyGraphServiceStub newStub(io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<TransparencyGraphServiceStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<TransparencyGraphServiceStub>() {
        @java.lang.Override
        public TransparencyGraphServiceStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new TransparencyGraphServiceStub(channel, callOptions);
        }
      };
    return TransparencyGraphServiceStub.newStub(factory, channel);
  }

  /**
   * Creates a new blocking-style stub that supports all types of calls on the service
   */
  public static TransparencyGraphServiceBlockingV2Stub newBlockingV2Stub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<TransparencyGraphServiceBlockingV2Stub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<TransparencyGraphServiceBlockingV2Stub>() {
        @java.lang.Override
        public TransparencyGraphServiceBlockingV2Stub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new TransparencyGraphServiceBlockingV2Stub(channel, callOptions);
        }
      };
    return TransparencyGraphServiceBlockingV2Stub.newStub(factory, channel);
  }

  /**
   * Creates a new blocking-style stub that supports unary and streaming output calls on the service
   */
  public static TransparencyGraphServiceBlockingStub newBlockingStub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<TransparencyGraphServiceBlockingStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<TransparencyGraphServiceBlockingStub>() {
        @java.lang.Override
        public TransparencyGraphServiceBlockingStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new TransparencyGraphServiceBlockingStub(channel, callOptions);
        }
      };
    return TransparencyGraphServiceBlockingStub.newStub(factory, channel);
  }

  /**
   * Creates a new ListenableFuture-style stub that supports unary calls on the service
   */
  public static TransparencyGraphServiceFutureStub newFutureStub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<TransparencyGraphServiceFutureStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<TransparencyGraphServiceFutureStub>() {
        @java.lang.Override
        public TransparencyGraphServiceFutureStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new TransparencyGraphServiceFutureStub(channel, callOptions);
        }
      };
    return TransparencyGraphServiceFutureStub.newStub(factory, channel);
  }

  /**
   * <pre>
   * TransparencyGraphService provides graph traversal over projected claims
   * with structured proof-state verification for xOSCAL.
   * </pre>
   */
  public interface AsyncService {

    /**
     */
    default void projectEdge(oscal.services.v1.TransparencyGraphServiceOuterClass.ProjectEdgeRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.ProjectEdgeResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getProjectEdgeMethod(), responseObserver);
    }

    /**
     */
    default void getEdge(oscal.services.v1.TransparencyGraphServiceOuterClass.GetEdgeRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.GetEdgeResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getGetEdgeMethod(), responseObserver);
    }

    /**
     */
    default void listEdges(oscal.services.v1.TransparencyGraphServiceOuterClass.ListEdgesRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.ListEdgesResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getListEdgesMethod(), responseObserver);
    }

    /**
     */
    default void deleteEdge(oscal.services.v1.TransparencyGraphServiceOuterClass.DeleteEdgeRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.DeleteEdgeResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getDeleteEdgeMethod(), responseObserver);
    }

    /**
     */
    default void getNode(oscal.services.v1.TransparencyGraphServiceOuterClass.GetNodeRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.GetNodeResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getGetNodeMethod(), responseObserver);
    }

    /**
     */
    default void listNodes(oscal.services.v1.TransparencyGraphServiceOuterClass.ListNodesRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.ListNodesResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getListNodesMethod(), responseObserver);
    }

    /**
     */
    default void traverse(oscal.services.v1.TransparencyGraphServiceOuterClass.TraverseRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.TraverseResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getTraverseMethod(), responseObserver);
    }

    /**
     */
    default void shortestPath(oscal.services.v1.TransparencyGraphServiceOuterClass.ShortestPathRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.ShortestPathResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getShortestPathMethod(), responseObserver);
    }

    /**
     */
    default void impactRadius(oscal.services.v1.TransparencyGraphServiceOuterClass.ImpactRadiusRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.ImpactRadiusResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getImpactRadiusMethod(), responseObserver);
    }

    /**
     */
    default void explainClaim(oscal.services.v1.TransparencyGraphServiceOuterClass.ExplainClaimRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.ExplainClaimResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getExplainClaimMethod(), responseObserver);
    }

    /**
     */
    default void computeTrustState(oscal.services.v1.TransparencyGraphServiceOuterClass.ComputeTrustStateRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.ComputeTrustStateResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getComputeTrustStateMethod(), responseObserver);
    }

    /**
     */
    default void verifyClosure(oscal.services.v1.TransparencyGraphServiceOuterClass.VerifyClosureRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.VerifyClosureResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getVerifyClosureMethod(), responseObserver);
    }
  }

  /**
   * Base class for the server implementation of the service TransparencyGraphService.
   * <pre>
   * TransparencyGraphService provides graph traversal over projected claims
   * with structured proof-state verification for xOSCAL.
   * </pre>
   */
  public static abstract class TransparencyGraphServiceImplBase
      implements io.grpc.BindableService, AsyncService {

    @java.lang.Override public final io.grpc.ServerServiceDefinition bindService() {
      return TransparencyGraphServiceGrpc.bindService(this);
    }
  }

  /**
   * A stub to allow clients to do asynchronous rpc calls to service TransparencyGraphService.
   * <pre>
   * TransparencyGraphService provides graph traversal over projected claims
   * with structured proof-state verification for xOSCAL.
   * </pre>
   */
  public static final class TransparencyGraphServiceStub
      extends io.grpc.stub.AbstractAsyncStub<TransparencyGraphServiceStub> {
    private TransparencyGraphServiceStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected TransparencyGraphServiceStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new TransparencyGraphServiceStub(channel, callOptions);
    }

    /**
     */
    public void projectEdge(oscal.services.v1.TransparencyGraphServiceOuterClass.ProjectEdgeRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.ProjectEdgeResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getProjectEdgeMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void getEdge(oscal.services.v1.TransparencyGraphServiceOuterClass.GetEdgeRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.GetEdgeResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getGetEdgeMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void listEdges(oscal.services.v1.TransparencyGraphServiceOuterClass.ListEdgesRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.ListEdgesResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getListEdgesMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void deleteEdge(oscal.services.v1.TransparencyGraphServiceOuterClass.DeleteEdgeRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.DeleteEdgeResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getDeleteEdgeMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void getNode(oscal.services.v1.TransparencyGraphServiceOuterClass.GetNodeRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.GetNodeResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getGetNodeMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void listNodes(oscal.services.v1.TransparencyGraphServiceOuterClass.ListNodesRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.ListNodesResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getListNodesMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void traverse(oscal.services.v1.TransparencyGraphServiceOuterClass.TraverseRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.TraverseResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getTraverseMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void shortestPath(oscal.services.v1.TransparencyGraphServiceOuterClass.ShortestPathRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.ShortestPathResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getShortestPathMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void impactRadius(oscal.services.v1.TransparencyGraphServiceOuterClass.ImpactRadiusRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.ImpactRadiusResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getImpactRadiusMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void explainClaim(oscal.services.v1.TransparencyGraphServiceOuterClass.ExplainClaimRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.ExplainClaimResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getExplainClaimMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void computeTrustState(oscal.services.v1.TransparencyGraphServiceOuterClass.ComputeTrustStateRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.ComputeTrustStateResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getComputeTrustStateMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void verifyClosure(oscal.services.v1.TransparencyGraphServiceOuterClass.VerifyClosureRequest request,
        io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.VerifyClosureResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getVerifyClosureMethod(), getCallOptions()), request, responseObserver);
    }
  }

  /**
   * A stub to allow clients to do synchronous rpc calls to service TransparencyGraphService.
   * <pre>
   * TransparencyGraphService provides graph traversal over projected claims
   * with structured proof-state verification for xOSCAL.
   * </pre>
   */
  public static final class TransparencyGraphServiceBlockingV2Stub
      extends io.grpc.stub.AbstractBlockingStub<TransparencyGraphServiceBlockingV2Stub> {
    private TransparencyGraphServiceBlockingV2Stub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected TransparencyGraphServiceBlockingV2Stub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new TransparencyGraphServiceBlockingV2Stub(channel, callOptions);
    }

    /**
     */
    public oscal.services.v1.TransparencyGraphServiceOuterClass.ProjectEdgeResponse projectEdge(oscal.services.v1.TransparencyGraphServiceOuterClass.ProjectEdgeRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getProjectEdgeMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyGraphServiceOuterClass.GetEdgeResponse getEdge(oscal.services.v1.TransparencyGraphServiceOuterClass.GetEdgeRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getGetEdgeMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyGraphServiceOuterClass.ListEdgesResponse listEdges(oscal.services.v1.TransparencyGraphServiceOuterClass.ListEdgesRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getListEdgesMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyGraphServiceOuterClass.DeleteEdgeResponse deleteEdge(oscal.services.v1.TransparencyGraphServiceOuterClass.DeleteEdgeRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getDeleteEdgeMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyGraphServiceOuterClass.GetNodeResponse getNode(oscal.services.v1.TransparencyGraphServiceOuterClass.GetNodeRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getGetNodeMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyGraphServiceOuterClass.ListNodesResponse listNodes(oscal.services.v1.TransparencyGraphServiceOuterClass.ListNodesRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getListNodesMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyGraphServiceOuterClass.TraverseResponse traverse(oscal.services.v1.TransparencyGraphServiceOuterClass.TraverseRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getTraverseMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyGraphServiceOuterClass.ShortestPathResponse shortestPath(oscal.services.v1.TransparencyGraphServiceOuterClass.ShortestPathRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getShortestPathMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyGraphServiceOuterClass.ImpactRadiusResponse impactRadius(oscal.services.v1.TransparencyGraphServiceOuterClass.ImpactRadiusRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getImpactRadiusMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyGraphServiceOuterClass.ExplainClaimResponse explainClaim(oscal.services.v1.TransparencyGraphServiceOuterClass.ExplainClaimRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getExplainClaimMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyGraphServiceOuterClass.ComputeTrustStateResponse computeTrustState(oscal.services.v1.TransparencyGraphServiceOuterClass.ComputeTrustStateRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getComputeTrustStateMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyGraphServiceOuterClass.VerifyClosureResponse verifyClosure(oscal.services.v1.TransparencyGraphServiceOuterClass.VerifyClosureRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getVerifyClosureMethod(), getCallOptions(), request);
    }
  }

  /**
   * A stub to allow clients to do limited synchronous rpc calls to service TransparencyGraphService.
   * <pre>
   * TransparencyGraphService provides graph traversal over projected claims
   * with structured proof-state verification for xOSCAL.
   * </pre>
   */
  public static final class TransparencyGraphServiceBlockingStub
      extends io.grpc.stub.AbstractBlockingStub<TransparencyGraphServiceBlockingStub> {
    private TransparencyGraphServiceBlockingStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected TransparencyGraphServiceBlockingStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new TransparencyGraphServiceBlockingStub(channel, callOptions);
    }

    /**
     */
    public oscal.services.v1.TransparencyGraphServiceOuterClass.ProjectEdgeResponse projectEdge(oscal.services.v1.TransparencyGraphServiceOuterClass.ProjectEdgeRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getProjectEdgeMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyGraphServiceOuterClass.GetEdgeResponse getEdge(oscal.services.v1.TransparencyGraphServiceOuterClass.GetEdgeRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getGetEdgeMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyGraphServiceOuterClass.ListEdgesResponse listEdges(oscal.services.v1.TransparencyGraphServiceOuterClass.ListEdgesRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getListEdgesMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyGraphServiceOuterClass.DeleteEdgeResponse deleteEdge(oscal.services.v1.TransparencyGraphServiceOuterClass.DeleteEdgeRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getDeleteEdgeMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyGraphServiceOuterClass.GetNodeResponse getNode(oscal.services.v1.TransparencyGraphServiceOuterClass.GetNodeRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getGetNodeMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyGraphServiceOuterClass.ListNodesResponse listNodes(oscal.services.v1.TransparencyGraphServiceOuterClass.ListNodesRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getListNodesMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyGraphServiceOuterClass.TraverseResponse traverse(oscal.services.v1.TransparencyGraphServiceOuterClass.TraverseRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getTraverseMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyGraphServiceOuterClass.ShortestPathResponse shortestPath(oscal.services.v1.TransparencyGraphServiceOuterClass.ShortestPathRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getShortestPathMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyGraphServiceOuterClass.ImpactRadiusResponse impactRadius(oscal.services.v1.TransparencyGraphServiceOuterClass.ImpactRadiusRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getImpactRadiusMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyGraphServiceOuterClass.ExplainClaimResponse explainClaim(oscal.services.v1.TransparencyGraphServiceOuterClass.ExplainClaimRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getExplainClaimMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyGraphServiceOuterClass.ComputeTrustStateResponse computeTrustState(oscal.services.v1.TransparencyGraphServiceOuterClass.ComputeTrustStateRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getComputeTrustStateMethod(), getCallOptions(), request);
    }

    /**
     */
    public oscal.services.v1.TransparencyGraphServiceOuterClass.VerifyClosureResponse verifyClosure(oscal.services.v1.TransparencyGraphServiceOuterClass.VerifyClosureRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getVerifyClosureMethod(), getCallOptions(), request);
    }
  }

  /**
   * A stub to allow clients to do ListenableFuture-style rpc calls to service TransparencyGraphService.
   * <pre>
   * TransparencyGraphService provides graph traversal over projected claims
   * with structured proof-state verification for xOSCAL.
   * </pre>
   */
  public static final class TransparencyGraphServiceFutureStub
      extends io.grpc.stub.AbstractFutureStub<TransparencyGraphServiceFutureStub> {
    private TransparencyGraphServiceFutureStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected TransparencyGraphServiceFutureStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new TransparencyGraphServiceFutureStub(channel, callOptions);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.TransparencyGraphServiceOuterClass.ProjectEdgeResponse> projectEdge(
        oscal.services.v1.TransparencyGraphServiceOuterClass.ProjectEdgeRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getProjectEdgeMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.TransparencyGraphServiceOuterClass.GetEdgeResponse> getEdge(
        oscal.services.v1.TransparencyGraphServiceOuterClass.GetEdgeRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getGetEdgeMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.TransparencyGraphServiceOuterClass.ListEdgesResponse> listEdges(
        oscal.services.v1.TransparencyGraphServiceOuterClass.ListEdgesRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getListEdgesMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.TransparencyGraphServiceOuterClass.DeleteEdgeResponse> deleteEdge(
        oscal.services.v1.TransparencyGraphServiceOuterClass.DeleteEdgeRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getDeleteEdgeMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.TransparencyGraphServiceOuterClass.GetNodeResponse> getNode(
        oscal.services.v1.TransparencyGraphServiceOuterClass.GetNodeRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getGetNodeMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.TransparencyGraphServiceOuterClass.ListNodesResponse> listNodes(
        oscal.services.v1.TransparencyGraphServiceOuterClass.ListNodesRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getListNodesMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.TransparencyGraphServiceOuterClass.TraverseResponse> traverse(
        oscal.services.v1.TransparencyGraphServiceOuterClass.TraverseRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getTraverseMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.TransparencyGraphServiceOuterClass.ShortestPathResponse> shortestPath(
        oscal.services.v1.TransparencyGraphServiceOuterClass.ShortestPathRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getShortestPathMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.TransparencyGraphServiceOuterClass.ImpactRadiusResponse> impactRadius(
        oscal.services.v1.TransparencyGraphServiceOuterClass.ImpactRadiusRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getImpactRadiusMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.TransparencyGraphServiceOuterClass.ExplainClaimResponse> explainClaim(
        oscal.services.v1.TransparencyGraphServiceOuterClass.ExplainClaimRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getExplainClaimMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.TransparencyGraphServiceOuterClass.ComputeTrustStateResponse> computeTrustState(
        oscal.services.v1.TransparencyGraphServiceOuterClass.ComputeTrustStateRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getComputeTrustStateMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<oscal.services.v1.TransparencyGraphServiceOuterClass.VerifyClosureResponse> verifyClosure(
        oscal.services.v1.TransparencyGraphServiceOuterClass.VerifyClosureRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getVerifyClosureMethod(), getCallOptions()), request);
    }
  }

  private static final int METHODID_PROJECT_EDGE = 0;
  private static final int METHODID_GET_EDGE = 1;
  private static final int METHODID_LIST_EDGES = 2;
  private static final int METHODID_DELETE_EDGE = 3;
  private static final int METHODID_GET_NODE = 4;
  private static final int METHODID_LIST_NODES = 5;
  private static final int METHODID_TRAVERSE = 6;
  private static final int METHODID_SHORTEST_PATH = 7;
  private static final int METHODID_IMPACT_RADIUS = 8;
  private static final int METHODID_EXPLAIN_CLAIM = 9;
  private static final int METHODID_COMPUTE_TRUST_STATE = 10;
  private static final int METHODID_VERIFY_CLOSURE = 11;

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
        case METHODID_PROJECT_EDGE:
          serviceImpl.projectEdge((oscal.services.v1.TransparencyGraphServiceOuterClass.ProjectEdgeRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.ProjectEdgeResponse>) responseObserver);
          break;
        case METHODID_GET_EDGE:
          serviceImpl.getEdge((oscal.services.v1.TransparencyGraphServiceOuterClass.GetEdgeRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.GetEdgeResponse>) responseObserver);
          break;
        case METHODID_LIST_EDGES:
          serviceImpl.listEdges((oscal.services.v1.TransparencyGraphServiceOuterClass.ListEdgesRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.ListEdgesResponse>) responseObserver);
          break;
        case METHODID_DELETE_EDGE:
          serviceImpl.deleteEdge((oscal.services.v1.TransparencyGraphServiceOuterClass.DeleteEdgeRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.DeleteEdgeResponse>) responseObserver);
          break;
        case METHODID_GET_NODE:
          serviceImpl.getNode((oscal.services.v1.TransparencyGraphServiceOuterClass.GetNodeRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.GetNodeResponse>) responseObserver);
          break;
        case METHODID_LIST_NODES:
          serviceImpl.listNodes((oscal.services.v1.TransparencyGraphServiceOuterClass.ListNodesRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.ListNodesResponse>) responseObserver);
          break;
        case METHODID_TRAVERSE:
          serviceImpl.traverse((oscal.services.v1.TransparencyGraphServiceOuterClass.TraverseRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.TraverseResponse>) responseObserver);
          break;
        case METHODID_SHORTEST_PATH:
          serviceImpl.shortestPath((oscal.services.v1.TransparencyGraphServiceOuterClass.ShortestPathRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.ShortestPathResponse>) responseObserver);
          break;
        case METHODID_IMPACT_RADIUS:
          serviceImpl.impactRadius((oscal.services.v1.TransparencyGraphServiceOuterClass.ImpactRadiusRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.ImpactRadiusResponse>) responseObserver);
          break;
        case METHODID_EXPLAIN_CLAIM:
          serviceImpl.explainClaim((oscal.services.v1.TransparencyGraphServiceOuterClass.ExplainClaimRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.ExplainClaimResponse>) responseObserver);
          break;
        case METHODID_COMPUTE_TRUST_STATE:
          serviceImpl.computeTrustState((oscal.services.v1.TransparencyGraphServiceOuterClass.ComputeTrustStateRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.ComputeTrustStateResponse>) responseObserver);
          break;
        case METHODID_VERIFY_CLOSURE:
          serviceImpl.verifyClosure((oscal.services.v1.TransparencyGraphServiceOuterClass.VerifyClosureRequest) request,
              (io.grpc.stub.StreamObserver<oscal.services.v1.TransparencyGraphServiceOuterClass.VerifyClosureResponse>) responseObserver);
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
          getProjectEdgeMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.TransparencyGraphServiceOuterClass.ProjectEdgeRequest,
              oscal.services.v1.TransparencyGraphServiceOuterClass.ProjectEdgeResponse>(
                service, METHODID_PROJECT_EDGE)))
        .addMethod(
          getGetEdgeMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.TransparencyGraphServiceOuterClass.GetEdgeRequest,
              oscal.services.v1.TransparencyGraphServiceOuterClass.GetEdgeResponse>(
                service, METHODID_GET_EDGE)))
        .addMethod(
          getListEdgesMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.TransparencyGraphServiceOuterClass.ListEdgesRequest,
              oscal.services.v1.TransparencyGraphServiceOuterClass.ListEdgesResponse>(
                service, METHODID_LIST_EDGES)))
        .addMethod(
          getDeleteEdgeMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.TransparencyGraphServiceOuterClass.DeleteEdgeRequest,
              oscal.services.v1.TransparencyGraphServiceOuterClass.DeleteEdgeResponse>(
                service, METHODID_DELETE_EDGE)))
        .addMethod(
          getGetNodeMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.TransparencyGraphServiceOuterClass.GetNodeRequest,
              oscal.services.v1.TransparencyGraphServiceOuterClass.GetNodeResponse>(
                service, METHODID_GET_NODE)))
        .addMethod(
          getListNodesMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.TransparencyGraphServiceOuterClass.ListNodesRequest,
              oscal.services.v1.TransparencyGraphServiceOuterClass.ListNodesResponse>(
                service, METHODID_LIST_NODES)))
        .addMethod(
          getTraverseMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.TransparencyGraphServiceOuterClass.TraverseRequest,
              oscal.services.v1.TransparencyGraphServiceOuterClass.TraverseResponse>(
                service, METHODID_TRAVERSE)))
        .addMethod(
          getShortestPathMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.TransparencyGraphServiceOuterClass.ShortestPathRequest,
              oscal.services.v1.TransparencyGraphServiceOuterClass.ShortestPathResponse>(
                service, METHODID_SHORTEST_PATH)))
        .addMethod(
          getImpactRadiusMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.TransparencyGraphServiceOuterClass.ImpactRadiusRequest,
              oscal.services.v1.TransparencyGraphServiceOuterClass.ImpactRadiusResponse>(
                service, METHODID_IMPACT_RADIUS)))
        .addMethod(
          getExplainClaimMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.TransparencyGraphServiceOuterClass.ExplainClaimRequest,
              oscal.services.v1.TransparencyGraphServiceOuterClass.ExplainClaimResponse>(
                service, METHODID_EXPLAIN_CLAIM)))
        .addMethod(
          getComputeTrustStateMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.TransparencyGraphServiceOuterClass.ComputeTrustStateRequest,
              oscal.services.v1.TransparencyGraphServiceOuterClass.ComputeTrustStateResponse>(
                service, METHODID_COMPUTE_TRUST_STATE)))
        .addMethod(
          getVerifyClosureMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              oscal.services.v1.TransparencyGraphServiceOuterClass.VerifyClosureRequest,
              oscal.services.v1.TransparencyGraphServiceOuterClass.VerifyClosureResponse>(
                service, METHODID_VERIFY_CLOSURE)))
        .build();
  }

  private static abstract class TransparencyGraphServiceBaseDescriptorSupplier
      implements io.grpc.protobuf.ProtoFileDescriptorSupplier, io.grpc.protobuf.ProtoServiceDescriptorSupplier {
    TransparencyGraphServiceBaseDescriptorSupplier() {}

    @java.lang.Override
    public com.google.protobuf.Descriptors.FileDescriptor getFileDescriptor() {
      return oscal.services.v1.TransparencyGraphServiceOuterClass.getDescriptor();
    }

    @java.lang.Override
    public com.google.protobuf.Descriptors.ServiceDescriptor getServiceDescriptor() {
      return getFileDescriptor().findServiceByName("TransparencyGraphService");
    }
  }

  private static final class TransparencyGraphServiceFileDescriptorSupplier
      extends TransparencyGraphServiceBaseDescriptorSupplier {
    TransparencyGraphServiceFileDescriptorSupplier() {}
  }

  private static final class TransparencyGraphServiceMethodDescriptorSupplier
      extends TransparencyGraphServiceBaseDescriptorSupplier
      implements io.grpc.protobuf.ProtoMethodDescriptorSupplier {
    private final java.lang.String methodName;

    TransparencyGraphServiceMethodDescriptorSupplier(java.lang.String methodName) {
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
      synchronized (TransparencyGraphServiceGrpc.class) {
        result = serviceDescriptor;
        if (result == null) {
          serviceDescriptor = result = io.grpc.ServiceDescriptor.newBuilder(SERVICE_NAME)
              .setSchemaDescriptor(new TransparencyGraphServiceFileDescriptorSupplier())
              .addMethod(getProjectEdgeMethod())
              .addMethod(getGetEdgeMethod())
              .addMethod(getListEdgesMethod())
              .addMethod(getDeleteEdgeMethod())
              .addMethod(getGetNodeMethod())
              .addMethod(getListNodesMethod())
              .addMethod(getTraverseMethod())
              .addMethod(getShortestPathMethod())
              .addMethod(getImpactRadiusMethod())
              .addMethod(getExplainClaimMethod())
              .addMethod(getComputeTrustStateMethod())
              .addMethod(getVerifyClosureMethod())
              .build();
        }
      }
    }
    return result;
  }
}
