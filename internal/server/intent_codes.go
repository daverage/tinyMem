package server

const (
	enforcementCodeModeCompliance = "MODE_COMPLIANCE"
	enforcementCodeModeTooLow     = "MODE_TOO_LOW"
	enforcementCodeModeNotSet     = "MODE_NOT_SET"
	enforcementCodeStrictRequired = "STRICT_REQUIRED"
	enforcementCodeFactEvidence   = "FACT_EVIDENCE_REQUIRED"
	enforcementCodeClaimViolation = "CLAIM_WITHOUT_ENFORCEMENT"
	enforcementCodeModeUpdated    = "MODE_UPDATED"
	enforcementCodeRecallRequired      = "RECALL_REQUIRED"
	enforcementCodeArtifactPathEscape  = "ARTIFACT_PATH_ESCAPE"

	// Exposed aliases for enforcement reporting.
	EnforcementCodeModeCompliance      = enforcementCodeModeCompliance
	EnforcementCodeModeTooLow          = enforcementCodeModeTooLow
	EnforcementCodeModeNotSet          = enforcementCodeModeNotSet
	EnforcementCodeStrictRequired      = enforcementCodeStrictRequired
	EnforcementCodeFactEvidence        = enforcementCodeFactEvidence
	EnforcementCodeClaimViolation      = enforcementCodeClaimViolation
	EnforcementCodeModeUpdated         = enforcementCodeModeUpdated
	EnforcementCodeRecallRequired      = enforcementCodeRecallRequired
	EnforcementCodeArtifactPathEscape  = enforcementCodeArtifactPathEscape
)
