# Framework semantic validation scope

Framework production enforces OSCAL **1.2.3 JSON Schema** and the separately
checksum-pinned **NIST oscal-cli 1.0.3**, whose **liboscal-java 3.0.3** bindings and
embedded schema identifiers implement OSCAL **1.1.2**. Its successful execution
does not establish general OSCAL 1.2.3 semantic coverage. The
[official CLI release notes](https://github.com/usnistgov/oscal-cli/releases/tag/v1.0.3)
identify its OSCAL 1.1.2 model update.

`scripts/oscal-semantic-scope.py` is an additional applicability gate for the
35 framework catalogs and one PBMM profile. It does not replace either validator.
It requires explicit OSCAL 1.2.3 metadata and rejects structured `hashes`,
`locations`, `location-uuids`, `resource-fragment`, `links`, `with-child-controls`,
and properties named `status` in any namespace. Presence is rejected even if
empty; prose mentioning these words is allowed. These are deliberately broad
framework producer restrictions, not restrictions on general server OSCAL input.
Adding one requires a reviewed compatible semantic producer and policy change;
an older CLI pass alone cannot admit it.

The deterministic `semantic-scope.json` receipt records each of the 36 exact
paths, SHA-256 digests and sizes, affected-feature counts, pinned tool/library/model
versions and digests, and official tagged model sources. Admission must replay
the guard against delivered JSON and compare the complete receipt. The strict
CLI gate remains required after this scope check. No exception or newer CLI
version is invented.

## Model comparison and concrete residual error

The reviewed catalog/profile closure includes their metadata and control-common
models plus the shared allowed-values entity. Source versions are pinned to
[OSCAL 1.1.2](https://github.com/usnistgov/OSCAL/tree/v1.1.2/src/metaschema) and
[OSCAL 1.2.3](https://github.com/usnistgov/OSCAL/tree/v1.2.3/src/metaschema).
Compiled annotations were inspected in the actual CLI distribution, not inferred
from workflow configuration.

| Semantic change in 1.2.3 | Consequence for this producer |
| --- | --- |
| SHA-224/256/384/512 and SHA3 hex lengths corrected from 28/32/48/64 to 56/64/96/128 | The older CLI accepts malformed short hashes and rejects correct hashes. Hash fields require explicit coverage beyond that CLI. |
| Control statuses expanded; reserved/superseded may omit a statement | Older rules are more restrictive; status properties are outside this subset. |
| Catalog local related/required/incorporated/moved link reference failures become warnings | Existing CLI gate retains stricter enforcement; links are outside this subset. |
| Metadata link uniqueness adds resource-fragment and becomes a warning | Links and resource-fragment are outside this subset. |
| Location type/class values expanded and a location reference constraint removed | Location constructs are outside this subset. |
| with-child-controls definition moves to control-common with the same yes/no values | Excluded to keep the reviewed producer subset explicit. |

Most other changes assign constraint IDs, rename Metaschema datatypes without
changing their JSON representation, or update explanatory text. The generated
JSON still must pass the separately enforced 1.2.3 structural schema.

An actual host probe of the checksum-pinned CLI confirmed SHA-256 with 32 hex
characters passes (exit 0), while 64 characters fails (exit 1) against its compiled
32-character regex. The existing embedded OSCAL 1.2.3 schema validator also accepts
the 32-character fixture: the official schema's hash value is a string and does
not encode the algorithm-dependent Metaschema regex. This is a semantic gap, not
a claim that schema validation covers the corrected constraint.

The checked `/tmp/xoscal-site-final-v3/frameworks` output contains exactly 35
catalogs and one profile, all declaring 1.2.3, with none of the excluded constructs.
31 artifacts contain back-matter resources/rlinks; those permitted fields do not
contain hashes. The guard passes all exact current subjects. This is a scoped
compatibility result, not a proof of every 1.2.3 rule or of future changed outputs.

Local review evidence: `/tmp/xoscal-oscal-model-review/constraint-semantic-delta.json`,
`output-relevance.json`, `semantic-scope-actual36.json`, compiled binding dumps,
and the two `sha256-32.log`/`sha256-64.log` probes. Hosted signing is outside this review.
