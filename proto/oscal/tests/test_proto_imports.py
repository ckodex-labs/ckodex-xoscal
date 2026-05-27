#!/usr/bin/env python3
"""
Proto validation tests for OSCAL protobuf schemas.
Tests that all generated Python SDK modules can be imported and instantiated.
"""

import sys
import os

# Add the generated Python SDK to the path
gen_path = os.path.join(os.path.dirname(__file__), '..', 'gen', 'python')
sys.path.insert(0, gen_path)

def test_common_imports():
    """Test that common proto types can be imported."""
    try:
        from common.v1 import common_pb2
        print("✓ Common proto imports successful")
        
        # Test basic instantiation
        uuid = common_pb2.UUID()
        uuid.value = "550e8400-e29b-41d4-a716-446655440000"
        
        metadata = common_pb2.Metadata()
        metadata.title = "Test Document"
        
        print("✓ Common proto types instantiated successfully")
        return True
    except Exception as e:
        print(f"✗ Common proto test failed: {e}")
        return False

def test_catalog_imports():
    """Test that catalog proto types can be imported."""
    try:
        from catalog.v1 import catalog_pb2
        print("✓ Catalog proto imports successful")
        
        # Test basic instantiation
        catalog = catalog_pb2.Catalog()
        catalog.uuid.value = "550e8400-e29b-41d4-a716-446655440000"
        
        control = catalog_pb2.Control()
        control.id.value = "ac-1"
        control.title.value = "Access Control Policy and Procedures"
        
        print("✓ Catalog proto types instantiated successfully")
        return True
    except Exception as e:
        print(f"✗ Catalog proto test failed: {e}")
        return False

def test_profile_imports():
    """Test that profile proto types can be imported."""
    try:
        from oscal_profile.v1 import profile_pb2
        print("✓ Profile proto imports successful")
        
        # Test basic instantiation
        profile = profile_pb2.Profile()
        profile.uuid.value = "550e8400-e29b-41d4-a716-446655440000"
        
        import_profile = profile_pb2.Import()
        import_profile.href.value = "https://example.com/catalog.json"
        
        print("✓ Profile proto types instantiated successfully")
        return True
    except Exception as e:
        print(f"✗ Profile proto test failed: {e}")
        return False

def test_component_definition_imports():
    """Test that component definition proto types can be imported."""
    try:
        from component_definition.v1 import component_pb2
        print("✓ Component Definition proto imports successful")
        
        # Test basic instantiation
        component_def = component_pb2.ComponentDefinition()
        component_def.uuid.value = "550e8400-e29b-41d4-a716-446655440000"
        
        component = component_pb2.DefinedComponent()
        component.uuid.value = "550e8400-e29b-41d4-a716-446655440000"
        
        print("✓ Component Definition proto types instantiated successfully")
        return True
    except Exception as e:
        print(f"✗ Component Definition proto test failed: {e}")
        return False

def test_ssp_imports():
    """Test that SSP proto types can be imported."""
    try:
        from ssp.v1 import ssp_pb2
        print("✓ SSP proto imports successful")
        
        # Test basic instantiation
        ssp = ssp_pb2.SystemSecurityPlan()
        ssp.uuid.value = "550e8400-e29b-41d4-a716-446655440000"
        
        system_characteristics = ssp_pb2.SystemCharacteristics()
        system_characteristics.system_name = "Test System"
        
        print("✓ SSP proto types instantiated successfully")
        return True
    except Exception as e:
        print(f"✗ SSP proto test failed: {e}")
        return False

def test_assessment_plan_imports():
    """Test that assessment plan proto types can be imported."""
    try:
        from assessment_plan.v1 import assessment_plan_pb2
        print("✓ Assessment Plan proto imports successful")
        
        # Test basic instantiation
        assessment_plan = assessment_plan_pb2.AssessmentPlan()
        assessment_plan.uuid.value = "550e8400-e29b-41d4-a716-446655440000"
        
        import_ssp = assessment_plan_pb2.ImportSsp()
        import_ssp.href.value = "https://example.com/ssp.json"
        
        print("✓ Assessment Plan proto types instantiated successfully")
        return True
    except Exception as e:
        print(f"✗ Assessment Plan proto test failed: {e}")
        return False

def test_assessment_results_imports():
    """Test that assessment results proto types can be imported."""
    try:
        from assessment_results.v1 import assessment_results_pb2
        print("✓ Assessment Results proto imports successful")
        
        # Test basic instantiation
        assessment_results = assessment_results_pb2.AssessmentResults()
        assessment_results.uuid.value = "550e8400-e29b-41d4-a716-446655440000"
        
        result = assessment_results_pb2.Result()
        result.uuid.value = "550e8400-e29b-41d4-a716-446655440000"
        
        print("✓ Assessment Results proto types instantiated successfully")
        return True
    except Exception as e:
        print(f"✗ Assessment Results proto test failed: {e}")
        return False

def test_poam_imports():
    """Test that POAM proto types can be imported."""
    try:
        from poam.v1 import poam_pb2
        print("✓ POAM proto imports successful")
        
        # Test basic instantiation
        poam = poam_pb2.PoamItem()
        poam.uuid.value = "550e8400-e29b-41d4-a716-446655440000"
        
        risk = poam_pb2.Risk()
        risk.uuid.value = "550e8400-e29b-41d4-a716-446655440000"
        
        print("✓ POAM proto types instantiated successfully")
        return True
    except Exception as e:
        print(f"✗ POAM proto test failed: {e}")
        return False

def test_mapping_imports():
    """Test that mapping proto types can be imported."""
    try:
        from mapping.v1 import mapping_pb2
        print("✓ Mapping proto imports successful")
        
        # Test basic instantiation
        mapping = mapping_pb2.MappingCollection()
        mapping.uuid.value = "550e8400-e29b-41d4-a716-446655440000"
        
        map_entry = mapping_pb2.Map()
        map_entry.uuid.value = "550e8400-e29b-41d4-a716-446655440000"
        
        print("✓ Mapping proto types instantiated successfully")
        return True
    except Exception as e:
        print(f"✗ Mapping proto test failed: {e}")
        return False

def main():
    """Run all proto validation tests."""
    print("=" * 60)
    print("OSCAL Protobuf Validation Tests")
    print("=" * 60)
    
    tests = [
        test_common_imports,
        test_catalog_imports,
        test_profile_imports,
        test_component_definition_imports,
        test_ssp_imports,
        test_assessment_plan_imports,
        test_assessment_results_imports,
        test_poam_imports,
        test_mapping_imports,
    ]
    
    results = []
    for test in tests:
        try:
            result = test()
            results.append(result)
        except Exception as e:
            print(f"✗ Test {test.__name__} failed with exception: {e}")
            results.append(False)
    
    print("=" * 60)
    passed = sum(results)
    total = len(results)
    print(f"Tests passed: {passed}/{total}")
    print("=" * 60)
    
    if passed == total:
        print("✓ All tests passed!")
        return 0
    else:
        print("✗ Some tests failed")
        return 1

if __name__ == "__main__":
    sys.exit(main())
