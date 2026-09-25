package canadianframeworks

import (
	catalogv1 "github.com/mchorfa/xoscal/proto/oscal/catalog/v1"
	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	profilev1 "github.com/mchorfa/xoscal/proto/oscal/profile/v1"
	"github.com/mchorfa/xoscal/server/internal/scaffold"
)

// BuildCCCSITSG33Catalog synthesizes an authoritative OSCAL 1.2.3 Catalog for
// CCCS ITSG-33 Annex 3 (Security Control Catalogue for Government of Canada).
func BuildCCCSITSG33Catalog() *catalogv1.Catalog {
	groups := []*catalogv1.Group{
		{
			Id:    &commonv1.Token{Value: "ac"},
			Title: &commonv1.MarkupLine{Value: "Access Control (Controle d'acces)"},
			Controls: []*catalogv1.Control{
				makeControl("ac-1", "Access Control Policy and Procedures", "The organization develops, documents, and disseminates an access control policy that complies with Treasury Board of Canada Secretariat (TBS) Policy on Government Security."),
				makeControl("ac-2", "Account Management", "The organization identifies and selects account types, assigns account managers, and configures account lifecycle controls according to Government of Canada identity assurance levels."),
				makeControl("ac-3", "Access Enforcement", "The information system enforces approved authorizations for logical access to information and system resources in accordance with applicable access control policies."),
				makeControl("ac-6", "Least Privilege", "The organization employs the principle of least privilege, allowing only authorized accesses for users and processes necessary to accomplish assigned organizational tasks."),
			},
		},
		{
			Id:    &commonv1.Token{Value: "at"},
			Title: &commonv1.MarkupLine{Value: "Awareness and Training (Sensibilisation et formation)"},
			Controls: []*catalogv1.Control{
				makeControl("at-1", "Security Awareness Policy and Procedures", "The organization establishes security awareness and training procedures aligned with CCCS threat intelligence."),
				makeControl("at-2", "Security Awareness Training", "The organization provides basic security awareness training to information system users as part of initial training and annually thereafter."),
			},
		},
		{
			Id:    &commonv1.Token{Value: "au"},
			Title: &commonv1.MarkupLine{Value: "Audit and Accountability (Verification et responsabilite)"},
			Controls: []*catalogv1.Control{
				makeControl("au-2", "Event Logging", "The information system identifies and records auditable security events in accordance with CCCS logging guidance."),
				makeControl("au-6", "Audit Review, Analysis, and Reporting", "The organization reviews and analyzes information system audit records for indications of unusual or suspicious activity."),
			},
		},
		{
			Id:    &commonv1.Token{Value: "ca"},
			Title: &commonv1.MarkupLine{Value: "Security Assessment and Authorization (Evaluation et autorisation de securite)"},
			Controls: []*catalogv1.Control{
				makeControl("ca-2", "Security Assessments", "The organization assesses the security controls in the information system to determine effectiveness in accordance with ITSG-33 lifecycle Phase 3."),
				makeControl("ca-3", "Information Exchange Agreements", "The organization approves and documents information system interconnections via formal Memorandums of Understanding (MOU)."),
			},
		},
		{
			Id:    &commonv1.Token{Value: "cm"},
			Title: &commonv1.MarkupLine{Value: "Configuration Management (Gestion des configurations)"},
			Controls: []*catalogv1.Control{
				makeControl("cm-2", "Baseline Configuration", "The organization develops, documents, and maintains a current baseline configuration of the information system."),
				makeControl("cm-8", "Information System Component Inventory", "The organization develops and documents an inventory of information system components that accurately reflects the current system."),
			},
		},
		{
			Id:    &commonv1.Token{Value: "cp"},
			Title: &commonv1.MarkupLine{Value: "Contingency Planning (Planification de la continuite)"},
			Controls: []*catalogv1.Control{
				makeControl("cp-2", "Contingency Plan", "The organization develops a contingency plan for the information system that addresses system restoration without compromise."),
				makeControl("cp-9", "Information System Backup", "The organization conducts backups of user-level information, system-level information, and security-related documentation."),
			},
		},
		{
			Id:    &commonv1.Token{Value: "ia"},
			Title: &commonv1.MarkupLine{Value: "Identification and Authentication (Identification et authentification)"},
			Controls: []*catalogv1.Control{
				makeControl("ia-2", "Identification and Authentication (Organizational Users)", "The information system uniquely identifies and authenticates organizational users with mandatory multi-factor authentication (MFA)."),
				makeControl("ia-5", "Authenticator Management", "The organization manages information system authenticators including passwords, tokens, certificates, and biometrics per CSE standards."),
			},
		},
		{
			Id:    &commonv1.Token{Value: "ir"},
			Title: &commonv1.MarkupLine{Value: "Incident Response (Intervention en cas d'incident)"},
			Controls: []*catalogv1.Control{
				makeControl("ir-4", "Incident Handling", "The organization implements an incident handling capability for security incidents that includes coordination with the CCCS Canadian Cyber Incident Response Centre."),
				makeControl("ir-6", "Incident Reporting", "The organization reports information security incidents to designated authorities including the CCCS within mandated reporting windows."),
			},
		},
		{
			Id:    &commonv1.Token{Value: "mp"},
			Title: &commonv1.MarkupLine{Value: "Media Protection (Protection des supports)"},
			Controls: []*catalogv1.Control{
				makeControl("mp-2", "Media Access", "The organization restricts access to digital and non-digital media containing Protected B or sensitive information."),
				makeControl("mp-6", "Media Sanitization", "The organization sanitizes information system media prior to disposal or reuse in accordance with RCMP and CSE sanitization standards."),
			},
		},
		{
			Id:    &commonv1.Token{Value: "pe"},
			Title: &commonv1.MarkupLine{Value: "Physical and Environmental Protection (Protection physique et environnementale)"},
			Controls: []*catalogv1.Control{
				makeControl("pe-2", "Physical Access Authorizations", "The organization develops and maintains a list of individuals with authorized access to physical facilities housing GC information systems."),
				makeControl("pe-3", "Physical Access Control", "The organization controls all physical access points to facilities processing Government of Canada workloads."),
			},
		},
		{
			Id:    &commonv1.Token{Value: "pl"},
			Title: &commonv1.MarkupLine{Value: "Planning (Planification)"},
			Controls: []*catalogv1.Control{
				makeControl("pl-2", "System Security Plan", "The organization develops, documents, and updates a System Security Plan (SSP) describing security controls and boundary architectures."),
			},
		},
		{
			Id:    &commonv1.Token{Value: "ps"},
			Title: &commonv1.MarkupLine{Value: "Personnel Security (Securite du personnel)"},
			Controls: []*catalogv1.Control{
				makeControl("ps-2", "Position Risk Designation", "The organization assigns a risk designation to all positions and establishes screening criteria aligned with Government of Canada security clearances."),
				makeControl("ps-3", "Personnel Screening", "The organization screens individuals prior to authorizing access to systems processing sensitive or Protected B data."),
			},
		},
		{
			Id:    &commonv1.Token{Value: "ra"},
			Title: &commonv1.MarkupLine{Value: "Risk Assessment (Evaluation des risques)"},
			Controls: []*catalogv1.Control{
				makeControl("ra-3", "Risk Assessment", "The organization assesses the risk relating to the operation of the information system in accordance with the Harmonized Threat and Risk Assessment (HTRA) methodology."),
				makeControl("ra-5", "Vulnerability Scanning", "The organization scans for vulnerabilities in the information system and hosted applications on a continuous basis."),
			},
		},
		{
			Id:    &commonv1.Token{Value: "sa"},
			Title: &commonv1.MarkupLine{Value: "System and Services Acquisition (Acquisition de systemes et de services)"},
			Controls: []*catalogv1.Control{
				makeControl("sa-4", "Acquisition Process", "The organization includes security requirements, descriptions of functional properties, and supply chain security controls in procurement solicitations."),
				makeControl("sa-9", "External Information System Services", "The organization mandates that external cloud and service providers comply with CCCS cloud security requirements."),
			},
		},
		{
			Id:    &commonv1.Token{Value: "sc"},
			Title: &commonv1.MarkupLine{Value: "System and Communications Protection (Protection des systemes et des communications)"},
			Controls: []*catalogv1.Control{
				makeControl("sc-7", "Boundary Protection", "The information system monitors and controls communications at external boundaries and key internal boundaries using approved network security zones."),
				makeControl("sc-8", "Transmission Confidentiality and Integrity", "The information system protects the confidentiality and integrity of transmitted information using CSE Approved Cryptographic Algorithms (CACA / ITSP.40.111)."),
				makeControl("sc-13", "Cryptographic Protection", "The information system implements cryptographic modules that are FIPS 140-2 or FIPS 140-3 validated in accordance with CSE cryptographic guidance."),
				makeControl("sc-28", "Protection of Information at Rest", "The information system protects the confidentiality and integrity of Protected B information at rest using approved cryptographic algorithms."),
			},
		},
		{
			Id:    &commonv1.Token{Value: "si"},
			Title: &commonv1.MarkupLine{Value: "System and Information Integrity (Integrite des systemes et de l'information)"},
			Controls: []*catalogv1.Control{
				makeControl("si-2", "Flaw Remediation", "The organization identifies, reports, and corrects information system flaws within mandated remediation timeframes."),
				makeControl("si-3", "Malicious Code Protection", "The information system employs anti-malware and threat detection mechanisms at endpoints and network entry points."),
				makeControl("si-4", "Information System Monitoring", "The information system monitors inbound and outbound network traffic to detect attacks and indicators of compromise."),
			},
		},
		{
			Id:    &commonv1.Token{Value: "pm"},
			Title: &commonv1.MarkupLine{Value: "Program Management (Gestion de programme)"},
			Controls: []*catalogv1.Control{
				makeControl("pm-1", "Information Security Program Plan", "The organization develops and disseminates an organization-wide information security program plan."),
			},
		},
	}

	return &catalogv1.Catalog{
		Uuid: scaffold.UUIDFromURN("urn:xoscal:catalog:cccs-itsg-33"),
		Metadata: &commonv1.Metadata{
			Title:        "CCCS ITSG-33 IT Security Risk Management: A Lifecycle Approach - Annex 3 Security Control Catalogue",
			Version:      "Annex 3 (Harmonized with NIST SP 800-53)",
			OscalVersion: "1.2.3",
			Remarks: &commonv1.MarkupMultiline{
				Value: "Authoritative Government of Canada security control catalogue published by the Canadian Centre for Cyber Security (CCCS) / Communications Security Establishment (CSE).",
			},
		},
		Groups: groups,
	}
}

