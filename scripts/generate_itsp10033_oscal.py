#!/usr/bin/env python3
"""
generate_itsp10033_oscal.py

Parses the official Canadian Centre for Cyber Security (CCCS) ITSP.10.033 PDF:
"Cyber security and privacy risk management series: A lifecycle approach:
Security and privacy controls and assurance activities catalogue"
(Effective March 31, 2026, superseding ITSG-33 Annex 3A)
and produces an authoritative, fully conformant OSCAL 1.2.3 JSON Catalog
implementing the 7 Subagent Strategies.
"""

import json
import os
import re
import sys
import uuid
from datetime import datetime, timezone
import pypdf

PDF_PATH = os.environ.get("ITSP_PDF_PATH", "data/sources/itsp.10.033-2026-eng.pdf")
if not os.path.exists(PDF_PATH) and os.path.exists("itsp.10.033-2026-eng.pdf"):
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
    raw_ids = re.findall(r"([A-Z]{2}\s*-\s*\d{2,3}(?:\s*\(\s*\d{1,3}\s*\))?)", cleaned)
    links = []
    seen = set()
    for r in raw_ids:
        compact = re.sub(r"\s+", "", r)
        m = re.match(r"^([A-Z]{2})-(\d+)(?:\((\d+)\))?$", compact)
        if m:
            fam = m.group(1).lower()
            num = int(m.group(2))
            enh = m.group(3)
            target_id = f"{fam}-{num}.{int(enh)}" if enh else f"{fam}-{num}"
            if target_id not in seen:
                seen.add(target_id)
                links.append({"rel": "related", "href": f"#{target_id}"})
    return links

def parse_references(ref_text: str):
    cleaned = ref_text.strip()
    if not cleaned or cleaned.lower().startswith("none"):
        return []
    lines = [l.strip() for l in cleaned.split("\n") if l.strip()]
    links = []
    for l in lines:
        title = re.sub(r"^[•\-\*]\s*", "", l).strip()
        if not title:
            continue
        # Extract URL if present in parentheses or text
        url_match = re.search(r"https?://[^\s)]+", title)
        href = url_match.group(0) if url_match else f"urn:xoscal:ref:{re.sub(r'[^a-zA-Z0-9]+', '-', title).strip('-').lower()}"
        links.append({
            "rel": "reference",
            "href": href,
            "text": title
        })
    return links

def extract_params(text: str, prefix: str):
    # Regex to match innermost [Assignment: ...] or [Selection...: ...]
    # Content must NOT contain unescaped '[' or ']' brackets.
    inner_odp = re.compile(r"\[(Assignment|Selection(?:\s*\([^\]\[]+\))?):\s*([^\[\]]+)\]")
    params = []
    p_count = 0

    # Iteratively substitute innermost ODPs until none remain (up to 50 iterations)
    for _ in range(50):
        m = inner_odp.search(text)
        if not m:
            break
        p_count += 1
        p_type = m.group(1)
        raw_desc = m.group(2)
        clean_desc = re.sub(r"\s+", " ", raw_desc).strip()
        pid = f"{prefix}_prm_{p_count}"

        param = {
            "id": pid,
            "label": clean_desc
        }
        if "Selection" in p_type:
            raw_choices = [re.sub(r"\s+", " ", c).strip() for c in clean_desc.split(";")]
            choices = [c for c in raw_choices if c]
            if not choices:
                choices = [clean_desc]
            how_many = "one-or-more" if ("1 or more" in p_type or "one or more" in p_type) else "one"
            param["select"] = {"how-many": how_many, "choice": choices}

        params.append(param)
        text = text[:m.start()] + f"{{{{ insert: param, {pid} }}}}" + text[m.end():]

    return text, params

