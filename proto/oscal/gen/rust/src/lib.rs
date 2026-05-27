// OSCAL Protobuf Rust SDK
// This library contains generated protobuf code for OSCAL models

// prost-build generates all modules in a single file structure
// The generated code is included automatically via build.rs

// Re-export all generated modules
pub mod oscal {
    pub mod common {
        pub mod v1 {
            include!(concat!(env!("OUT_DIR"), "/oscal.common.v1.rs"));
        }
    }
    pub mod catalog {
        pub mod v1 {
            include!(concat!(env!("OUT_DIR"), "/oscal.catalog.v1.rs"));
        }
    }
    pub mod profile {
        pub mod v1 {
            include!(concat!(env!("OUT_DIR"), "/oscal.profile.v1.rs"));
        }
    }
    pub mod component_definition {
        pub mod v1 {
            include!(concat!(
                env!("OUT_DIR"),
                "/oscal.component_definition.v1.rs"
            ));
        }
    }
    pub mod ssp {
        pub mod v1 {
            include!(concat!(env!("OUT_DIR"), "/oscal.ssp.v1.rs"));
        }
    }
    pub mod assessment_plan {
        pub mod v1 {
            include!(concat!(env!("OUT_DIR"), "/oscal.assessment_plan.v1.rs"));
        }
    }
    pub mod assessment_results {
        pub mod v1 {
            include!(concat!(env!("OUT_DIR"), "/oscal.assessment_results.v1.rs"));
        }
    }
    pub mod poam {
        pub mod v1 {
            include!(concat!(env!("OUT_DIR"), "/oscal.poam.v1.rs"));
        }
    }
    pub mod mapping {
        pub mod v1 {
            include!(concat!(env!("OUT_DIR"), "/oscal.mapping.v1.rs"));
        }
    }
    pub mod services {
        pub mod v1 {
            include!(concat!(env!("OUT_DIR"), "/oscal.services.v1.rs"));
        }
    }
}
