package cmd

import (
	"github.com/Mathew-Estafanous/bridge/fs"
	"github.com/Mathew-Estafanous/bridge/p2p"
	"github.com/spf13/cobra"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var openCmd = &cobra.Command{
	Use:   "open",
	Short: "Start a bridge instance within the current folder.",
	Long: `Creates a bridge instance that is open for clients to connect to and sync with.
All folders and files within the current directory will be synced with the connecting clients.
	
When a bridge has successfully opened, you will be prompted with the Session ID that can
be shared with those you wish to connect with.`,
	Run: runOpen,
}

func init() {
	rootCmd.AddCommand(openCmd)
	openCmd.PersistentFlags().BoolP("test", "t", false, "determine the mode of the bridge")
}

func runOpen(cmd *cobra.Command, args []string) {
	isTest, err := cmd.Flags().GetBool("test")
	if err != nil {
		log.Println(err)
		return
	}

	bridge, err := p2p.NewBridge(isTest)
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf("Session ID: %s", bridge.Session())
	go handledJoined(bridge)
	run(bridge)
}

// TODO: Change how joined peers are handled.
func handledJoined(bridge *p2p.Bridge) {
	for {
		select {
		case p := <-bridge.JoinedPeerListener():
			ws, err := fs.NewFileSender(p, bridge)
			if err != nil {
				log.Printf("Failed to create write stream: %v", err)
				continue
			}
			ws.Start()
			// TODO: Handle receiving file events.
			go func() {
				for {
					e := <- ws.ReceiveEvents()
					if e.Err != nil {
						log.Println(e.Err)
					}
				}
			}()
		}
	}
}

func run(closer io.Closer) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	<-c

	log.Printf("\rExiting...\n")
	if err := closer.Close(); err != nil {
		log.Printf("Encountered issue while closing bridge: %v", err)
		os.Exit(1)
	}
	os.Exit(0)
}
