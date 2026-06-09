package mcp

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type Service struct {
	dataDir string

	mu       sync.Mutex
	sessions map[string]*stdioSession
	emit     func(name string, payload any)
}

type stdioSession struct {
	id      string
	framing stdioFraming
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	done    chan struct{}
	waitCh  chan error

	writeMu   sync.Mutex
	closeOnce sync.Once
	closeErr  error
}

type stdioFraming string

const (
	framingJSONL         stdioFraming = "jsonl"
	framingContentLength stdioFraming = "content_length"
)

func NewService(dataDir string) (*Service, error) {
	return &Service{
		dataDir:  dataDir,
		sessions: make(map[string]*stdioSession),
	}, nil
}

func (s *Service) SetEventEmitter(emit func(name string, payload any)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.emit = emit
}

func (s *Service) StartStdioSession(command string, args []string, env map[string]string, framing *string, cwd *string) (string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "", errors.New("stdio MCP transport requires command")
	}

	normalizedArgs := make([]string, 0, len(args))
	for _, arg := range args {
		if arg != "" {
			normalizedArgs = append(normalizedArgs, arg)
		}
	}

	cmd := exec.Command(command, normalizedArgs...)
	cmd.Env = os.Environ()
	for key, value := range env {
		if key != "" {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}
	if cwd != nil && strings.TrimSpace(*cwd) != "" {
		cmd.Dir = *cwd
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		message := fmt.Sprintf("failed to spawn process %q: %v", command, err)
		if errors.Is(err, exec.ErrNotFound) {
			message += " - ensure the executable exists and is reachable via PATH"
		}
		return "", errors.New(message)
	}

	sessionID, err := newSessionID()
	if err != nil {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return "", err
	}

	session := &stdioSession{
		id:      sessionID,
		framing: normalizeFraming(framing),
		cmd:     cmd,
		stdin:   stdin,
		done:    make(chan struct{}),
		waitCh:  make(chan error, 1),
	}

	s.mu.Lock()
	s.sessions[sessionID] = session
	s.mu.Unlock()

	go func() {
		session.waitCh <- cmd.Wait()
	}()
	go s.readStdout(session, stdout)
	go s.readStderr(stderr)

	return sessionID, nil
}

func (s *Service) SendStdioMessage(sessionID string, payload string) error {
	session := s.getSession(sessionID)
	if session == nil {
		return fmt.Errorf("unknown MCP stdio session: %s", sessionID)
	}

	session.writeMu.Lock()
	defer session.writeMu.Unlock()

	select {
	case <-session.done:
		return fmt.Errorf("MCP stdio session is closed: %s", sessionID)
	default:
	}

	var frame string
	if session.framing == framingJSONL {
		frame = payload + "\n"
	} else {
		frame = formatContentLengthFrame(payload)
	}

	if _, err := io.WriteString(session.stdin, frame); err != nil {
		return fmt.Errorf("failed to write MCP stdio payload: %w", err)
	}
	return nil
}

func (s *Service) CloseStdioSession(sessionID string) error {
	session := s.removeSession(sessionID)
	if session == nil {
		return nil
	}
	return session.close()
}

func (s *Service) CloseAll() {
	s.mu.Lock()
	ids := make([]string, 0, len(s.sessions))
	for id := range s.sessions {
		ids = append(ids, id)
	}
	s.mu.Unlock()

	for _, id := range ids {
		_ = s.CloseStdioSession(id)
	}
}

func (s *Service) getSession(sessionID string) *stdioSession {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessions[sessionID]
}

func (s *Service) removeSession(sessionID string) *stdioSession {
	s.mu.Lock()
	defer s.mu.Unlock()
	session := s.sessions[sessionID]
	delete(s.sessions, sessionID)
	return session
}

func (s *Service) removeSessionIfCurrent(session *stdioSession) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sessions[session.id] != session {
		return false
	}
	delete(s.sessions, session.id)
	return true
}

func (s *Service) currentEmitter() func(name string, payload any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.emit
}

func (s *Service) emitEvent(name string, payload any) {
	if emit := s.currentEmitter(); emit != nil {
		emit(name, payload)
	}
}

func (s *Service) readStdout(session *stdioSession, stdout io.Reader) {
	var readErr error
	reader := bufio.NewReader(stdout)
	if session.framing == framingJSONL {
		readErr = s.readJSONL(session, reader)
	} else {
		readErr = s.readContentLengthLoop(session, reader)
	}

	if readErr != nil && !errors.Is(readErr, io.EOF) {
		s.emitEvent("mcp-stdio-"+session.id+"-error", map[string]string{"error": readErr.Error()})
	}

	if s.removeSessionIfCurrent(session) {
		_ = session.close()
		s.emitEvent("mcp-stdio-"+session.id+"-closed", map[string]any{})
	}
}

