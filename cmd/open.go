package cmd

import (
	"fmt"
	"github.com/Mathew-Estafanous/bridge/p2p"
	"github.com/spf13/cobra"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var openCmd = &cobra.Command{
	Use: "open",
	Short: "Start a bridge instance within the current folder.",
	Long: `Creates a bridge instance that is open for clients to connect to and sync with.
All folders and files within the current directory will be synced with the connecting clients.
	
When a bridge has successfully opened, you will be prompted with the Session ID that can
be shared with those you wish to connect with.`,
	Run: runOpen,
}

func init() {
	rootCmd.AddCommand(openCmd)
	rootCmd.PersistentFlags().BoolP("test", "t", false, "determine the mode of the bridge")
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
	go waitForJoinedPeer(bridge)
	run(bridge)
}

func waitForJoinedPeer(bridge p2p.Bridge) {
	for {
		select {
		case peer := <- bridge.JoinedPeerListener():
			if err := bridge.Send(peer.Id , []byte("hi friend!")); err != nil {
				log.Println(err)
			}
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

type FileData struct {
	os.FileInfo
	parent string // the parent directory.
}

func (f FileData) String() string {
	return f.parent + "/" + f.Name()
}

// will iterate through all the files within the directory and when a directory is present within
// the current directory, then it will recursively call the child directory.
func allFilesWithinDirectory(dir string) ([]FileData, error) {
	dirInfo, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to get directory info: %v", err)
	}

	info := make([]FileData, 0, len(dirInfo))
	for _, d := range dirInfo {
		if d.IsDir() {
			childInfo, err := allFilesWithinDirectory(dir + "/" + d.Name())
			if err != nil {
				fmt.Printf("Failed to get info of child directory %s", dir+"/"+d.Name())
				continue
			}
			info = append(info, childInfo...)
		} else {
			fileInfo, err := d.Info()
			if err != nil {
				fmt.Printf("Failed to get file info of file %s", d.Name())
				continue
			}
			info = append(info, FileData{
				FileInfo: fileInfo,
				parent: dir,
			})
		}
	}
	return info, nil
}
