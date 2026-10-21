#!/usr/bin/env python3
"""
generate_itsp10033_oscal.py

Parses the official Canadian Centre for Cyber Security (CCCS) ITSP.10.033 PDF:
"Security and Privacy Controls and Assurance Activities Catalogue"
(Effective March 31, 2026, superseding ITSG-33 Annex 3A)
and produces an authoritative, fully conformant OSCAL 1.2.3 JSON Catalog.
"""

import json
import os
import re
import sys
import uuid
from datetime import datetime, timezone
import pypdf

PDF_PATH = "itsp.10.033-2026-eng.pdf"
OUTPUT_DIR = "data/frameworks/cccs-itsp-10-033"
OUTPUT_FILE = os.path.join(OUTPUT_DIR, "catalog.json")

# Namespace UUID for deterministic UUIDv5 generation
NS_UUID = uuid.uuid5(uuid.NAMESPACE_DNS, "cyber.gc.ca/itsp.10.033")

FAMILY_DEFINITIONS = [
    ("AC", "ac", "Access Control (Contrôle d'accès)"),
    ("AT", "at", "Awareness and Training (Sensibilisation et formation)"),
    ("AU", "au", "Audit and Accountability (Vérification et responsabilité)"),
    ("CA", "ca", "Assessment, Authorization, and Monitoring (Évaluation, autorisation et surveillance)"),
    ("CM", "cm", "Configuration Management (Gestion de la configuration)"),
    ("CP", "cp", "Contingency Planning (Planification de la continuité des opérations)"),
    ("IA", "ia", "Identification and Authentication (Identification et authentification)"),
    ("IR", "ir", "Incident Response (Intervention en cas d'incident)"),
    ("MA", "ma", "Maintenance (Maintenance)"),
    ("MP", "mp", "Media Protection (Protection des supports)"),
    ("PE", "pe", "Physical and Environmental Protection (Protection physique et environnementale)"),
    ("PL", "pl", "Planning (Planification)"),
    ("PM", "pm", "Program Management (Gestion des programmes)"),
    ("PS", "ps", "Personnel Security (Sécurité du personnel)"),
    ("PT", "pt", "Personal Information Handling and Transparency (Traitement des renseignements personnels et transparence)"),
    ("RA", "ra", "Risk Assessment (Évaluation des risques)"),
    ("SA", "sa", "System and Services Acquisition (Acquisition de systèmes et de services)"),
    ("SC", "sc", "System and Communications Protection (Protection des systèmes et des communications)"),
    ("SI", "si", "System and Information Integrity (Intégrité des systèmes et de l'information)"),
    ("SR", "sr", "Supply Chain Risk Management (Gestion des risques liés à la chaîne d'approvisionnement)"),
]

def make_uuid(name: str) -> str:
    return str(uuid.uuid5(NS_UUID, name))

def extract_pdf_lines(pdf_path: str):
    reader = pypdf.PdfReader(pdf_path)
    cleaned = []
    # Controls start at page 30 (index 29) to page 420
    for p in range(29, len(reader.pages)):
        lines = reader.pages[p].extract_text().split("\n")
        for l in lines:
            s = l.strip()
            if not s:
                continue
            if s == "UNCLASSIFIED / NON CLASSIFIÉ":
                continue
            if s == "ITSP.10.033":
                continue
            if re.match(r"^\d+$", s):
                continue
            cleaned.append(s)
    return cleaned

def parse_related(rel_text: str):
    cleaned = rel_text.strip()
    if not cleaned or cleaned.lower().startswith("none"):
        return []
    # Find all control IDs like AC-03, IA-04(400), etc.
    raw_ids = re.findall(r"([A-Z]{2}\s*-\s*\d{2,3}(?:\s*\(\s*\d{1,3}\s*\))?)", cleaned)
    links = []
    for r in raw_ids:
        compact = re.sub(r"\s+", "", r)
        # Convert AC-02(01) -> ac-2.1 or AC-03 -> ac-3
        m = re.match(r"^([A-Z]{2})-(\d+)(?:\((\d+)\))?$", compact)
        if m:
            fam = m.group(1).lower()
            num = int(m.group(2))
            enh = m.group(3)
            if enh:
                target_id = f"{fam}-{num}.{int(enh)}"
            else:
                target_id = f"{fam}-{num}"
            links.append({"rel": "related", "href": f"#{target_id}"})
    return links

