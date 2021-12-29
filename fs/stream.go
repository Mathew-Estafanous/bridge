package fs

import (
	"encoding/binary"
	"fmt"
	"github.com/Mathew-Estafanous/bridge/p2p"
	"io"
	"os"
)

// StreamOpener is an interface that enables the ability to open a read/write
// connection with another peer using the peer's ID.
type StreamOpener interface {
	OpenStream(peerId p2p.Peer) (io.ReadWriter, error)
}

// SendStream , when running, asynchronously streams all file data within the
// running directory to the target peer.
type SendStream struct {
	opener StreamOpener
	p      p2p.Peer
	fd     []FileData
}

func NewWriteStream(peer p2p.Peer, opener StreamOpener) (*SendStream, error) {
	fileData, err := allFilesWithinDirectory(".")
	if err != nil {
		return nil, err
	}
	return &SendStream{
		opener: opener,
		p:      peer,
		fd:     fileData,
	}, nil
}

// Start will start the streaming process in a separate goroutine.
func (s *SendStream) Start() {
	go func() {
		fileJobs := make(chan FileData, len(s.fd))
		results := make(chan Result, len(s.fd))
		for i := 0; i < 5; i++ {
			go s.transferFile(fileJobs, results)
		}

		for _, f := range s.fd {
			fileJobs <- f
		}
		close(fileJobs)

		for a := 1; a <= len(s.fd); a++ {
			<-results
		}
		close(results)
	}()
}

func (s *SendStream) transferFile(jobs <-chan FileData, result chan Result) {
	for fd := range jobs {
		strm, err := s.opener.OpenStream(s.p)
		if err != nil {
			result <- Result{fd.Name(), err}
			continue
		}
		ln := uint32(len([]byte(fd.String())))
		pathLn := make([]byte, 5)
		binary.LittleEndian.PutUint32(pathLn, ln)
		pathData := append(pathLn, []byte(fd.String())...)
		if _, err := strm.Write(pathData); err != nil {
			result <- Result{fd.Name(), err}
			continue
		}

		f, err := os.Open(fd.String())
		if err != nil {
			result <- Result{fd.Name(), err}
			continue
		}

		if _, err := io.Copy(strm, f); err != nil {
			result <- Result{fd.Name(), err}
			continue
		}
		result <- Result{fd.Name(), nil}
	}
}

type Result struct {
	name string // name of the file.
	err  error  // nil if work was successful or non-nil if failed.
}

type FileData struct {
	os.FileInfo
	path string // the path directory.
}

func (f FileData) String() string {
	return f.path + "/" + f.Name()
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
				fmt.Printf("Failed to get fs info of fs %s", d.Name())
				continue
			}
			info = append(info, FileData{
				FileInfo: fileInfo,
				path:     dir,
			})
		}
	}
	return info, nil
}