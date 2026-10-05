const assert = require("node:assert/strict");
const { create } = require("@bufbuild/protobuf");
const { ListCatalogsRequestSchema, OscalService } = require("@ckodex/xoscal-sdk/services/v1/oscal_service_pb");
assert.equal(create(ListCatalogsRequestSchema, { pageSize: 17 }).pageSize, 17);
assert.ok(OscalService.methods.length > 0);
console.log("Node CJS SDK import passed");
