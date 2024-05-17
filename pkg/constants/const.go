package constants

const (
	TestParticipants = 4
	TestThreshold    = 2
)

var (
	TestPartyIdentifiers = []PartyIdentifier{
		{
			ID:      "p1",
			Moniker: "tss1",
			Key:     "1",
		},
		{
			ID:      "p2",
			Moniker: "tss2",
			Key:     "2",
		},
		{
			ID:      "p3",
			Moniker: "tss3",
			Key:     "3",
		},
		{
			ID:      "p4",
			Moniker: "tss4",
			Key:     "4",
		},
	}

	TestGrpcHost = map[string][2]string{
		"p1": {"127.0.0.1", "50051"},
		"p2": {"127.0.0.1", "50052"},
		"p3": {"127.0.0.1", "50053"},
		"p4": {"127.0.0.1", "50054"},
	}

	// !Important The unselected party must come from the last one(i.e, p4, p3, etc) because of
	// this test env to use sorted parties
	SelectedParties = map[string]struct{}{
		"p1": {},
		"p2": {},
		"p3": {},
		"p4": {},
	}
)

type PartyIdentifier struct {
	ID, Moniker, Key string
}

var EnvPartyID string = "PARTY_ID"

var SignMessage string = "hey this is a test"

type MessageType string

const (
	MessageTypeKeygen  MessageType = "keygen"
	MessageTypeSigning MessageType = "signing"
)
