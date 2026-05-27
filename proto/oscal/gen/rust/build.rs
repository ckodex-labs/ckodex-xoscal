fn main() {
    let proto_files = vec![
        "../../common/v1/common.proto",
        "../../catalog/v1/catalog.proto",
        "../../profile/v1/profile.proto",
        "../../component_definition/v1/component.proto",
        "../../ssp/v1/ssp.proto",
        "../../assessment_plan/v1/assessment_plan.proto",
        "../../assessment_results/v1/assessment_results.proto",
        "../../poam/v1/poam.proto",
        "../../mapping/v1/mapping.proto",
        "../../services/v1/oscal_service.proto",
    ];

    prost_build::Config::new()
        .btree_map(["."])
        .compile_protos(&proto_files, &["../../"])
        .expect("Failed to compile protos");
}
