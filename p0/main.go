package main

import (
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/signing"
	"github.com/bnb-chain/tss-lib/v2/tss"
)

const (
	testParticipants = 3
	testThreshold    = 1
)

func main() {
	m := &mock{}
	m.Init()
	go m.Keygen()

	// wait until keygen is finished for all parties
	<-m.keyFinish

	go m.Sign([]byte("hello, world"))

	// hang the app
	end := make(chan string)
	<-end
}

// mock two parties to join the signing process
type mock struct {
	pIDs      tss.SortedPartyIDs
	preParams *keygen.LocalPreParams

	// a mock key storage
	// in real prod env, key data could be saved within a file on the device/machine or a database like Mysql, Postgres.
	keyData []*keygen.LocalPartySaveData

	// key finish chan
	keyFinish chan struct{}
}

func (m *mock) Init() {
	// init party ids and params
	parties := make([]*tss.PartyID, testParticipants)
	for i := 0; i < testParticipants; i++ {
		parties[i] = tss.NewPartyID(fmt.Sprintf("id-%d", i), "", big.NewInt(int64(i+1)))
	}

	m.pIDs = tss.SortPartyIDs(parties)

	preParams, err := keygen.GeneratePreParams(1 * time.Minute)
	if err != nil {
		panic("error runing keygen preparam:" + err.Error())
	}
	m.preParams = preParams
	m.keyData = make([]*keygen.LocalPartySaveData, testParticipants)
	m.keyFinish = make(chan struct{}, 1)
}

func (m *mock) Keygen() error {
	pIDs := m.pIDs

	p2pCtx := tss.NewPeerContext(m.pIDs)
	parties := make([]*keygen.LocalParty, 0, len(m.pIDs))

	errCh := make(chan *tss.Error, len(pIDs))
	outCh := make(chan tss.Message, len(pIDs))
	endCh := make(chan *keygen.LocalPartySaveData, len(pIDs))

	// init the parties keygen
	for i := 0; i < len(pIDs); i++ {
		var P *keygen.LocalParty
		params := tss.NewParameters(tss.S256(), p2pCtx, pIDs[i], len(pIDs), testThreshold)
		P = keygen.NewLocalParty(params, outCh, endCh).(*keygen.LocalParty)
		parties = append(parties, P)
		go func(P *keygen.LocalParty) {
			if err := P.Start(); err != nil {
				errCh <- err
			}
		}(P)
	}
	// @todo
	// - I would assume each party should be initialized in separate Go app, instead of a loop here within one Go app
	// - add the communication between parties, including broadcast, and point to point channel via protobuf, instead
	//   of the global outCh, endCh here.
	// - save the keygen-save-data to each party's own storage engine once the process is done.
	var ended int32
keygen:
	for {
		log.Printf("ACTIVE GOROUTINES: %d\n", runtime.NumGoroutine())
		select {
		case err := <-errCh:
			log.Printf("keygen err: %v\n", err)
			break keygen
		case msg := <-outCh:
			dest := msg.GetTo()
			if dest == nil {
				// broadcast
				for _, P := range parties {
					if P.PartyID().Index == msg.GetFrom().Index {
						continue
					}
					go m.partyUpdate(P, msg, errCh)
				}
			} else {
				// point to point
				if dest[0].Index == msg.GetFrom().Index {
					return fmt.Errorf("party %d tried to send a message to itself (%d)", dest[0].Index, msg.GetFrom().Index)

				}
				go m.partyUpdate(parties[dest[0].Index], msg, errCh)
			}
		case save := <-endCh:
			index, err := save.OriginalIndex()
			if err != nil {
				return fmt.Errorf("keygen save data index error: %w", err)
			}
			m.keyData[index] = save

			atomic.AddInt32(&ended, 1)
			if atomic.LoadInt32(&ended) == int32(len(pIDs)) {
				log.Printf("Done. Received save data from %d participants at keygen\n", ended)
				m.keyFinish <- struct{}{}
			}
		}
	}
	return nil
}

func (m *mock) Sign(msgData []byte) error {

	// @todo, ideally select testThreshold+1 parties instead of all parties to sign
	keys := m.keyData
	signPIDs := m.pIDs

	// PHASE: signing
	p2pCtx := tss.NewPeerContext(signPIDs)
	parties := make([]*signing.LocalParty, 0, len(signPIDs))

	errCh := make(chan *tss.Error, len(signPIDs))
	outCh := make(chan tss.Message, len(signPIDs))
	endCh := make(chan *common.SignatureData, len(signPIDs))

	// init the parties
	for i := 0; i < len(signPIDs); i++ {
		params := tss.NewParameters(tss.S256(), p2pCtx, signPIDs[i], len(signPIDs), testThreshold)
		P := signing.NewLocalParty(new(big.Int).SetBytes(msgData), params, *keys[i], outCh, endCh, len(msgData)).(*signing.LocalParty)
		parties = append(parties, P)
		go func(P *signing.LocalParty) {
			if err := P.Start(); err != nil {
				errCh <- err
			}
		}(P)
	}

	// @todo
	// - initialize each party at its own Go app
	// - add the communication between parties, including broadcast, and point to point channel via protobuf, instead
	//   of the global outCh, endCh here.
	// - generate public key from final signature data and verify the signature data.
	var ended int32
signing:
	for {
		select {
		case err := <-errCh:
			log.Printf("sign err: %v\n", err)
			break signing
		case msg := <-outCh:
			dest := msg.GetTo()
			if dest == nil {
				for _, P := range parties {
					if P.PartyID().Index == msg.GetFrom().Index {
						continue
					}
					go m.partyUpdate(P, msg, errCh)
				}
			} else {
				if dest[0].Index == msg.GetFrom().Index {
					return fmt.Errorf("party %d tried to send a message to itself (%d)", dest[0].Index, msg.GetFrom().Index)
				}
				go m.partyUpdate(parties[dest[0].Index], msg, errCh)
			}
		case save := <-endCh:
			atomic.AddInt32(&ended, 1)
			if atomic.LoadInt32(&ended) == int32(len(signPIDs)) {
				log.Printf("Done. Received save data from %d participants at signing\n", ended)

				pkX, pkY := keys[0].ECDSAPub.X(), keys[0].ECDSAPub.Y()
				pk := ecdsa.PublicKey{
					Curve: tss.S256(),
					X:     pkX,
					Y:     pkY,
				}

				// valid := ecdsa.VerifyASN1(&pk, new(big.Int).SetBytes(msgData).Bytes(), save.Signature)
				valid := ecdsa.Verify(&pk, msgData, new(big.Int).SetBytes(save.R), new(big.Int).SetBytes(save.S))
				log.Printf("Signature verify result: [%+v]\n", valid)
			}
		}
	}
	return nil
}

func (m *mock) partyUpdate(party tss.Party, msg tss.Message, errCh chan<- *tss.Error) {
	// do not send a message from this party back to itself
	if party.PartyID() == msg.GetFrom() {
		return
	}
	bz, _, err := msg.WireBytes()
	if err != nil {
		errCh <- party.WrapError(err)
		return
	}
	pMsg, err := tss.ParseWireMessage(bz, msg.GetFrom(), msg.IsBroadcast())
	if err != nil {
		errCh <- party.WrapError(err)
		return
	}
	if _, err := party.Update(pMsg); err != nil {
		errCh <- err
	}
}
