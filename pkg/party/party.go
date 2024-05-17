package party

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"runtime"
	"time"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/signing"
	"github.com/bnb-chain/tss-lib/v2/tss"

	"github.com/smiletrl/tss-lib-starter/pkg/constants"
	pb "github.com/smiletrl/tss-lib-starter/pkg/grpc/client"
)

// Step reference
// https://docs.bnbchain.org/docs/beaconchain/learn/threshold-signature-scheme/#step-1-init-tss

func NewPartyID(identifier constants.PartyIdentifier) *tss.PartyID {
	return tss.NewPartyID(identifier.ID, identifier.Moniker, new(big.Int).SetBytes([]byte(identifier.Key)))
}

type Party interface {
	// set local party id
	SetLocalID(identifier string)

	// gatether all shared parties
	GatherSharedParties()

	// prepare keygen parameter
	PrepareKeygen()

	// run keygen process
	Keygen() error

	// broadcast messages to all nodes (parties)
	MessageAll(ctx context.Context, msgType constants.MessageType, msg tss.Message)

	// send message to one specific node
	MessageNode(ctx context.Context, pid string, msgType constants.MessageType, msg tss.Message)

	// react on one message is received
	OnReceiveMessage(ctx context.Context, msgType constants.MessageType, fromPID string, isBroadcast bool, content []byte) error

	// sign the message
	Sign(ctx context.Context, msgData []byte) error

	// hold to wait for all keygen process finishes
	WaitForKeygen()
}

type party struct {
	// local party id
	id        *tss.PartyID
	preParams *keygen.LocalPreParams

	keygenParty  *keygen.LocalParty
	signingParty *signing.LocalParty

	// a mock key storage
	// in real prod env, key data could be saved within a file on the device/machine or a database like Mysql, Postgres.
	keyData *keygen.LocalPartySaveData

	// all party ids within this round
	partyIDMap map[string]*tss.PartyID

	pIDs tss.SortedPartyIDs

	// key finish chan
	keyFinish chan struct{}

	// sign finish chan
	signFinish chan struct{}

	// p2p client
	client pb.Client
}

func NewParty(client pb.Client) Party {
	return &party{
		client:     client,
		keyFinish:  make(chan struct{}, 1),
		signFinish: make(chan struct{}, 1),
	}
}

func (p *party) GatherSharedParties() {
	// @todo as per https://docs.bnbchain.org/docs/beaconchain/learn/threshold-signature-scheme/#step-1-init-tss, this
	// step should probably be automatically done via message exchange among these nodes to gather all party id data in
	// this round.

	// Save all shared parties in one node's local state
	p.partyIDMap = make(map[string]*tss.PartyID)
	parties := make([]*tss.PartyID, constants.TestParticipants)
	for i, pi := range constants.TestPartyIdentifiers {
		parties[i] = NewPartyID(pi)
		p.partyIDMap[pi.ID] = parties[i]
	}

	p.pIDs = tss.SortPartyIDs(parties)
}

func (p *party) SetLocalID(identifier string) {
	var ok bool
	if p.id, ok = p.partyIDMap[identifier]; !ok {
		panic("unexpected identifier:" + identifier)
	}
	// set up party id for its grpc client
	p.client.WithPartyID(p.id)
}

func (p *party) PrepareKeygen() {
	preParams, err := keygen.GeneratePreParams(1 * time.Minute)
	if err != nil {
		panic("error runing keygen preparam:" + err.Error())
	}
	p.preParams = preParams
}

func (p *party) Keygen() error {
	pIDs := p.pIDs

	p2pCtx := tss.NewPeerContext(p.pIDs)

	errCh := make(chan *tss.Error, len(pIDs))
	outCh := make(chan tss.Message, len(pIDs))
	endCh := make(chan *keygen.LocalPartySaveData, len(pIDs))

	params := tss.NewParameters(tss.S256(), p2pCtx, p.id, len(pIDs), constants.TestThreshold)
	p.keygenParty = keygen.NewLocalParty(params, outCh, endCh).(*keygen.LocalParty)

	go func() {
		if err := p.keygenParty.Start(); err != nil {
			errCh <- err
		}
	}()

	for {
		log.Printf("Keygen ACTIVE GOROUTINES: %d\n", runtime.NumGoroutine())
		select {
		case err := <-errCh:
			log.Printf("Keygen err: %v\n", err)
		case msg := <-outCh:
			log.Printf("Keygen out msg: %+v", msg)
			dest := msg.GetTo()
			if dest == nil {
				// broadcast
				go p.MessageAll(context.TODO(), constants.MessageTypeKeygen, msg)
			} else {
				// point to point
				if dest[0].Index == msg.GetFrom().Index {
					return fmt.Errorf("party %d tried to send a message to itself (%d)", dest[0].Index, msg.GetFrom().Index)
				}
				go p.MessageNode(context.TODO(), dest[0].GetId(), constants.MessageTypeKeygen, msg)
			}
		case save := <-endCh:
			log.Printf("keygen save data done start")
			p.keyData = save
			p.keyFinish <- struct{}{}
			log.Printf("keygen save data done")
		}
	}

}

