package cmd

import (
	"fmt"
	"github.com/Mathew-Estafanous/bridge/fs"
	"github.com/Mathew-Estafanous/bridge/p2p"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"log"
	"sync"
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
	sm := &syncModel{
		er:         fr,
		session:    args[0],
		currSync:   make([]syncingFile, 0),
		failedSync: make([]failedFile, 0),
		closeCh:    make(chan struct{}),
	}
	p := tea.NewProgram(sm)
	if err := p.Start(); err != nil {
		log.Println(err)
	}
}

type EventReceiver interface {
	ReceiveEvents() <-chan fs.FileEvent
}

type syncingFile struct {
	name  string
	track fs.Tracker
}

type failedFile struct {
	name string
	err  error
}

type syncModel struct {
	er         EventReceiver
	session    string
	syncMu     sync.Mutex
	currSync   []syncingFile
	failMu     sync.Mutex
	failedSync []failedFile
	closeCh    chan struct{}
}

func (m *syncModel) Init() tea.Cmd {
	return listenForEvents(m.er)
}

func (m *syncModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		close(m.closeCh)
		return m, tea.Quit
	case fs.FileEvent:
		switch msg.Typ {
		case fs.Start:
			m.syncMu.Lock()
			m.currSync = append(m.currSync, syncingFile{msg.Name, msg.Track})
			m.syncMu.Unlock()
		case fs.Done:
			m.syncMu.Lock()
			m.currSync = remove(m.currSync, msg.Name)
			m.syncMu.Unlock()
		case fs.Failed:
			if msg.Name != "" {
				m.syncMu.Lock()
				m.currSync = remove(m.currSync, msg.Name)
				m.syncMu.Unlock()
			}
			m.failMu.Lock()
			m.failedSync = append(m.failedSync, failedFile{name: msg.Name, err: msg.Err})
			m.failMu.Unlock()
		}
		return m, listenForEvents(m.er)
	default:
		return m, func() tea.Msg {
			time.Sleep(200 * time.Millisecond)
			return "refresh"
		}
	}
}

func (m *syncModel) View() string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("36")).PaddingLeft(2)
	s := fmt.Sprintf("Bridge: \n%v\n\n", style.Render(m.session))

	headStyle := lipgloss.NewStyle().Background(lipgloss.Color("36")).
		Foreground(lipgloss.Color("231"))
	fileStyle := lipgloss.NewStyle().Bold(true).PaddingLeft(1)
	if len(m.currSync) > 0 {
		s += headStyle.Render(" Syncing: ") + "\n"
		m.syncMu.Lock()
		for _, f := range m.currSync {
			s += fileStyle.Render(fmt.Sprintf("%v -- %v", f.name, f.track.SyncedSize())) + "\n"
		}
		m.syncMu.Unlock()
	} else {
		headStyle.Background(lipgloss.Color("1"))
		s += headStyle.Render(" Errors: ") + "\n"
		if len(m.failedSync) == 0 {
			s += fileStyle.Render("(None)")
		} else {
			m.failMu.Lock()
			for _, f := range m.failedSync {
				s += fileStyle.Render(fmt.Sprintf("[%s]: %v", f.name, f.err)) + "\n"
			}
			m.failMu.Unlock()
		}
	}

	return s
}

func listenForEvents(er EventReceiver) func() tea.Msg {
	return func() tea.Msg {
		return <-er.ReceiveEvents()
	}
}

func remove(slice []syncingFile, name string) []syncingFile {
	for i, v := range slice {
		if v.name == name {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}
