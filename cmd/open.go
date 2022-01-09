package cmd

import (
	"github.com/Mathew-Estafanous/bridge/fs"
	"github.com/Mathew-Estafanous/bridge/p2p"
	"github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"log"
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
	sendMdl := sendModel{
		joinLis: bridge,
		strmOpen: bridge,
		session: bridge.Session(),
	}
	p := tea.NewProgram(sendMdl)
	if err := p.Start(); err != nil {
		log.Println(err)
	}
}

type JoinListener interface {
	JoinedPeerListener() <- chan p2p.Peer
}

type sendModel struct {
	joinLis JoinListener
	strmOpen fs.StreamOpener
	session string
}

func (s sendModel) Init() tea.Cmd {
	return handleJoinedPeer(s.joinLis)
}

func (s sendModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case p2p.Peer:
		ws, err := fs.NewFileSender(m, s.strmOpen)
		if err != nil {
			return s, handleJoinedPeer(s.joinLis)
		}
		ws.Start()
		go func() {
			for {
				e := <- ws.ReceiveEvents()
				if e.Err != nil {
					log.Println(e.Err)
				}
			}
		}()
	}
	return s, handleJoinedPeer(s.joinLis)
}

func (s sendModel) View() string {
	panic("implement me")
}

func handleJoinedPeer(joinLis JoinListener) func() tea.Msg {
	return func() tea.Msg {
		select {
		case p := <- joinLis.JoinedPeerListener():
			return p
		}
	}
}