def parse_enhancement(enh_match, next_enh_start, enh_text, base_norm_id, base_raw_id):
    enh_num_str = enh_match.group(1).strip()
    enh_num = int(enh_num_str)
    enh_title_part1 = enh_match.group(2).strip()
    enh_title_part2 = enh_match.group(3).strip() if enh_match.group(3) else ""
    
    # In PDF, format is: "(01) Account management | Automated system account management" or "(01) Title"
    if enh_title_part2:
        title = enh_title_part2
    else:
        # Check if title has colon or bar
        if ":" in enh_title_part1:
            title = enh_title_part1.split(":", 1)[1].strip()
        elif "|" in enh_title_part1:
            title = enh_title_part1.split("|", 1)[1].strip()
        else:
            title = enh_title_part1
            
    norm_id = f"{base_norm_id}.{enh_num}"
    label = f"{base_raw_id}({enh_num_str})"
    
    enh_body = enh_text[enh_match.end():next_enh_start].strip()
    
    # Extract Statement, Discussion, Related controls
    disc_idx = enh_body.find("Discussion:")
    rel_idx = enh_body.find("Related controls and activities:")
    if rel_idx == -1:
        rel_idx = enh_body.find("Related controls:")
        
    stmt_text = ""
    disc_text = ""
    rel_text = ""
    
    if disc_idx != -1:
        stmt_text = enh_body[:disc_idx].strip()
        if rel_idx != -1 and rel_idx > disc_idx:
            disc_text = enh_body[disc_idx + len("Discussion:"):rel_idx].strip()
            rel_text = enh_body[rel_idx:].strip()
            # remove "Related controls and activities:" header
            rel_text = re.sub(r"^Related controls(?: and activities)?:\s*", "", rel_text).strip()
        else:
            disc_text = enh_body[disc_idx + len("Discussion:"):].strip()
    elif rel_idx != -1:
        stmt_text = enh_body[:rel_idx].strip()
        rel_text = enh_body[rel_idx:].strip()
        rel_text = re.sub(r"^Related controls(?: and activities)?:\s*", "", rel_text).strip()
    else:
        stmt_text = enh_body.strip()
        
    parts = []
    if stmt_text:
        parts.append({
            "id": f"{norm_id}_smt",
            "name": "statement",
            "prose": stmt_text
        })
    if disc_text:
        parts.append({
            "id": f"{norm_id}_gdn",
            "name": "guidance",
            "prose": disc_text
        })
        
    links = parse_related(rel_text) if rel_text else []
    
    enh_ctrl = {
        "id": norm_id,
        "class": "enhancement",
        "title": title,
        "props": [
            {"name": "label", "value": label},
            {"name": "sort-id", "value": f"{base_norm_id}.{enh_num:03d}"}
        ],
        "parts": parts
    }
    if links:
        enh_ctrl["links"] = links
        
    return enh_ctrl

def parse_control_body(base_norm_id, base_raw_id, title, kind, body):
    is_withdrawn = kind.startswith("withdrawn")
    
    # Split into sections:
    # 1. Statement (starts right after Control: or Activity: until Discussion:)
    # 2. Discussion: (until GC discussion: or Related controls...)
    # 3. GC discussion: (if present)
    # 4. Related controls and activities:
    # 5. Enhancements:
    # 6. References:
    
    disc_idx = body.find("Discussion:")
    gc_disc_idx = body.find("GC discussion:")
    rel_idx = body.find("Related controls and activities:")
    if rel_idx == -1:
        rel_idx = body.find("Related controls:")
    enh_idx = body.find("Enhancements:")
    ref_idx = body.find("References:")
    
    stmt_end = len(body)
    for pos in [disc_idx, gc_disc_idx, rel_idx, enh_idx, ref_idx]:
        if pos != -1 and pos < stmt_end:
            stmt_end = pos
            
    stmt_text = body[:stmt_end].strip()
    
    disc_text = ""
    if disc_idx != -1:
        d_end = len(body)
        for pos in [gc_disc_idx, rel_idx, enh_idx, ref_idx]:
            if pos != -1 and pos > disc_idx and pos < d_end:
                d_end = pos
        disc_text = body[disc_idx + len("Discussion:"):d_end].strip()
        
    gc_disc_text = ""
    if gc_disc_idx != -1:
        gcd_end = len(body)
        for pos in [rel_idx, enh_idx, ref_idx]:
            if pos != -1 and pos > gc_disc_idx and pos < gcd_end:
                gcd_end = pos
        gc_disc_text = body[gc_disc_idx + len("GC discussion:"):gcd_end].strip()
        
    rel_text = ""
    if rel_idx != -1:
        r_end = len(body)
        for pos in [enh_idx, ref_idx]:
            if pos != -1 and pos > rel_idx and pos < r_end:
                r_end = pos
        raw_r = body[rel_idx:r_end].strip()
        rel_text = re.sub(r"^Related controls(?: and activities)?:\s*", "", raw_r).strip()
        
    # Enhancements
    enh_list = []
    if enh_idx != -1:
        e_end = ref_idx if ref_idx != -1 and ref_idx > enh_idx else len(body)
        enh_section = body[enh_idx:e_end].strip()
        if "None." not in enh_section[:25]:
            enh_re = re.compile(r"(?:^|\n)\(\s*(\d{2,3})\s*\)\s+([^\n:]+)(?::\s*([^\n]+))?", re.MULTILINE)
            enh_matches = list(enh_re.finditer(enh_section))
            for i, em in enumerate(enh_matches):
                next_start = enh_matches[i+1].start() if i+1 < len(enh_matches) else len(enh_section)
                enh_ctrl = parse_enhancement(em, next_start, enh_section, base_norm_id, base_raw_id)
                enh_list.append(enh_ctrl)
                
    parts = []
    if is_withdrawn:
        parts.append({
            "id": f"{base_norm_id}_wth",
            "name": "withdrawn",
            "prose": stmt_text or "This control has been withdrawn."
        })
    else:
        if stmt_text:
            parts.append({
                "id": f"{base_norm_id}_smt",
                "name": "statement",
                "prose": stmt_text
            })
        if disc_text:
            parts.append({
                "id": f"{base_norm_id}_gdn",
                "name": "guidance",
                "prose": disc_text
            })
        if gc_disc_text:
            parts.append({
                "id": f"{base_norm_id}_gc_gdn",
                "name": "gc-guidance",
                "prose": gc_disc_text
            })
            
    props = [
        {"name": "label", "value": base_raw_id},
        {"name": "sort-id", "value": base_norm_id},
        {"name": "status", "value": "withdrawn" if is_withdrawn else "active"}
    ]
    if not is_withdrawn:
        props.append({"name": "type", "value": kind})
        
    ctrl_obj = {
        "id": base_norm_id,
        "class": "activity" if kind == "activity" else "control",
        "title": title,
        "props": props,
        "parts": parts
    }
    
    links = parse_related(rel_text) if rel_text else []
    if links:
        ctrl_obj["links"] = links
        
    if enh_list:
        ctrl_obj["controls"] = enh_list
        
    return ctrl_obj

