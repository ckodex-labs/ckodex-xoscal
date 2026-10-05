import oscal.services.v1.OscalServiceOuterClass.ListCatalogsRequest;
import oscal.services.v1.OscalServiceGrpc;
import oscal.services.v1.GovernanceServiceGrpc;
import oscal.services.v1.TransparencyExchangeServiceGrpc;
import oscal.services.v1.TransparencyGraphServiceGrpc;

public class JavaSmoke {
 public static void main(String[] args) throws Exception {
  var message = ListCatalogsRequest.newBuilder().setPageSize(17).build();
  if (ListCatalogsRequest.parseFrom(message.toByteArray()).getPageSize() != 17) throw new AssertionError("protobuf roundtrip");
  if (OscalServiceGrpc.getServiceDescriptor().getMethods().isEmpty() || GovernanceServiceGrpc.getServiceDescriptor().getMethods().isEmpty() || TransparencyExchangeServiceGrpc.getServiceDescriptor().getMethods().isEmpty() || TransparencyGraphServiceGrpc.getServiceDescriptor().getMethods().isEmpty()) throw new AssertionError("service descriptors missing");
  System.out.println("Java SDK message and four gRPC descriptors passed");
 }
}
