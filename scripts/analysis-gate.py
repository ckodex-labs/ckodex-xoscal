#!/usr/bin/env python3
"""Admission policy for raw Dagger analysis evidence. Missing proof fails closed."""
import argparse
import datetime as dt
import hashlib
import importlib.util
import json
from pathlib import Path
import re
import sys


class GateError(ValueError):
    pass


def require(condition, message):
    if not condition:
        raise GateError(message)


def unique_object(pairs):
    value = {}
    for key, item in pairs:
        require(key not in value, "duplicate JSON property: " + key)
        value[key] = item
    return value


def read_json(path):
    value = json.loads(path.read_text(), object_pairs_hook=unique_object)
    require(isinstance(value, dict), f"{path.name}: expected JSON object")
    return value


def status(root, name, allowed=(0,)):
    raw = (root / f"{name}.status").read_text().strip()
    require(re.fullmatch(r"[0-9]+", raw), f"{name}: malformed exit status")
    value = int(raw)
    require(value in allowed, f"{name}: operational failure (exit {value})")
    return value


def fingerprint(finding):
    # UUIDs and timestamps can change between runs; match semantic finding bytes.
    data = {key: finding[key] for key in ("title", "description", "target")}
    return hashlib.sha256(json.dumps(data, sort_keys=True, separators=(",", ":")).encode()).hexdigest()