def build_oscal_catalog():
    print(f"Reading PDF from {PDF_PATH}...")
    lines = extract_pdf_lines(PDF_PATH)
    full_text = "\n".join(lines)
    print(f"Extracted {len(lines)} cleaned lines.")
    
    # Match control boundaries
    ctrl_re = re.compile(
        r"(?:^|\n)([A-Z]{2}\s*-\s*\d{2,3})\s+([^\n]+)\s*\n(Control:|Activity:|Withdrawn:)",
        re.MULTILINE
    )
    matches = list(ctrl_re.finditer(full_text))
    print(f"Found {len(matches)} control headers.")
    
    controls_by_family = {fam_code: [] for _, fam_code, _ in FAMILY_DEFINITIONS}
    total_enhancements = 0
    
    for i, m in enumerate(matches):
        raw_id_dirty = m.group(1).strip()
        raw_id = re.sub(r"\s+", "", raw_id_dirty)
        title = m.group(2).strip()
        kind = m.group(3).strip().replace(":", "").lower()
        
        start_idx = m.end()
        end_idx = matches[i+1].start() if i+1 < len(matches) else len(full_text)
        body = full_text[start_idx:end_idx].strip()
        
        fam_upper = raw_id.split("-")[0]
        num_str = raw_id.split("-")[1]
        try:
            num = int(num_str)
            norm_id = f"{fam_upper.lower()}-{num}"
        except ValueError:
            norm_id = raw_id.lower()
            
        ctrl = parse_control_body(norm_id, raw_id, title, kind, body)
        total_enhancements += len(ctrl.get("controls", []))
        
        fam_code = fam_upper.lower()
        if fam_code in controls_by_family:
            controls_by_family[fam_code].append(ctrl)
        else:
            print(f"Warning: unknown family code: {fam_code}")
            
    print(f"Parsed {len(matches)} base controls and {total_enhancements} enhancements.")
    
    # Build OSCAL Groups
    groups = []
    for fam_upper, fam_code, fam_title in FAMILY_DEFINITIONS:
        family_ctrls = controls_by_family.get(fam_code, [])
        grp = {
            "id": fam_code,
            "class": "family",
            "title": fam_title,
            "props": [
                {"name": "label", "value": fam_upper}
            ],
            "controls": family_ctrls
        }
        groups.append(grp)
        print(f"  Group {fam_upper}: {len(family_ctrls)} controls")
        
    catalog_uuid = make_uuid("catalog:cccs-itsp-10-033")
    party_uuid = make_uuid("party:cccs")
    
    catalog = {
        "catalog": {
            "uuid": catalog_uuid,
            "metadata": {
                "title": "ITSP.10.033: Security and Privacy Controls and Assurance Activities Catalogue",
                "published": "2026-03-31T00:00:00Z",
                "last-modified": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
                "version": "1.0 (2026-03-31)",
                "oscal-version": "1.2.3",
                "props": [
                    {"name": "canonical-id", "value": "ITSP.10.033"},
                    {"name": "publisher", "value": "Canadian Centre for Cyber Security / Communications Security Establishment"},
                    {"name": "jurisdiction", "value": "CA"},
                    {"name": "catalogue-number", "value": "D97-3/10-033-2026E-PDF"},
                    {"name": "isbn", "value": "978-0-660-76132-9"},
                    {"name": "effective-date", "value": "2026-03-31"},
                    {"name": "supersedes", "value": "ITSG-33 Annex 3A"}
                ],
                "roles": [
                    {
                        "id": "creator",
                        "title": "Document Author / Issuing Authority"
                    }
                ],
                "parties": [
                    {
                        "uuid": party_uuid,
                        "type": "organization",
                        "name": "Canadian Centre for Cyber Security / Communications Security Establishment",
                        "short-name": "CCCS",
                        "email-addresses": [
                            "contact@cyber.gc.ca"
                        ],
                        "telephone-numbers": [
                            {"type": "office", "number": "+1-833-292-3788"}
                        ],
                        "addresses": [
                            {
                                "type": "work",
                                "addr-lines": ["P.O. Box 9703, Terminal"],
                                "city": "Ottawa",
                                "state": "ON",
                                "postal-code": "K1G 3Z4",
                                "country": "CA"
                            }
                        ]
                    }
                ],
                "responsible-parties": [
                    {
                        "role-id": "creator",
                        "party-uuids": [party_uuid]
                    }
                ],
                "remarks": "Official Government of Canada security and privacy control and assurance activities catalogue issued under the authority of the Head, Canadian Centre for Cyber Security (Cyber Centre). Harmonized with NIST SP 800-53 Rev. 5 and adapted for Canadian unique legislative, business, and national security requirements."
            },
            "groups": groups,
            "back-matter": {
                "resources": [
                    {
                        "uuid": make_uuid("resource:itsp.10.033.pdf"),
                        "title": "ITSP.10.033: Security and Privacy Controls and Assurance Activities Catalogue (PDF)",
                        "citation": {
                            "text": "Canadian Centre for Cyber Security. (2026). Security and privacy controls and assurance activities catalogue (ITSP.10.033). Communications Security Establishment, Ottawa."
                        },
                        "rlinks": [
                            {"href": "https://publications.gc.ca/collections/collection_2026/cstc-csec/D97-3-10-033-2026-eng.pdf", "media-type": "application/pdf"},
                            {"href": "https://www.cyber.gc.ca/en/guidance/cyber-security-privacy-risk-management/itsp10033", "media-type": "text/html"}
                        ]
                    },
                    {
                        "uuid": make_uuid("resource:nist.sp.800-53.r5"),
                        "title": "NIST Special Publication 800-53 Revision 5: Security and Privacy Controls for Information Systems and Organizations",
                        "citation": {
                            "text": "National Institute of Standards and Technology. (2020). Security and Privacy Controls for Information Systems and Organizations (NIST SP 800-53, Rev. 5). https://doi.org/10.6028/NIST.SP.800-53r5"
                        }
                    },
                    {
                        "uuid": make_uuid("resource:tbs.psg"),
                        "title": "Treasury Board of Canada Secretariat: Policy on Government Security",
                        "citation": {
                            "text": "Treasury Board of Canada Secretariat. (2019). Policy on Government Security. Government of Canada."
                        }
                    },
                    {
                        "uuid": make_uuid("resource:tbs.dsm"),
                        "title": "Treasury Board of Canada Secretariat: Directive on Security Management",
                        "citation": {
                            "text": "Treasury Board of Canada Secretariat. (2019). Directive on Security Management. Government of Canada."
                        }
                    }
                ]
            }
        }
    }
    
    os.makedirs(OUTPUT_DIR, exist_ok=True)
    with open(OUTPUT_FILE, "w", encoding="utf-8") as f:
        json.dump(catalog, f, indent=2, ensure_ascii=False)
        
    # Also write top-level export copy
    with open("itsp-10-033-catalog.oscal.json", "w", encoding="utf-8") as f:
        json.dump(catalog, f, indent=2, ensure_ascii=False)
        
    size_mb = os.path.getsize(OUTPUT_FILE) / (1024 * 1024)
    print(f"\nSuccessfully generated OSCAL 1.2.3 catalog: {OUTPUT_FILE} ({size_mb:.2f} MB)")
    print(f"Top-level export written to: itsp-10-033-catalog.oscal.json")

if __name__ == "__main__":
    build_oscal_catalog()
