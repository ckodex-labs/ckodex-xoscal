import grpc
import pickle
from xoscal_sdk.services.v1 import oscal_service_pb2 as messages
from xoscal_sdk.services.v1 import oscal_service_pb2_grpc as oscal
from xoscal_sdk.services.v1 import governance_service_pb2_grpc as governance
from xoscal_sdk.services.v1 import transparency_exchange_service_pb2_grpc as exchange
from xoscal_sdk.services.v1 import transparency_graph_service_pb2_grpc as graph

original = messages.ListCatalogsRequest(page_size=17)
assert messages.ListCatalogsRequest.FromString(original.SerializeToString()).page_size == 17
assert pickle.loads(pickle.dumps(original)).page_size == 17
with grpc.insecure_channel("localhost:1") as channel:
    assert callable(oscal.OscalServiceStub(channel).ListCatalogs)
    assert governance.GovernanceServiceStub(channel)
    assert exchange.TransparencyExchangeServiceStub(channel)
    assert graph.TransparencyGraphServiceStub(channel)
print("Python SDK message and four gRPC stubs passed")
