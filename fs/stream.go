package fs

import (
	"encoding/binary"
	"fmt"
	"github.com/Mathew-Estafanous/bridge/p2p"
	"io"
	"os"
	"strings"
	"sync/atomic"
)

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
	Typ   EventType
	Name  string  // Name of the file for the event.
	Track Tracker // Track is non-nil when the event type is Start.
	Err   error   // Err is non-nil if the event type is Failed.
}

// StreamOpener is an interface that enables the ability to open a read/write
// connection with another peer using the peer's ID.
type StreamOpener interface {
	OpenStream(peerId p2p.Peer) (p2p.WriteResetter, error)
}

// FileSender , when running, asynchronously streams all file data within the
// running directory to the target peer.
type FileSender struct {
	opener  StreamOpener
	p       p2p.Peer
	fd      []fileData
	eventCh chan FileEvent
}

func NewFileSender(peer p2p.Peer, opener StreamOpener) (*FileSender, error) {
	fileData, err := allFilesWithinDirectory(".")
	if err != nil {
		return nil, err
	}
	return &FileSender{
		opener:  opener,
		p:       peer,
		fd:      fileData,
		eventCh: make(chan FileEvent, len(fileData)),
	}, nil
}

func (s *FileSender) ReceiveEvents() <-chan FileEvent {
	return s.eventCh
}

// Start will start the streaming process in a separate goroutine.
func (s *FileSender) Start() {
	go func() {
		fileJobs := make(chan fileData, len(s.fd))
		for i := 0; i < 5; i++ {
			go s.sendFileWorker(fileJobs)
		}

		for _, f := range s.fd {
			fileJobs <- f
		}
		close(fileJobs)
	}()
}

func (s *FileSender) sendFileWorker(jobs <-chan fileData) {
	for fd := range jobs {
		newFileEvent := func(et EventType, err error) FileEvent {
			return FileEvent{
				Typ:  et,
				Name: fd.Name(),
				Err:  err,
			}
		}
		strm, err := s.opener.OpenStream(s.p)
		if err != nil {
			err = fmt.Errorf("couldn't open the stream: %w", err)
			s.eventCh <- newFileEvent(Failed, err)
			strm.Reset()
			continue
		}
		ln := uint32(len([]byte(fd.String())))
		pathLn := make([]byte, 5)
		binary.LittleEndian.PutUint32(pathLn, ln)
		pathData := append(pathLn, []byte(fd.String())...)
		if _, err := strm.Write(pathData); err != nil {
			err = fmt.Errorf("could not write path data: %w", err)
			s.eventCh <- newFileEvent(Failed, err)
			strm.Reset()
			continue
		}

		f, err := os.Open(fd.String())
		if err != nil {
			err = fmt.Errorf("unable to open path %s: %w", fd.String(), err)
			s.eventCh <- newFileEvent(Failed, err)
			strm.Reset()
			f.Close()
			continue
		}

		fileEvent := newFileEvent(Start, nil)
		syncTrk := &syncTracker{rw: f}
		fileEvent.Track = syncTrk
		s.eventCh <- fileEvent
		if _, err := io.Copy(strm, syncTrk); err != nil {
			err = fmt.Errorf("failed to complete streaming: %w", err)
			s.eventCh <- newFileEvent(Failed, err)
			strm.Reset()
			f.Close()
			continue
		}
		s.eventCh <- newFileEvent(Done, nil)
		strm.Reset()
		f.Close()
	}
}

type fileData struct {
	os.FileInfo
	path string // the path directory.
}

func (f fileData) String() string {
	return f.path + "/" + f.Name()
}

// will iterate through all the files within the directory and when a directory is present within
// the current directory, then it will recursively call the child directory.
func allFilesWithinDirectory(dir string) ([]fileData, error) {
	dirInfo, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to get directory info: %v", err)
	}

	info := make([]fileData, 0, len(dirInfo))
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
			info = append(info, fileData{
				FileInfo: fileInfo,
				path:     dir,
			})
		}
	}
	return info, nil
}

// StreamListener is an interfaces that allows for listening for any new stream
// connections with this peer.
type StreamListener interface {
	ListenForStream() <-chan io.ReadCloser
}

// FileReceiver is used to accept file data through a stream and write the file data within
// the current execution directory.
type FileReceiver struct {
	lis     StreamListener
	eventCh chan FileEvent
}

func NewFileReceiver(lis StreamListener) *FileReceiver {
	r := &FileReceiver{
		lis:     lis,
		eventCh: make(chan FileEvent, 10),
	}
	go r.startListening()
	return r
}

func (r *FileReceiver) ReceiveEvents() <-chan FileEvent {
	return r.eventCh
}

func (r *FileReceiver) startListening() {
	for {
		select {
		case strm := <-r.lis.ListenForStream():
			go writeFile(strm, r.eventCh)
		}
	}
}

func writeFile(strm io.ReadCloser, eventCh chan FileEvent) {
	b := make([]byte, 5)
	if _, err := strm.Read(b); err != nil {
		err = fmt.Errorf("couldn't read first 5 bytes: %w", err)
		eventCh <- FileEvent{Failed, "", nil, err}
		return
	}
	pathLn := binary.LittleEndian.Uint32(b)
	pathB := make([]byte, pathLn)
	if _, err := strm.Read(pathB); err != nil {
		err = fmt.Errorf("couldn't read file path and name: %w", err)
		eventCh <- FileEvent{Failed, "", nil, err}
		return
	}

	path := string(pathB)
	i := strings.LastIndex(path, "/")
	dir := path[:i]

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0777); err != nil {
			eventCh <- FileEvent{Failed, path[i:], nil, err}
			return
		}
	}

	f, err := os.Create(path)
	wt := &syncTracker{rw: f}
	eventCh <- FileEvent{Start, path[i:], wt, nil}
	defer f.Close()
	if err != nil {
		err = fmt.Errorf("unable to create file %s: %w", path, err)
		eventCh <- FileEvent{Failed, path[i:], nil, err}
		return
	}

	if _, err := io.Copy(wt, strm); err != nil {
		err = fmt.Errorf("failed to complete streaming %s: %w", path[i:], err)
		eventCh <- FileEvent{Failed, path[i:], nil, err}
		return
	}
	eventCh <- FileEvent{Done, path[i:], nil, nil}
}

// Tracker is used to get data regarding how much data has successfully
// synced (either sent or received).
type Tracker interface {
	SyncedSize() uint64
}

type syncTracker struct {
	rw   io.ReadWriter
	size uint64 // size is the total number of bits that have been written.
}

func (s *syncTracker) SyncedSize() uint64 {
	return atomic.LoadUint64(&s.size)
}

func (s *syncTracker) Write(p []byte) (n int, err error) {
	n, err = s.rw.Write(p)
	atomic.AddUint64(&s.size, uint64(n))
	return n, err
}

func (s *syncTracker) Read(p []byte) (n int, err error) {
	n, err = s.rw.Read(p)
	atomic.AddUint64(&s.size, uint64(n))
	return n, err
}
