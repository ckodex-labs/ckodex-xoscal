import os
import json
import re
import uuid
import datetime
import pdfplumber

def normalize_id(fam, raw_id):
    fam = fam.strip().lower()
    raw_id = raw_id.strip()
    m_base = re.match(r"^(\d+)$", raw_id)
    if m_base:
        num = int(m_base.group(1))
        return f"{fam}-{num}"
    m_enh = re.match(r"^(\d+)\((\d+)\)$", raw_id)
    if m_enh:
        base_num = int(m_enh.group(1))
        enh_num = int(m_enh.group(2))
        return f"{fam}-{base_num}.{enh_num}"
    return f"{fam}-{raw_id.lower()}"

def main():
    print("Reading catalog to ensure reference validity...")
    catalog_path = "data/frameworks/cccs-itsp-10-033/catalog.json"
    if not os.path.exists(catalog_path):
        catalog_path = "server/internal/canadianframeworks/itsp10033_catalog.json"
    with open(catalog_path) as f:
        cat = json.load(f)
    
    catalog_ids = set()
    for g in cat["catalog"]["groups"]:
        for c in g.get("controls", []):
            catalog_ids.add(c["id"])
            for sc in c.get("controls", []):
                catalog_ids.add(sc["id"])
    
    print(f"Catalog contains {len(catalog_ids)} control IDs.")

    pdf_path = os.environ.get("ITSP_PROFILE_PDF_PATH", "data/sources/itsp.10.033-01-2026-eng.pdf")
    if not os.path.exists(pdf_path) and os.path.exists("itsp.10.033-01-2026-eng.pdf"):
        pdf_path = "itsp.10.033-01-2026-eng.pdf"

    print(f"Parsing ITSP.10.033-01 PDF Table 4 from {pdf_path}...")
    with pdfplumber.open(pdf_path) as pdf:
        curr_fam = ""
        selected_ids = []
        not_selected_ids = []
        withdrawn_ids = []
        param_settings = []
        alters = []

        rows = []
        for page_idx in range(15, len(pdf.pages)):
            p = pdf.pages[page_idx]
            tables = p.extract_tables()
            for t in tables:
                for r in t:
                    if len(r) >= 6:
                        cleaned = [c.replace("\n", " ").strip() if c else "" for c in r]
                        while len(cleaned) < 8:
                            cleaned.append("")
                        if cleaned[0] == "Family" or (cleaned[1] == "ID" and cleaned[2] == "Name"):
                            continue
                        if cleaned[4] in ("Control/", "Activity") and cleaned[5] in ("Suggested for this", "profile"):
                            continue
                        if not any(cleaned):
                            continue
                        if cleaned[1]:
                            if cleaned[0]:
                                curr_fam = cleaned[0]
                            cid = normalize_id(curr_fam, cleaned[1])
                            rows.append({"cid": cid, "cols": cleaned})
                        elif rows:
                            prev = rows[-1]["cols"]
                            for idx in range(2, 8):
                                if cleaned[idx]:
                                    prev[idx] = (prev[idx] + " " + cleaned[idx]).strip() if prev[idx] else cleaned[idx]

        for row in rows:
            cid = row["cid"]
            cleaned = row["cols"]
            name = cleaned[2]
            desc = cleaned[3]
            sel = cleaned[5]
            vals = cleaned[6]
            notes = cleaned[7]

            if sel.lower() == "selected":
                selected_ids.append(cid)
            elif "not selected" in sel.lower():
                not_selected_ids.append(cid)
            elif "withdrawn" in desc.lower() or "withdrawn" in name.lower() or sel == "NA":
                withdrawn_ids.append(cid)

            if vals and vals != "NA":
                param_id = f"{cid}_prm_1"
                param_settings.append({
                    "param-id": param_id,
                    "values": [vals]
                })

            if notes and notes != "NA":
                alter_entry = {
                    "control-id": cid,
                    "adds": [
                        {
                            "position": "after",
                            "parts": [
                                {
                                    "id": f"{cid}_profile-guidance",
                                    "name": "guidance",
                                    "prose": notes
                                }
                            ]
                        }
                    ]
                }
                alters.append(alter_entry)

    print(f"Extracted {len(selected_ids)} selected controls.")
    print(f"Extracted {len(param_settings)} parameter settings.")
    print(f"Extracted {len(alters)} alters (profile notes).")

    # Ensure deterministic deduplicated ordering
    unique_selected = []
    seen = set()
    for cid in selected_ids:
        if cid not in seen:
            seen.add(cid)
            unique_selected.append(cid)

    # Deterministic UUID for the profile
    profile_uuid = str(uuid.uuid5(uuid.NAMESPACE_DNS, "cccs.cyber.gc.ca/itsp.10.033-01/profile"))

    # Metadata
    metadata = {
        "title": "CCCS ITSP.10.033-01 Suggested Organizational Security and Privacy Control and Activity Profile - Medium Impact",
        "published": "2026-04-01T00:00:00Z",
        "last-modified": "2026-04-01T00:00:00Z",
        "version": "1.0 (Supersedes ITSG-33 Annex 4A Profile 1 PBMM)",
        "oscal-version": "1.2.3",
        "remarks": "Authoritative Government of Canada Medium Impact (Protected B / Medium Integrity / Medium Availability - PBMM) profile published by the Canadian Centre for Cyber Security (CCCS) / Communications Security Establishment (CSE). Supersedes ITSG-33 Annex 4A Profile 1.",
        "roles": [
            {
                "id": "creator",
                "title": "Document Creator"
            },
            {
                "id": "publisher",
                "title": "Publisher"
            }
        ],
        "parties": [
            {
                "uuid": str(uuid.uuid5(uuid.NAMESPACE_DNS, "cyber.gc.ca")),
                "type": "organization",
                "name": "Canadian Centre for Cyber Security",
                "short-name": "CCCS",
                "email-addresses": [
                    "contact@cyber.gc.ca"
                ],
                "telephone-numbers": [
                    {
                        "type": "office",
                        "number": "+1-833-292-3722"
                    }
                ],
                "addresses": [
                    {
                        "addr-lines": [
                            "P.O. Box 9703, Terminal"
                        ],
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
                "role-id": "publisher",
                "party-uuids": [
                    str(uuid.uuid5(uuid.NAMESPACE_DNS, "cyber.gc.ca"))
                ]
            }
        ]
    }

    profile_doc = {
        "profile": {
            "uuid": profile_uuid,
            "metadata": metadata,
            "imports": [
                {
                    "href": "catalog.json",
                    "include-controls": [
                        {
                            "with-ids": unique_selected
                        }
                    ]
                }
            ],
            "modify": {
                "set-parameters": param_settings,
                "alters": alters
            }
        }
    }

    output_path = "data/frameworks/cccs-itsp-10-033/profile.json"
    os.makedirs(os.path.dirname(output_path), exist_ok=True)
    with open(output_path, "w") as f:
        json.dump(profile_doc, f, indent=2)
    print(f"Wrote profile to {output_path}")

    embed_path = "server/internal/canadianframeworks/itsp10033_profile.json"
    if os.path.exists(os.path.dirname(embed_path)):
        with open(embed_path, "w") as f:
            json.dump(profile_doc, f, indent=2)
        print(f"Updated embedded profile in {embed_path}")

if __name__ == "__main__":
    main()
