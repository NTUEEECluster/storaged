package storaged

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/user"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type Server struct {
	ServerConfig

	limiterMutex sync.Mutex
	limiter      map[int]clientLimit
	updateMutex  sync.Mutex
}

func NewServer(cfg ServerConfig) *Server {
	return &Server{
		ServerConfig: cfg,
		limiterMutex: sync.Mutex{},
		limiter:      make(map[int]clientLimit),
		updateMutex:  sync.Mutex{},
	}
}

type ServerConfig struct {
	AllowedEncodeHost *net.IPNet

	ProjectFS QuotaFS
	Tiers     map[string]QuotaFS
	// Allocations is the map from the group name to the Allocation the user is entitled to.
	Allocations map[string][]Allocation
}

type Allocation struct {
	Tier     string `toml:"tier"`
	MaxBytes int    `toml:"max_bytes"`
}

type clientLimit struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func (s *Server) Listen(address string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /quota", s.handleCheckQuota)
	mux.HandleFunc("POST /folders", s.handleUpdateFolder)
	if err := http.ListenAndServe(address, mux); err != nil {
		return fmt.Errorf("error listening: %w", err)
	}
	return nil
}

func (s *Server) readRequest(writer http.ResponseWriter, req *http.Request, dest any) (submitter *user.User, ok bool) {
	// We have bigger issues if the request is larger than 1MB.
	reqBody, err := io.ReadAll(io.LimitReader(req.Body, 1024*1024))
	if err != nil {
		http.Error(writer, "Failed to read your request, try again", http.StatusBadRequest)
		return nil, false
	}
	mungeOutput, err := Unmunge(string(reqBody))
	if err != nil {
		http.Error(writer, "Failed to authenticate request: "+err.Error(), http.StatusUnauthorized)
		return nil, false
	}
	if mungeOutput.GroupID == nil || mungeOutput.UserID == nil || mungeOutput.EncodeHost == nil {
		http.Error(
			writer,
			"Failed to authenticate request: Missing field in Munge",
			http.StatusUnauthorized,
		)
		return nil, false
	}
	if !s.AllowedEncodeHost.Contains(mungeOutput.EncodeHost) {
		http.Error(
			writer,
			"Failed to authenticate request: Invalid encode host "+mungeOutput.EncodeHost.String(),
			http.StatusUnauthorized,
		)
		return nil, false
	}
	uid := *mungeOutput.UserID
	s.limiterMutex.Lock()
	if _, ok := s.limiter[uid]; !ok {
		s.limiter[uid] = clientLimit{
			limiter:  rate.NewLimiter(2, 5),
			lastSeen: time.Now(),
		}
	}
	ok = s.limiter[uid].limiter.Allow()
	s.limiterMutex.Unlock()
	if !ok {
		http.Error(
			writer,
			"You have sent too many requests recently. Please slow down.",
			http.StatusTooManyRequests,
		)
		return nil, false
	}
	submitter, err = user.LookupId(strconv.Itoa(uid))
	if err != nil {
		http.Error(
			writer,
			"Unable to find user details: "+err.Error(),
			http.StatusBadRequest,
		)
		return nil, false
	}
	decoder := json.NewDecoder(bytes.NewReader(mungeOutput.Payload))
	decoder.DisallowUnknownFields()
	err = decoder.Decode(dest)
	if err != nil {
		http.Error(
			writer,
			"Munge payload was not valid JSON: "+err.Error()+"\n"+"got: "+string(mungeOutput.Payload),
			http.StatusBadRequest,
		)
		return nil, false
	}
	return submitter, true
}
