package cmd

import (
	"fmt"
	"github.com/Mathew-Estafanous/bridge/fs"
	"github.com/Mathew-Estafanous/bridge/p2p"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"log"
	"time"
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

func runOpen(cmd *cobra.Command, _ []string) {
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
	sendMdl := sendModel{
		joinLis:  bridge,
		strmOpen: bridge,
		session:  bridge.Session(),
	}
	p := tea.NewProgram(sendMdl)
	if err := p.Start(); err != nil {
		log.Println(err)
	}
}

type JoinListener interface {
	JoinedPeerListener() <-chan p2p.Peer
}

type sendModel struct {
	joinLis  JoinListener
	strmOpen fs.StreamOpener
	session  string
	closeCh  chan struct{}

	totalClients uint
}

func (s sendModel) Init() tea.Cmd {
	return handleJoinedPeer(s.joinLis)
}

func (s sendModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.KeyMsg:
		close(s.closeCh)
		return s, tea.Quit
	case p2p.Peer:
		ws, err := fs.NewFileSender(m, s.strmOpen)
		if err != nil {
			return s, handleJoinedPeer(s.joinLis)
		}
		s.totalClients++
		ws.Start()

		go func() {
			for {
				select {
				case e := <-ws.ReceiveEvents():
					if e.Err != nil {
						log.Println(e.Err)
						return
					}
				case <-s.closeCh:
					return
				}
			}
		}()
		return s, handleJoinedPeer(s.joinLis)
	default:
		return s, func() tea.Msg {
			time.Sleep(200 * time.Millisecond)
			return "refresh"
		}
	}
}

func (s sendModel) View() string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("36")).PaddingLeft(2)
	out := fmt.Sprintf("Bridge: \n%v\n\n", style.Render(s.session))

	headStyle := lipgloss.NewStyle().Background(lipgloss.Color("36")).
		Foreground(lipgloss.Color("231"))
	out += headStyle.Render(" Sending: ") + "\n"
	return out
}

func handleJoinedPeer(joinLis JoinListener) func() tea.Msg {
	return func() tea.Msg {
		select {
		case p := <-joinLis.JoinedPeerListener():
			return p
		}
	}
}