def sbom_gate(root, exceptions_path):
    code = status(root, "sbom-validation", (0, 1))
    sbom_bytes = (root / "sbom.cyclonedx.json").read_bytes()
    sbom = read_json(root / "sbom.cyclonedx.json")
    require(sbom.get("bomFormat") == "CycloneDX" and isinstance(sbom.get("components"), list) and sbom["components"], "SBOM lacks CycloneDX component inventory")
    require(sbom.get("specVersion") in ("1.4", "1.5", "1.6", "1.7") and type(sbom.get("version")) is int and sbom["version"] >= 1, "SBOM lacks valid document version")
    require(all(isinstance(component, dict) and isinstance(component.get("name"), str) and component["name"] for component in sbom["components"]), "malformed SBOM component")
    digest = hashlib.sha256(sbom_bytes).hexdigest()
    report = read_json(root / "sbom-assessment-results.oscal.json")
    ar = report.get("assessment-results", {})
    require(isinstance(ar, dict), "malformed assessment-results")
    require(isinstance(ar.get("metadata"), dict) and ar["metadata"].get("oscal-version") == "1.1.2", "unexpected SBOM assessment format")
    results = ar.get("results")
    require(isinstance(results, list) and results, "SBOM assessment has no results")
    failures, rollups = [], []
    rollup_counts = []
    for result in results:
        require(isinstance(result, dict), "malformed assessment result")
        observations = result.get("observations", [])
        require(isinstance(observations, list), "malformed assessment observations")
        levels = {}
        for observation in observations:
            require(isinstance(observation, dict) and isinstance(observation.get("uuid"), str), "malformed SBOM observation")
            severity = [p.get("value") for p in observation.get("props", []) if p.get("name") == "severity"]
            require(len(severity) == 1 and severity[0] in ("error", "warning", "info"), "SBOM observation lacks severity")
            require(observation["uuid"] not in levels, "duplicate SBOM observation")
            levels[observation["uuid"]] = severity[0]
        findings = result.get("findings")
        require(isinstance(findings, list) and findings, "SBOM assessment has no findings or compliance verdict")
        for finding in findings:
            require(isinstance(finding, dict) and all(isinstance(finding.get(k), str) and finding[k] for k in ("title", "description")), "malformed SBOM finding")
            target = finding["target"]
            require(isinstance(target, dict), "malformed SBOM finding target")
            state = target.get("status", {}).get("state")
            require(state in ("satisfied", "not-satisfied"), "SBOM finding lacks explicit verdict")
            require(isinstance(target.get("target-id"), str), "SBOM finding lacks target-id")
            if target["target-id"] == "ntiaminimum-compliance":
                rollups.append(state)
                counts = re.fullmatch(r"Overall verdict for NTIA Minimum Elements: (\d+) error\(s\), (\d+) warning\(s\), (\d+) informational finding\(s\)\.", finding["description"])
                require(counts is not None, "unrecognized NTIA roll-up semantics")
                declared = tuple(map(int, counts.groups()))
                require(declared == tuple(list(levels.values()).count(level) for level in ("error", "warning", "info")), "SBOM severity counts contradict roll-up")
                require((state == "not-satisfied") == (declared[0] > 0), "SBOM error count contradicts NTIA verdict")
                rollup_counts.append(declared)
            elif state == "not-satisfied":
                related = finding.get("related-observations", [])
                require(related and all(o.get("observation-uuid") in levels for o in related), "failed finding lacks severity evidence")
                failures.append(finding)
    require(rollups, "SBOM assessment lacks NTIA compliance roll-up")
    require(not (code == 1 and not failures), "validator failed without actionable findings")
    require(not ("not-satisfied" in rollups and not failures), "failed SBOM roll-up without actionable findings")
    # The tool reports NTIA satisfaction when there are no errors. Its warnings
    # remain unsatisfied targets, and our strict policy still blocks every one.
    require(not (failures and not any(sum(counts) for counts in rollup_counts)), "SBOM findings contradict compliance roll-up")
    policy = read_json(exceptions_path)
    require(policy.get("version") == 1 and isinstance(policy.get("exceptions"), list), "unsupported exception policy")
    approved = set()
    today = dt.datetime.now(dt.timezone.utc).date()
    ids = set()
    for exception in policy["exceptions"]:
        require(isinstance(exception, dict), "malformed exception")
        for key in ("id", "rationale", "approved_by", "expires", "sbom_sha256", "finding_sha256"):
            require(isinstance(exception.get(key), str) and exception[key].strip(), f"exception lacks {key}")
        require(exception["id"] not in ids, "duplicate exception id")
        ids.add(exception["id"])
        require(re.fullmatch(r"[0-9a-f]{64}", exception["sbom_sha256"]) and re.fullmatch(r"[0-9a-f]{64}", exception["finding_sha256"]), "exception digest malformed")
        expiry = dt.date.fromisoformat(exception["expires"])
        require(expiry >= today, f"exception {exception['id']} expired")
        if exception["sbom_sha256"] == digest:
            approved.add(exception["finding_sha256"])
    blocked = [fingerprint(f) for f in failures if fingerprint(f) not in approved]
    result = {"sbom_sha256": digest, "violations": len(failures), "exceptions_used": len(failures) - len(blocked), "blocked": blocked}
    subject_path = root / "subject.sha256"
    if subject_path.exists():
        subject = subject_path.read_text()
        require(re.fullmatch(r"[0-9a-f]{64}\n?", subject), "malformed SBOM subject digest")
        require((root / "sbom-producer.txt").is_file() and "1.51.0" in (root / "sbom-producer.txt").read_text(), "missing pinned SBOM producer evidence")
        result["subject_sha256"] = subject.strip()
    ledger_path = root / "publishers" / "publishers.json"
    if ledger_path.is_file():
        result["publishers_sha256"] = hashlib.sha256(ledger_path.read_bytes()).hexdigest()
    enrichment_path = root / "sbom-enrichment.json"
    if enrichment_path.exists():
        require("subject_sha256" in result, "enrichment lacks original artifact binding")
        receipt = read_json(enrichment_path)
        raw_bytes = (root / "sbom.raw.cyclonedx.json").read_bytes()
        policy_bytes = (root / "sbom-distributor.json").read_bytes()
        require(receipt.get("version") == 1 and receipt.get("role") == "release-distributor", "invalid enrichment receipt")
        for field, expected in (("subject_sha256", result["subject_sha256"]), ("sbom_sha256", digest), ("raw_sbom_sha256", hashlib.sha256(raw_bytes).hexdigest()), ("policy_sha256", hashlib.sha256(policy_bytes).hexdigest())):
            require(receipt.get(field) == expected, "enrichment digest mismatch: " + field)
        helper_path = Path(__file__).with_name("enrich_sbom.py")
        if not helper_path.is_file():
            helper_path = Path(__file__).resolve().parent.parent / "dagger" / "enrich_sbom.py"
        spec = importlib.util.spec_from_file_location("sbom_enrichment", helper_path)
        helper = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(helper)
        sys.path.insert(0, str(helper_path.parent))
        replay, detail = helper.enrich(json.loads(raw_bytes, object_pairs_hook=unique_object), json.loads(policy_bytes, object_pairs_hook=unique_object), root / "publishers" if (root / "publishers").is_dir() else None, result["subject_sha256"])
        require(replay == sbom and all(receipt.get(k) == v for k, v in detail.items()), "enrichment does not reproduce from observed bytes")
    return result


