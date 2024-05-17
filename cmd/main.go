package cmd

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/smiletrl/tss-lib-starter/pkg/constants"
	pbClient "github.com/smiletrl/tss-lib-starter/pkg/grpc/client"
	pbServer "github.com/smiletrl/tss-lib-starter/pkg/grpc/server"
	"github.com/smiletrl/tss-lib-starter/pkg/party"
)

func Cmd(signMessage string) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	envPartyID := os.Getenv(constants.EnvPartyID)
	if envPartyID == "" {
		panic("env party id is not set yet")
	}

	// init pb clients
	client, err := pbClient.NewClient(constants.TestGrpcHost)
	if err != nil {
		panic("error initializing pb client:" + err.Error())
	}

	// init local party
	p := party.NewParty(client)

	// init pb server
	go func(p party.Party) {
		log.Println("grpc server starts")
		if err := pbServer.RegisterServer(envPartyID, p); err != nil {
			panic("error register server:" + err.Error())
		}
	}(p)

	p.GatherSharedParties()
	p.SetLocalID(envPartyID)

	log.Printf("prepare keygen")

	p.PrepareKeygen()

	// start keygen process
	go p.Keygen()
	log.Printf("wait for keygen")

	// wait for keygen process finished
	p.WaitForKeygen()
	log.Printf("keygen process finished")

	if _, ok := constants.SelectedParties[envPartyID]; ok {
		go p.Sign(context.Background(), []byte(signMessage))
	}

	// hang the app
	end := make(chan string)
	<-end
}
