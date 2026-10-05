import XoscalSDK
var message = Oscal_Services_V1_ListCatalogsRequest()
message.pageSize = 17
let decoded = try Oscal_Services_V1_ListCatalogsRequest(serializedBytes: message.serializedData())
precondition(decoded.pageSize == 17)
let _ = Oscal_Services_V1_ListConflictsRequest()
let _ = Oscal_Services_V1_CreateClaimRequest()
let _ = Oscal_Services_V1_CreateEntityRequest()
print("Swift public protobuf models passed")