def _security_gate(root):
    status(root, "govulncheck")
    stream = (root / "govulncheck.json").read_text()
    decoder, pos, messages = json.JSONDecoder(object_pairs_hook=unique_object), 0, []
    while pos < len(stream):
        while pos < len(stream) and stream[pos].isspace():
            pos += 1
        if pos == len(stream):
            break
        message, pos = decoder.raw_decode(stream, pos)
        require(isinstance(message, dict), "malformed govulncheck message")
        messages.append(message)
    require(messages and any("config" in m for m in messages) and any("progress" in m for m in messages), "govulncheck lacks completed scan evidence")
    configs = [m["config"] for m in messages if "config" in m]
    require(len(configs) == 1 and configs[0].get("scan_level") == "symbol", "govulncheck did not perform symbol analysis")
    called = []
    for message in messages:
        if "finding" in message:
            finding = message["finding"]
            require(isinstance(finding, dict) and finding.get("osv") and isinstance(finding.get("trace"), list) and finding["trace"], "malformed govulncheck finding")
            if any(frame.get("function") for frame in finding["trace"]):
                called.append(finding["osv"])
    code = status(root, "gosec", (0, 1))
    status(root, "gosec-sarif", (0, 1))
    report = read_json(root / "gosec-results.json")
    require("Golang errors" in report and not report["Golang errors"], "gosec processing errors")
    require(isinstance(report.get("Stats"), dict) and report["Stats"].get("files", 0) > 0, "gosec did not analyze files")
    issues = report.get("Issues")
    require(issues is None or isinstance(issues, list), "malformed gosec issues")
    issues = issues or []
    require(not (code == 1 and not issues), "gosec failed without findings")
    blocked, suppressed = [], 0
    for issue in issues:
        require(isinstance(issue, dict) and issue.get("severity") in ("LOW", "MEDIUM", "HIGH") and issue.get("confidence") in ("LOW", "MEDIUM", "HIGH"), "gosec finding missing severity/confidence")
        suppressions = issue.get("suppressions") or []
        require(isinstance(suppressions, list), "malformed gosec suppressions")
        reviewed_source_suppression = bool(suppressions) and all(isinstance(s, dict) and s.get("kind") == "inSource" and isinstance(s.get("justification"), str) and s["justification"].strip() for s in suppressions)
        if reviewed_source_suppression:
            suppressed += 1
        if issue["severity"] == "HIGH" and issue["confidence"] == "HIGH":
            if not reviewed_source_suppression:
                blocked.append(issue.get("rule_id", "unknown"))
    sarif = read_json(root / "gosec-results.sarif")
    require(sarif.get("version") == "2.1.0" and isinstance(sarif.get("runs"), list) and sarif["runs"], "missing/malformed gosec SARIF")
    require(sum(len(run.get("results", [])) for run in sarif["runs"]) == len(issues), "gosec JSON/SARIF disagree")
    return {"findings": len(issues), "source_suppressions": suppressed, "called_vulnerabilities": called, "blocked": blocked + called}


def image_gate(root, ci=False, artifact=False):
    if artifact:
        status(root, "extraction")
    status(root, "trivy")
    digest_name, input_path = ("artifact.sha256", "/input/artifact") if artifact else ("image.sha256", "/input/image.tar")
    digest_record = (root / digest_name).read_text()
    require(re.fullmatch(r"[0-9a-f]{64}  " + re.escape(input_path) + r"\n?", digest_record), "missing artifact digest")
    report = read_json(root / "trivy-results.json")
    if artifact:
        helper_path = Path(__file__).with_name("artifact_inventory.py")
        if not helper_path.exists():
            helper_path = Path(__file__).resolve().parents[1] / "dagger" / "artifact_inventory.py"
        spec = importlib.util.spec_from_file_location("artifact_inventory", helper_path)
        helper = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(helper)
        require(helper.replay_merge(root) == report, "artifact combined report differs from raw scanner evidence")
    artifact_type = "filesystem" if artifact else "container_image"
    require(report.get("SchemaVersion") == 2 and report.get("ArtifactName") and report.get("ArtifactType") == artifact_type, "malformed artifact report")
    require(isinstance(report.get("Results"), list) and report["Results"], "Trivy has no scan targets")
    require(report.get("Trivy", {}).get("Version") == "0.72.0", "unexpected Trivy version")
    blocked, count, packages = [], 0, 0
    unversioned_roots = []
    for result in report["Results"]:
        require(isinstance(result, dict) and result.get("Target"), "malformed Trivy target")
        inventory = result.get("Packages", [])
        require(isinstance(inventory, list) and all(isinstance(package, dict) and package.get("Name") and (package.get("Version") or (artifact and package.get("Relationship") == "root")) for package in inventory), "malformed package inventory")
        unversioned_roots.extend(package["Name"] for package in inventory if not package.get("Version"))
        packages += sum(bool(p.get("Version")) for p in inventory)
        vulnerabilities = result.get("Vulnerabilities", [])
        require(isinstance(vulnerabilities, list), "malformed Trivy findings")
        for vuln in vulnerabilities:
            require(isinstance(vuln, dict) and vuln.get("Severity") in ("UNKNOWN", "LOW", "MEDIUM", "HIGH", "CRITICAL") and vuln.get("VulnerabilityID"), "malformed vulnerability")
            count += 1
            if (ci and vuln["Severity"] in ("CRITICAL", "HIGH")) or (not ci and vuln["Severity"] == "CRITICAL" and vuln.get("FixedVersion")):
                blocked.append(vuln["VulnerabilityID"])
    require(packages > 0, "unsupported artifact: no identifiable versioned package inventory")
    return {"findings": count, "packages": packages, "unversioned_project_roots": unversioned_roots, "artifact_sha256": digest_record.split()[0], "threshold": "HIGH,CRITICAL" if ci else "CRITICAL/fixed", "blocked": blocked}


