"""Synthetic receipt fixtures; never actual browser acceptance evidence."""
import copy
import presentation_browser_verify as browser

def fixture_route_evidence(root, receipt):
    for check in receipt["checks"]:
        fixture_model_movement(check)
        if check["id"].endswith(": cold contract exact actual held response"):
            check["detail"] = {"status": 200, "digest": browser.digest(root / "openapi.json"),
                               "size": (root / "openapi.json").stat().st_size}
        if ": API native reentry/" in check["id"] and check["id"].endswith(("actual prior Scalar fragment", "native close retains prior fragment", "initial model deep link preserves requested route")):
            check["detail"] = {"fragment": "#description/introduction" if "/intro reentry:" in check["id"] else "#models/oscalservicesv1CreateComponentDefinitionRequest"}
        if check["id"].endswith("model route: newer host fragment cancels previous route"):
            check["detail"] = [{"hash": "#unknown-api-model-route", "owned": True}] * 4
        if check["id"].endswith("model route: actual Tab cancels previous model ownership"):
            check["detail"] = [{"owned": True, "moved": True}] * 4
        if check["id"].endswith("pre-close setup preserves fragment and summary focus"):
            check["detail"] = {"fragment": "#description/introduction" if "/intro reentry:" in check["id"] else "#models/oscalservicesv1CreateComponentDefinitionRequest", "summaryFocused": True}
        if check["id"].endswith("new route: actual hash navigation occurs during loading"):
            check["detail"] = {"phase": "loading", "requested": "#models/oscalservicesv1CreateComponentDefinitionRequest"}
        if check["id"].endswith(("new route: requested model owns visible focus after readiness", "initial model deep link: requested model owns visible focus after readiness", "ready model navigation: requested model owns visible focus after readiness")):
            viewport = {"width": 390, "height": 844} if check["id"].startswith("mobile") else {"width": 1280, "height": 900}
            sample = {"ready": True, "target": True, "visible": True, "unoccluded": True,
                      "hash": "#models/oscalservicesv1CreateComponentDefinitionRequest", "viewport": viewport,
                      "rect": {"top": 100, "bottom": 120, "left": 10, "right": 210, "width": 200, "height": 20}}
            check["detail"] = [copy.deepcopy(sample) for _ in range(4)]


def fixture_model_movement(check):
    if check["id"].endswith("pre-ready host focus is observed"):
        check["detail"] = {"phase": "loading", "modelHadFocus": True, "hostFocus": True,
                           "requested": "#models/oscalservicesv1CreateComponentDefinitionRequest", "tag": "A", "href": "openapi.json", "host": True}
    if check["id"].endswith("ready preserves moved host focus"):
        sample = {"ready": True, "target": True, "visible": True, "unoccluded": True, "hash": "#description/introduction",
                  "viewport": {"width": 390, "height": 844} if check["id"].startswith("mobile") else {"width": 1280, "height": 900},
                  "rect": {"top": 100, "bottom": 120, "left": 10, "right": 210, "width": 200, "height": 20}}
        check["detail"] = [copy.deepcopy(sample) for _ in range(4)]
