package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
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
}

func runOpen(cmd *cobra.Command, args []string) {
	files, err := allFilesWithinDirectory(".")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(len(files))
}

type FileData struct {
	os.FileInfo
	parent string // the parent directory.
}

func (f FileData) String() string {
	return f.parent + "/" + f.Name()
}

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