def artifact_gate(root):
    return image_gate(root, artifact=True)



def frontend_gate(root):
    status(root, "trivy")
    status(root, "npm-ci")
    status(root, "build")
    helper_path=Path(__file__).with_name("frontend_inventory.py")
    if not helper_path.is_file():helper_path=Path(__file__).resolve().parent.parent/"dagger/frontend_inventory.py"
    spec=importlib.util.spec_from_file_location("frontend_inventory",helper_path);helper=importlib.util.module_from_spec(spec);spec.loader.exec_module(helper)
    receipt,sbom=helper.replay(root)
    require(receipt.get("schema_version")==2, "frontend release requires patched published-module rebuild")
    report=read_json(root/"trivy-results.json")
    require(report.get("SchemaVersion")==2 and report.get("ArtifactType")=="cyclonedx" and report.get("Trivy",{}).get("Version")=="0.72.0", "unsupported frontend scanner report")
    require(isinstance(report.get("Results"),list) and report["Results"], "frontend scanner has no targets")
    seen,blocked,findings=set(),[],0
    for target in report["Results"]:
        require(isinstance(target,dict) and target.get("Target"), "malformed frontend scan target")
        packages=target.get("Packages",[])
        require(isinstance(packages,list),"malformed frontend package inventory")
        for package in packages:
            require(isinstance(package,dict) and package.get("Name") and package.get("Version"),"unversioned frontend scan package")
            seen.add((package["Name"],package["Version"]))
        vulnerabilities=target.get("Vulnerabilities",[])
        require(isinstance(vulnerabilities,list),"malformed frontend findings")
        for finding in vulnerabilities:
            require(isinstance(finding,dict) and finding.get("VulnerabilityID") and finding.get("Severity") in ("UNKNOWN","LOW","MEDIUM","HIGH","CRITICAL"),"malformed frontend vulnerability")
            findings+=1
            if finding["Severity"]=="CRITICAL" and finding.get("FixedVersion"):blocked.append(finding["VulnerabilityID"])
    expected={(c["name"],c["version"]) for c in sbom["components"]}
    require(expected and expected.issubset(seen),"frontend scanner omitted observed source packages")
    return {"artifact_sha256":receipt["js_sha256"],"source_map_sha256":receipt["source_map_sha256"],"sbom_sha256":receipt["sbom_sha256"],"inventory_sha256":hashlib.sha256((root/"frontend-inventory.json").read_bytes()).hexdigest(),"observed_packages":len(expected),"scanned_packages":len(seen),"findings":findings,"threshold":"CRITICAL/fixed","blocked":blocked}

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("kind", choices=("security", "sbom", "image-release", "image-ci", "artifact-release", "frontend-release"))
    parser.add_argument("--evidence", type=Path, required=True)
    parser.add_argument("--exceptions", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    result = {"policy_version": 1, "kind": args.kind, "passed": False}
    try:
        if args.kind == "security":
            result.update(_security_gate(args.evidence))
        elif args.kind == "frontend-release":
            result.update(frontend_gate(args.evidence))
        elif args.kind == "sbom":
            result.update(sbom_gate(args.evidence, args.exceptions))
        else:
            result.update(image_gate(args.evidence, args.kind == "image-ci", args.kind == "artifact-release"))
        result["passed"] = not result["blocked"]
    except (GateError, OSError, ValueError, TypeError, KeyError, AttributeError) as error:
        result["error"] = str(error)
    args.output.write_text(json.dumps(result, indent=2, sort_keys=True) + "\n")
    print(json.dumps(result, sort_keys=True))
    return 0 if result["passed"] else 1


if __name__ == "__main__":
    sys.exit(main())
