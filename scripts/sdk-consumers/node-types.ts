import { create } from "@bufbuild/protobuf";
import { ListCatalogsRequestSchema, type ListCatalogsRequest } from "@ckodex/xoscal-sdk/services/v1/oscal_service_pb";
const request: ListCatalogsRequest = create(ListCatalogsRequestSchema, { pageSize: 17 });
const pageSize: number = request.pageSize;
void pageSize;