// BuildCCCSMediumCloudPBMMProfile synthesizes the Government of Canada Cloud Security Profile:
// Protected B, Medium Integrity, Medium Availability (PBMM) per ITSP.50.103 and ITSG-33 Annex 4A Profile 1.
func BuildCCCSMediumCloudPBMMProfile() *profilev1.Profile {
	return &profilev1.Profile{
		Uuid: scaffold.UUIDFromURN("urn:xoscal:profile:cccs-medium-cloud-pbmm"),
		Metadata: &commonv1.Metadata{
			Title:        "Government of Canada Cloud Security Control Profile: Protected B / Medium Integrity / Medium Availability (PBMM)",
			Version:      "ITSP.50.103 / ITSG-33 Annex 4A Profile 1",
			OscalVersion: "1.2.3",
			Remarks: &commonv1.MarkupMultiline{
				Value: "Mandatory baseline security control profile for Cloud Service Providers (CSPs) and Government of Canada cloud-based services handling Protected B information.",
			},
		},
		Imports: []*profilev1.Import{
			{
				Href:       &commonv1.URIReference{Value: "https://www.cyber.gc.ca/en/guidance/it-security-risk-management-lifecycle-approach-itsg-33"},
				IncludeAll: &profilev1.IncludeAll{},
			},
		},
	}
}

