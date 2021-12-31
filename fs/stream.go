package fs

import (
	"encoding/binary"
	"fmt"
	"github.com/Mathew-Estafanous/bridge/p2p"
	"io"
	"log"
	"os"
	"strings"
)

// StreamOpener is an interface that enables the ability to open a read/write
// connection with another peer using the peer's ID.
type StreamOpener interface {
	OpenStream(peerId p2p.Peer) (io.WriteCloser, error)
}

// StreamListener is an interfaces that allows for listening for any new stream
// connections with this peer.
type StreamListener interface {
	ListenForStream() <-chan io.ReadCloser
}

type EventType uint8

const (
	// Start event represents the beginning of a file job.
	Start EventType = iota
	// Done event represents the successful completion of the file job.
	Done
	// Failed event occurs when a file job failed and can no longer continue.
	Failed
)

type FileEvent struct {
	typ EventType
	name string // name of the file for the event.
	err error // an associated error if the event is an error.
}

// FileSender , when running, asynchronously streams all file data within the
// running directory to the target peer.
type FileSender struct {
	opener StreamOpener
	p      p2p.Peer
	fd     []FileData
	resultCh chan FileEvent
}

func NewFileSender(peer p2p.Peer, opener StreamOpener) (*FileSender, error) {
	fileData, err := allFilesWithinDirectory(".")
	if err != nil {
		return nil, err
	}
	return &FileSender{
		opener: opener,
		p:      peer,
		fd:     fileData,
		resultCh: make(chan FileEvent, len(fileData)),
	}, nil
}

// Start will start the streaming process in a separate goroutine.
func (s *FileSender) Start() {
	go func() {
		fileJobs := make(chan FileData, len(s.fd))
		for i := 0; i < 5; i++ {
			go s.transferFile(fileJobs)
		}

		for _, f := range s.fd {
			fileJobs <- f
		}
		close(fileJobs)
	}()
}

func (s *FileSender) transferFile(jobs <-chan FileData) {
	for fd := range jobs {
		newFileEvent := func(t EventType, err error) FileEvent {
			return FileEvent{
				typ: t,
				name: fd.Name(),
				err: err,
			}
		}

		strm, err := s.opener.OpenStream(s.p)
		if err != nil {
			s.resultCh <- newFileEvent(Failed, err)
			strm.Close()
			continue
		}
		ln := uint32(len([]byte(fd.String())))
		pathLn := make([]byte, 5)
		binary.LittleEndian.PutUint32(pathLn, ln)
		pathData := append(pathLn, []byte(fd.String())...)
		if _, err := strm.Write(pathData); err != nil {
			s.resultCh <- newFileEvent(Failed, err)
			strm.Close()
			continue
		}

		f, err := os.Open(fd.String())
		if err != nil {
			s.resultCh <- newFileEvent(Failed, err)
			strm.Close()
			f.Close()
			continue
		}

		if _, err := io.Copy(strm, f); err != nil {
			s.resultCh <- newFileEvent(Failed, err)
			strm.Close()
			f.Close()
			continue
		}
		s.resultCh <- newFileEvent(Done, nil)
		strm.Close()
		f.Close()
	}
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

// FileReceiver is used to accept file data through a stream and write the file data within
// the current execution directory.
type FileReceiver struct {
	lis StreamListener
}

func NewFileReceiver(lis StreamListener) *FileReceiver {
	r := &FileReceiver{lis}
	go r.startListening()
	return r
}

func (r *FileReceiver) startListening() {
	for {
		select {
		case strm := <-r.lis.ListenForStream():
			go func() {
				if err := writeFile(strm); err != nil {
					log.Println(err)
				}
			}()
		}
	}
}

func writeFile(strm io.ReadCloser) error {
	defer strm.Close()
	b := make([]byte, 5)
	if _, err := strm.Read(b); err != nil {
		return err
	}
	pathLn := binary.LittleEndian.Uint32(b)
	pathB := make([]byte, pathLn)
	if _, err := strm.Read(pathB); err != nil {
		return err
	}

	path := string(pathB)
	i := strings.LastIndex(path, "/")
	dir := path[:i]

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0777); err != nil {
			return err
		}
	}

	f, err := os.Create(path)
	defer f.Close()
	if err != nil {
		return err
	}

	if _, err := io.Copy(f, strm); err != nil {
		return err
	}
	return nil
}