func (s *Service) readJSONL(session *stdioSession, reader *bufio.Reader) error {
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			trimmed := strings.TrimRight(line, "\r\n")
			if trimmed != "" {
				if isContentLengthHeader(trimmed) {
					message, err := readContentLengthBodyAfterHeader(reader, trimmed)
					if err != nil {
						return err
					}
					s.emitEvent("mcp-stdio-"+session.id+"-message", map[string]string{"message": message})
					return s.readContentLengthLoop(session, reader)
				}
				s.emitEvent("mcp-stdio-"+session.id+"-message", map[string]string{"message": trimmed})
			}
		}
		if err != nil {
			return err
		}
	}
}

func (s *Service) readContentLengthLoop(session *stdioSession, reader *bufio.Reader) error {
	for {
		message, err := readContentLengthMessage(reader)
		if err != nil {
			return err
		}
		s.emitEvent("mcp-stdio-"+session.id+"-message", map[string]string{"message": message})
	}
}

func (s *Service) readStderr(stderr io.Reader) {
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		// Stdio MCP servers often use stderr for regular logs. Match the old
		// Rust proxy by draining it without surfacing every log line as a
		// transport error.
	}
}

func (s *stdioSession) close() error {
	s.closeOnce.Do(func() {
		close(s.done)
		_ = s.stdin.Close()
		if s.cmd.Process == nil {
			return
		}

		exited, err := s.waitWithTimeout(3 * time.Second)
		if err != nil {
			s.closeErr = err
		}
		if exited {
			return
		}
		if killErr := s.cmd.Process.Kill(); killErr != nil {
			s.closeErr = killErr
			return
		}
		_, err = s.waitWithTimeout(3 * time.Second)
		if err != nil {
			s.closeErr = err
		}
	})
	return s.closeErr
}

func normalizeFraming(value *string) stdioFraming {
	if value == nil {
		return framingContentLength
	}
	switch strings.ToLower(strings.TrimSpace(*value)) {
	case "jsonl", "json_lines", "json-lines":
		return framingJSONL
	default:
		return framingContentLength
	}
}

func newSessionID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("failed to generate stdio session id: %w", err)
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(raw[:])
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:], nil
}

func (s *stdioSession) waitWithTimeout(timeout time.Duration) (bool, error) {
	select {
	case err := <-s.waitCh:
		return true, err
	case <-time.After(timeout):
		return false, nil
	}
}

func formatContentLengthFrame(message string) string {
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len([]byte(message)), message)
}

func readContentLengthMessage(reader *bufio.Reader) (string, error) {
	var contentLength int
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		line = strings.TrimRight(line, "\r\n")
		if strings.TrimSpace(line) == "" {
			break
		}
		if isContentLengthHeader(line) {
			parsed, err := parseContentLength(line)
			if err != nil {
				return "", err
			}
			contentLength = parsed
		}
	}
	if contentLength <= 0 {
		return "", errors.New("missing or invalid Content-Length header")
	}
	return readExactString(reader, contentLength)
}

func readContentLengthBodyAfterHeader(reader *bufio.Reader, firstHeader string) (string, error) {
	contentLength, err := parseContentLength(firstHeader)
	if err != nil {
		return "", err
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(line) == "" {
			break
		}
	}
	return readExactString(reader, contentLength)
}

func isContentLengthHeader(line string) bool {
	key, _, ok := strings.Cut(line, ":")
	return ok && strings.EqualFold(strings.TrimSpace(key), "Content-Length")
}

func parseContentLength(line string) (int, error) {
	_, value, ok := strings.Cut(line, ":")
	if !ok {
		return 0, fmt.Errorf("invalid Content-Length header: %q", line)
	}
	var length int
	if _, err := fmt.Sscanf(strings.TrimSpace(value), "%d", &length); err != nil {
		return 0, fmt.Errorf("invalid Content-Length value: %w", err)
	}
	const maxMessageSize = 100 * 1024 * 1024
	if length <= 0 || length > maxMessageSize {
		return 0, fmt.Errorf("invalid Content-Length value: %d", length)
	}
	return length, nil
}

func readExactString(reader *bufio.Reader, length int) (string, error) {
	buffer := make([]byte, length)
	if _, err := io.ReadFull(reader, buffer); err != nil {
		return "", err
	}
	return string(buffer), nil
}
