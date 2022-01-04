package cmd

import (
	"fmt"
	"github.com/Mathew-Estafanous/bridge/fs"
	"github.com/Mathew-Estafanous/bridge/p2p"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"log"
)

var useCmd = &cobra.Command{
	Use:   "use SessionID",
	Short: "Connect to a bridge and start fs sync.",
	Long: `Uses the provided session id token to connect with a bridge instance and when a
connection has be successful fs sync will automatically start within the current directory.

SessionID - Provided ID of the bridge instance that you intent to connect with.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("expects exactly 1 session id but got %v", len(args))
		}
		return nil
	},
	Run: runUse,
}

func init() {
	rootCmd.AddCommand(useCmd)
}

func runUse(cmd *cobra.Command, args []string) {
	client, err := p2p.NewClient(args[0])
	if err != nil {
		log.Println(err)
		return
	}
	fr := fs.NewFileReceiver(client)
	sm := syncModel{
		er:       fr,
		session: args[0],
		currSync: make(map[string]fs.Tracker),
		closeCh: make(chan struct{}),
	}
	p := tea.NewProgram(sm)
	if err := p.Start(); err != nil {
		log.Println(err)
		return
	}
}

type EventReceiver interface {
	ReceiveEvents() <- chan fs.FileEvent
}

type syncModel struct {
	er EventReceiver
	session string
	currSync map[string]fs.Tracker
	closeCh chan struct{}
}

func (m syncModel) Init() tea.Cmd {
	return listenForEvents(m.er)
}

func (m syncModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		close(m.closeCh)
		return m, tea.Quit
	case fs.FileEvent:
		switch msg.Typ {
		case fs.Start:
			m.currSync[msg.Name] = msg.Track
		case fs.Done:
			delete(m.currSync, msg.Name)
		}
		return m, listenForEvents(m.er)
	default:
		return m, nil

	}
}

func (m syncModel) View() string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("36")).PaddingLeft(2)
	s := fmt.Sprintf("Bridge: \n%v\n\n", style.Render(m.session))

	headStyle := lipgloss.NewStyle().Background(lipgloss.Color("36")).
		Foreground(lipgloss.Color("231"))
	s += headStyle.Render(" Files: ") + "\n"
	fileStyle := lipgloss.NewStyle().Bold(true).PaddingLeft(1)
	for k, v := range m.currSync {
		s += fileStyle.Render(fmt.Sprintf("%v -- %v", k, v.SyncedSize())) + "\n"
	}
	return s
}

func listenForEvents(er EventReceiver) func() tea.Msg {
	return func() tea.Msg {
		return <- er.ReceiveEvents()
	}
}