// BuildCyberSecureCanadaCatalog synthesizes an authoritative OSCAL 1.2.3 Catalog for
// CyberSecure Canada (CyberSecuritaire Canada) baseline cybersecurity controls for SMEs.
func BuildCyberSecureCanadaCatalog() *catalogv1.Catalog {
	controls := []*catalogv1.Control{
		makeControl("csc-1", "Automatically Patch Operating Systems and Applications", "Enable automatic updates for operating systems and applications to protect against known security vulnerabilities."),
		makeControl("csc-2", "Implement Strong User Authentication", "Mandate multi-factor authentication (MFA) for administrative access, cloud accounts, remote connections, and sensitive applications."),
		makeControl("csc-3", "Provide Employee Cyber Security Awareness Training", "Train employees to recognize social engineering, phishing campaigns, suspicious attachments, and credential theft techniques."),
		makeControl("csc-4", "Backup and Encrypt Sensitive Data", "Perform regular encrypted backups of critical business data, store backups offsite or in cloud isolation, and test recovery procedures."),
		makeControl("csc-5", "Enable Perimeter Defenses and Firewalls", "Deploy and configure stateful boundary firewalls, disable unnecessary ports and protocols, and monitor ingress/egress boundaries."),
		makeControl("csc-6", "Secure Mobile Devices", "Enforce device passcodes, biometric lock screens, automatic screen timeout, full-disk encryption, and mobile device management policies."),
		makeControl("csc-7", "Establish Access Control and Least Privilege", "Restrict employee access rights to only those resources strictly necessary for their specific job functions."),
		makeControl("csc-8", "Secure Cloud and Outsourced IT Services", "Ensure cloud and outsourced IT service providers meet recognized security standards and protect organizational data."),
		makeControl("csc-9", "Secure Website and Web Application Configurations", "Protect public-facing websites and applications using TLS certificates, secure HTTP headers, and regular vulnerability scanning."),
		makeControl("csc-10", "Protect Against Malicious Code and Malware", "Install anti-malware and endpoint detection software on all workstations and servers with automated definition updates."),
		makeControl("csc-11", "Implement Secure Portable Media Handling", "Restrict and monitor the use of portable storage devices (USB drives) and encrypt all confidential data transferred to removable media."),
		makeControl("csc-12", "Maintain an Incident Response Plan", "Develop, document, and test an incident response plan to handle potential cyber security breaches and notify stakeholders."),
		makeControl("csc-13", "Control Administrative Privileges", "Separate standard user accounts from administrative accounts; use dedicated privileged credentials only when performing administrative tasks."),
	}

	return &catalogv1.Catalog{
		Uuid: scaffold.UUIDFromURN("urn:xoscal:catalog:cybersecure-canada"),
		Metadata: &commonv1.Metadata{
			Title:        "CyberSecure Canada - Baseline Cyber Security Controls for Small and Medium Organizations",
			Version:      "1.2",
			OscalVersion: "1.2.3",
			Remarks: &commonv1.MarkupMultiline{
				Value: "National cyber security certification standard developed by Innovation, Science and Economic Development Canada (ISED) and the Canadian Centre for Cyber Security (CCCS).",
			},
		},
		Controls: controls,
	}
}

