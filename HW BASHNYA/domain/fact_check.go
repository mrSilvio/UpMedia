package domain

import "time"

type FactCheckStatus string

const (
	CheckPending      FactCheckStatus = "pending"
	CheckConfirmed    FactCheckStatus = "confirmed"
	CheckContradicted FactCheckStatus = "contradicted"
	CheckInsufficient FactCheckStatus = "insufficient_data"
)

type Claim struct {
	ID           ID
	NewsItemID   ID
	SubmissionID ID
	Text         string
	CreatedAt    time.Time
}

type FactCheck struct {
	ID                    ID
	ClaimID               ID
	Status                FactCheckStatus
	SupportingEvidence    []Evidence
	ContradictingEvidence []Evidence
	Explanation           string
	CreatedAt             time.Time
	CompletedAt           *time.Time
}