def parse_statement_parts(stmt_text: str, norm_id: str):
    lines = [l.strip() for l in stmt_text.split("\n") if l.strip()]
    if not lines:
        return []

    item_pattern = re.compile(r"^(?:([A-Z]{1,2})\.|\s*(\d+)\.|\s*([a-z])\.|\s*(\d+)\))\s+(.*)$")
    has_items = any(item_pattern.match(l) for l in lines)

    if not has_items:
        return [{
            "id": f"{norm_id}_smt",
            "name": "statement",
            "prose": " ".join(lines)
        }]

    root_part = {
        "id": f"{norm_id}_smt",
        "name": "statement",
        "parts": []
    }

    stack = [(0, root_part)]

    for line in lines:
        m = item_pattern.match(line)
        if m:
            lvl1, lvl2, lvl3, lvl4 = m.group(1), m.group(2), m.group(3), m.group(4)
            if lvl1:
                cur_lvl = 1
                label = lvl1 + "."
                raw_id_suffix = lvl1.lower()
                prose = m.group(5)
            elif lvl2:
                cur_lvl = 2
                label = lvl2 + "."
                raw_id_suffix = lvl2
                prose = m.group(5)
            elif lvl3:
                cur_lvl = 3
                label = lvl3 + "."
                raw_id_suffix = lvl3
                prose = m.group(5)
            elif lvl4:
                cur_lvl = 4
                label = lvl4 + ")"
                raw_id_suffix = lvl4
                prose = m.group(5)

            while len(stack) > 1 and stack[-1][0] >= cur_lvl:
                stack.pop()

            parent_part = stack[-1][1]
            parent_id = parent_part["id"]
            part_id = f"{parent_id}.{raw_id_suffix}"

            new_part = {
                "id": part_id,
                "name": "item",
                "props": [{"name": "label", "value": label}],
                "prose": prose
            }
            if lvl1 and len(lvl1) > 1: # AA, BB (Canadian specific)
                new_part["props"].append({"name": "jurisdiction", "value": "ca"})
                new_part["props"].append({"name": "sovereign-specification", "value": "true"})

            if "parts" not in parent_part:
                parent_part["parts"] = []
            parent_part["parts"].append(new_part)
            stack.append((cur_lvl, new_part))
        else:
            if len(stack) > 1:
                stack[-1][1]["prose"] += " " + line
            else:
                if "prose" not in root_part:
                    root_part["prose"] = line
                else:
                    root_part["prose"] += " " + line

    return [root_part]

def parse_enhancement(enh_match, next_enh_start, enh_text, base_norm_id, base_raw_id):
    enh_num_str = enh_match.group(1).strip()
    enh_num = int(enh_num_str)
    enh_title_part1 = enh_match.group(2).strip()
    enh_title_part2 = enh_match.group(3).strip() if enh_match.group(3) else ""

    if enh_title_part2:
        title = enh_title_part2
    else:
        if ":" in enh_title_part1:
            title = enh_title_part1.split(":", 1)[1].strip()
        elif "|" in enh_title_part1:
            title = enh_title_part1.split("|", 1)[1].strip()
        else:
            title = enh_title_part1

    title = re.sub(r"\s+", " ", title).strip()
    norm_id = f"{base_norm_id}.{enh_num}"
    label = re.sub(r"\s+", " ", f"{base_raw_id}({enh_num_str})").strip()
    is_canadian = enh_num >= 400

    enh_body = enh_text[enh_match.end():next_enh_start].strip()

    # Extract sections
    disc_idx = enh_body.find("Discussion:")
    gc_disc_idx = enh_body.find("GC discussion:")
    rel_idx = enh_body.find("Related controls and activities:")
    if rel_idx == -1:
        rel_idx = enh_body.find("Related controls:")
    ref_idx = enh_body.find("References:")

    stmt_end = len(enh_body)
    for pos in [disc_idx, gc_disc_idx, rel_idx, ref_idx]:
        if pos != -1 and pos < stmt_end:
            stmt_end = pos

    raw_stmt_text = enh_body[:stmt_end].strip()

    disc_text = ""
    if disc_idx != -1:
        d_end = len(enh_body)
        for pos in [gc_disc_idx, rel_idx, ref_idx]:
            if pos != -1 and pos > disc_idx and pos < d_end:
                d_end = pos
        disc_text = enh_body[disc_idx + len("Discussion:"):d_end].strip()

    gc_disc_text = ""
    if gc_disc_idx != -1:
        gcd_end = len(enh_body)
        for pos in [rel_idx, ref_idx]:
            if pos != -1 and pos > gc_disc_idx and pos < gcd_end:
                gcd_end = pos
        gc_disc_text = enh_body[gc_disc_idx + len("GC discussion:"):gcd_end].strip()

    rel_text = ""
    if rel_idx != -1:
        r_end = ref_idx if ref_idx != -1 and ref_idx > rel_idx else len(enh_body)
        raw_r = enh_body[rel_idx:r_end].strip()
        rel_text = re.sub(r"^Related controls(?: and activities)?:\s*", "", raw_r).strip()

    ref_text = ""
    if ref_idx != -1:
        raw_rf = enh_body[ref_idx:].strip()
        ref_text = re.sub(r"^References:\s*", "", raw_rf).strip()

    # Extract parameters from statement
    trans_stmt, params = extract_params(raw_stmt_text, norm_id)

    # Hierarchical statement decomposition
    parts = parse_statement_parts(trans_stmt, norm_id)

    if disc_text and not disc_text.lower().startswith("none"):
        parts.append({
            "id": f"{norm_id}_gdn",
            "name": "guidance",
            "prose": disc_text
        })
    if gc_disc_text and not gc_disc_text.lower().startswith("none"):
        parts.append({
            "id": f"{norm_id}_gc_gdn",
            "name": "guidance",
            "title": "GC Discussion",
            "props": [
                {"name": "jurisdiction", "value": "ca"},
                {"name": "sovereign-guidance", "value": "true"}
            ],
            "prose": gc_disc_text
        })

    links = []
    if rel_text:
        links.extend(parse_related(rel_text))
    if ref_text:
        links.extend(parse_references(ref_text))

    props = [
        {"name": "label", "value": label},
        {"name": "sort-id", "value": f"{base_norm_id}.{enh_num:03d}"}
    ]
    if is_canadian:
        props.append({"name": "jurisdiction", "value": "ca"})
        props.append({"name": "sovereign-control", "value": "true"})

    enh_ctrl = {
        "id": norm_id,
        "class": "enhancement",
        "title": title,
        "props": props,
        "parts": parts
    }
    if params:
        enh_ctrl["params"] = params
    if links:
        enh_ctrl["links"] = links

    return enh_ctrl