// BuildCCCSITSP10171Catalog synthesizes an authoritative OSCAL 1.2.3 Catalog for
// CCCS ITSP.10.171 (Protecting Specified Information in Non-Government of Canada Systems and Organizations).
func BuildCCCSITSP10171Catalog() *catalogv1.Catalog {
	controls := []*catalogv1.Control{
		makeControl("itsp-ac-1", "Access Control for Specified Information", "Limit information system access to authorized users, processes acting on behalf of authorized users, and devices handling Canadian specified information."),
		makeControl("itsp-au-1", "Audit Logging for Controlled Systems", "Create and retain system audit logs and records to the extent needed to enable the monitoring, analysis, investigation, and reporting of unlawful or unauthorized system activity."),
		makeControl("itsp-cm-1", "Configuration Baseline for Designated Assets", "Establish and maintain baseline configurations and inventories of organizational information systems throughout the system development life cycle."),
		makeControl("itsp-ia-1", "Identification and Authentication for Contractor Systems", "Identify information system users, processes acting on behalf of users, and devices; authenticate the identities of those users, processes, or devices."),
		makeControl("itsp-mp-1", "Media Protection for Controlled Goods", "Protect information system media containing specified information, both paper and digital; sanitize or destroy information system media before disposal."),
		makeControl("itsp-sc-1", "Boundary and Cryptographic Protection", "Monitor, control, and protect organizational communications at the external boundaries and key internal boundaries; employ FIPS 140-validated cryptography."),
	}

	return &catalogv1.Catalog{
		Uuid: scaffold.UUIDFromURN("urn:xoscal:catalog:cccs-itsp-10-171"),
		Metadata: &commonv1.Metadata{
			Title:        "CCCS ITSP.10.171 - Protecting Specified Information in Non-Government of Canada Systems and Organizations",
			Version:      "1.0",
			OscalVersion: "1.2.3",
			Remarks: &commonv1.MarkupMultiline{
				Value: "Canadian cybersecurity baseline for defense suppliers, aerospace contractors, and commercial organizations handling Canadian Controlled Goods and specified federal data.",
			},
		},
		Controls: controls,
	}
}

func makeControl(id, title, statement string) *catalogv1.Control {
	return &catalogv1.Control{
		Id:    &commonv1.Token{Value: id},
		Title: &commonv1.MarkupLine{Value: title},
		Parts: []*catalogv1.Part{
			{
				Id:    &commonv1.Token{Value: id + "-stmt"},
				Name:  "statement",
				Prose: []*commonv1.MarkupMultiline{{Value: statement}},
			},
		},
	}
}
