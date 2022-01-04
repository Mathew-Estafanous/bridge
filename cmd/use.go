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
		currSync: make([]syncingFile, 0),
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

type syncingFile struct {
	name string
	track fs.Tracker
}

type syncModel struct {
	er EventReceiver
	session string
	currSync []syncingFile
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
			m.currSync = append(m.currSync, syncingFile{msg.Name, msg.Track})
		case fs.Done, fs.Failed:
			for i, v := range m.currSync {
				if v.name == msg.Name {
					m.currSync = append(m.currSync[:i], m.currSync[i+1:]...)
					break
				}
			}
		}
		return m, listenForEvents(m.er)
	default:
		return m, func() tea.Msg {
			time.Sleep(200 * time.Millisecond)
			return "refresh"
		}
	}
}

func (m syncModel) View() string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("36")).PaddingLeft(2)
	s := fmt.Sprintf("Bridge: \n%v\n\n", style.Render(m.session))

	headStyle := lipgloss.NewStyle().Background(lipgloss.Color("36")).
		Foreground(lipgloss.Color("231"))
	s += headStyle.Render(" Files: ") + "\n"
	fileStyle := lipgloss.NewStyle().Bold(true).PaddingLeft(1)
	for _, f := range m.currSync {
		s += fileStyle.Render(fmt.Sprintf("%v -- %v", f.name, f.track.SyncedSize())) + "\n"
	}
	return s
}

func listenForEvents(er EventReceiver) func() tea.Msg {
	return func() tea.Msg {
		return <- er.ReceiveEvents()
	}
}