def parse_control_body(base_norm_id, base_raw_id, title, kind, body):
    title = re.sub(r"\s+", " ", title).strip()
    base_raw_id = re.sub(r"\s+", " ", base_raw_id).strip()
    is_withdrawn = kind.startswith("withdrawn")

    # Check if this base control is a Canadian sovereign control (>= 400)
    num_part = re.search(r"\d+", base_norm_id)
    is_canadian = False
    if num_part and int(num_part.group(0)) >= 400:
        is_canadian = True

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

    raw_stmt_text = body[:stmt_end].strip()

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
        r_end = enh_idx if enh_idx != -1 else (ref_idx if ref_idx != -1 else len(body))
        raw_r = body[rel_idx:r_end].strip()
        rel_text = re.sub(r"^Related controls(?: and activities)?:\s*", "", raw_r).strip()

    ref_text = ""
    if ref_idx != -1:
        raw_rf = body[ref_idx:].strip()
        ref_text = re.sub(r"^References:\s*", "", raw_rf).strip()

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
    params = []
    if is_withdrawn:
        parts.append({
            "id": f"{base_norm_id}_wth",
            "name": "withdrawn",
            "prose": raw_stmt_text or "This control has been withdrawn."
        })
    else:
        # Extract ODP parameters
        trans_stmt, params = extract_params(raw_stmt_text, base_norm_id)

        # Decompose statement into hierarchical items
        parts = parse_statement_parts(trans_stmt, base_norm_id)

        if disc_text and not disc_text.lower().startswith("none"):
            parts.append({
                "id": f"{base_norm_id}_gdn",
                "name": "guidance",
                "prose": disc_text
            })
        if gc_disc_text and not gc_disc_text.lower().startswith("none"):
            parts.append({
                "id": f"{base_norm_id}_gc_gdn",
                "name": "guidance",
                "title": "GC Discussion",
                "props": [
                    {"name": "jurisdiction", "value": "ca"},
                    {"name": "sovereign-guidance", "value": "true"}
                ],
                "prose": gc_disc_text
            })

    props = [
        {"name": "label", "value": base_raw_id},
        {"name": "sort-id", "value": base_norm_id},
        {"name": "status", "value": "withdrawn" if is_withdrawn else "active"}
    ]
    if not is_withdrawn:
        props.append({"name": "type", "value": kind})
    if is_canadian:
        props.append({"name": "jurisdiction", "value": "ca"})
        props.append({"name": "sovereign-control", "value": "true"})

    ctrl_obj = {
        "id": base_norm_id,
        "class": "activity" if kind == "activity" else "control",
        "title": title,
        "props": props,
        "parts": parts
    }
    if params:
        ctrl_obj["params"] = params

    links = []
    if rel_text:
        links.extend(parse_related(rel_text))
    if ref_text:
        links.extend(parse_references(ref_text))
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
    total_params = 0

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
        enh_count = len(ctrl.get("controls", []))
        total_enhancements += enh_count

        # Count params
        total_params += len(ctrl.get("params", []))
        for sc in ctrl.get("controls", []):
            total_params += len(sc.get("params", []))

        fam_code = fam_upper.lower()
        if fam_code in controls_by_family:
            controls_by_family[fam_code].append(ctrl)
        else:
            print(f"Warning: Unknown family code '{fam_code}' for control {raw_id}")

    groups = []
    total_base_controls = 0
    for fam_prefix, fam_code, fam_title in FAMILY_DEFINITIONS:
        fam_ctrls = controls_by_family[fam_code]
        total_base_controls += len(fam_ctrls)
        group = {
            "id": fam_code,
            "class": "family",
            "title": fam_title,
            "props": [
                {"name": "label", "value": fam_prefix}
            ],
            "controls": fam_ctrls
        }
        groups.append(group)

    print(f"Assembled {len(groups)} families.")
    print(f"Total base controls: {total_base_controls}")
    print(f"Total control enhancements: {total_enhancements}")
    print(f"Total controls + enhancements: {total_base_controls + total_enhancements}")
    print(f"Total extracted ODP parameters: {total_params}")

    party_uuid = make_uuid("party:cccs")

    catalog = {
        "catalog": {
            "uuid": make_uuid("catalog:cccs-itsp-10-033"),
            "metadata": {
                "title": "CCCS ITSP.10.033: Security and Privacy Controls and Assurance Activities Catalogue",
                "published": "2026-03-31T00:00:00Z",
                "last-modified": "2026-03-31T00:00:00Z",
                "version": "1.0 (2026-03-31)",
                "oscal-version": "1.2.3",
                "props": [
                    {"name": "jurisdiction", "value": "ca"},
                    {"name": "standard-id", "value": "ITSP.10.033"},
                    {"name": "isbn", "value": "978-0-660-76132-9"},
                    {"name": "cat-number", "value": "D97-3/10-033-2026E-PDF"},
                    {"name": "supersedes", "value": "ITSG-33 Annex 3A - Security control catalogue"},
                    {"name": "alignment", "value": "NIST SP 800-53 Rev. 5"}
                ],
                "roles": [
                    {"id": "creator", "title": "Document Creator / Author"},
                    {"id": "publisher", "title": "Publishing Authority"}
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
                    },
                    {
                        "role-id": "publisher",
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
                        "uuid": make_uuid("resource:itsp.10.033-01.pdf"),
                        "title": "ITSP.10.033-01: Suggested Organizational Security and Privacy Control and Activity Profile - Medium Impact (PDF)",
                        "citation": {
                            "text": "Canadian Centre for Cyber Security. (2026). Suggested organizational security and privacy control and activity profile - Medium impact (ITSP.10.033-01). Communications Security Establishment, Ottawa."
                        },
                        "rlinks": [
                            {"href": "https://publications.gc.ca/collections/collection_2026/cstc-csec/D97-3-10-033-01-2026-eng.pdf", "media-type": "application/pdf"}
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

    size_mb = os.path.getsize(OUTPUT_FILE) / (1024 * 1024)
    print(f"\nSuccessfully generated OSCAL 1.2.3 catalog: {OUTPUT_FILE} ({size_mb:.2f} MB)")

if __name__ == "__main__":
    build_oscal_catalog()