func (p *party) MessageAll(ctx context.Context, msgType constants.MessageType, msg tss.Message) {
	// grpc request to all parties, except the trigger node
	if err := p.client.BroadcastNodes(ctx, msgType, msg); err != nil {
		log.Printf("error broadcasting nodes: %v", err)
	}
}

func (p *party) MessageNode(ctx context.Context, pid string, msgType constants.MessageType, msg tss.Message) {
	// send grpc call to target node
	if err := p.client.ToNode(ctx, pid, msgType, msg); err != nil {
		log.Printf("error messaging node: %v", err)
	}
}

func (p *party) verifyParty(msgType constants.MessageType) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", err)
		}
	}()

	// find which local party to update, `keygen` or `signing`
	var party tss.Party
	switch msgType {
	case constants.MessageTypeKeygen:
		party = p.keygenParty
	case constants.MessageTypeSigning:
		party = p.signingParty
	default:
		return fmt.Errorf("unexpected msg type: %s", msgType)
	}

	// if party is not ready, this line will panic
	t := party.PartyID().Id
	if t != "" {
		return err
	}
	return err
}

func (p *party) OnReceiveMessage(ctx context.Context, msgType constants.MessageType, fromPID string, isBroadcast bool, content []byte) error {
	// temporary hack
	ve := p.verifyParty(msgType)
	for ve != nil {
		log.Printf("Party is not ready yet: %v", ve)
		time.Sleep(time.Second)
		ve = p.verifyParty(msgType)
	}

	// find which local party to update, `keygen` or `signing`
	var party tss.Party
	switch msgType {
	case constants.MessageTypeKeygen:
		party = p.keygenParty
	case constants.MessageTypeSigning:
		party = p.signingParty
	default:
		return fmt.Errorf("unexpected msg type: %s", msgType)
	}

	// do not send a message from this party back to itself
	if p.id.GetId() == fromPID {
		return nil
	}

	if party.PartyID().Id == fromPID {
		return nil
	}

	fromParty := p.partyIDMap[fromPID]

	// update local party
	ok, err := party.UpdateFromBytes(content, fromParty, isBroadcast)
	if err != nil {
		return fmt.Errorf("error updating from bytes at OnReceiveMessage: %w", err)
	}
	if !ok {
		return fmt.Errorf("updating from bytes at OnReceiveMessage fails")
	}
	return nil
}

func (p *party) Sign(ctx context.Context, msgData []byte) error {
	// ideally select testThreshold+1 parties instead of all parties to sign
	// signPIDs := p.pIDs
	signPIDs := make(tss.SortedPartyIDs, 0, len(constants.SelectedParties))
	for _, P := range p.pIDs {
		if _, ok := constants.SelectedParties[P.GetId()]; ok {
			signPIDs = append(signPIDs, P)
		}
	}

	// PHASE: signing
	p2pCtx := tss.NewPeerContext(signPIDs)

	errCh := make(chan *tss.Error, len(signPIDs))
	outCh := make(chan tss.Message, len(signPIDs))
	endCh := make(chan *common.SignatureData, len(signPIDs))

	// init the party
	params := tss.NewParameters(tss.S256(), p2pCtx, p.id, len(signPIDs), constants.TestThreshold)
	p.signingParty = signing.NewLocalParty(new(big.Int).SetBytes(msgData), params, *p.keyData, outCh, endCh, len(msgData)).(*signing.LocalParty)

	go func() {
		if err := p.signingParty.Start(); err != nil {
			errCh <- err
		}
	}()

	for {
		log.Printf("Signing ACTIVE GOROUTINES: %d\n", runtime.NumGoroutine())
		select {
		case err := <-errCh:
			log.Printf("Sign err: %v\n", err)
		case msg := <-outCh:
			log.Printf("Sign out msg: %+v", msg)
			dest := msg.GetTo()
			if dest == nil {
				// broadcast
				go p.MessageAll(context.TODO(), constants.MessageTypeSigning, msg)
			} else {
				if dest[0].Index == msg.GetFrom().Index {
					return fmt.Errorf("party %d tried to send a message to itself (%d)", dest[0].Index, msg.GetFrom().Index)
				}
				go p.MessageNode(context.TODO(), dest[0].GetId(), constants.MessageTypeSigning, msg)
			}
		case sigRaw := <-endCh:
			log.Printf("Signature raw data: %+v", sigRaw)

			// get the signature and verify it
			pkX, pkY := p.keyData.ECDSAPub.X(), p.keyData.ECDSAPub.Y()
			pk := ecdsa.PublicKey{
				Curve: tss.S256(),
				X:     pkX,
				Y:     pkY,
			}

			// valid := ecdsa.VerifyASN1(&pk, new(big.Int).SetBytes(msgData).Bytes(), save.Signature)
			valid := ecdsa.Verify(&pk, msgData, new(big.Int).SetBytes(sigRaw.R), new(big.Int).SetBytes(sigRaw.S))
			log.Printf("Signature verify result: [%+v]\n", valid)
		}
	}
}

func (p *party) WaitForKeygen() {
	<-p.keyFinish
}
