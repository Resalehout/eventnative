package authorization

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/ksensehq/eventnative/resources"
	"github.com/spf13/viper"
	"log"
	"strings"
	"sync"
)

type Token struct {
	Value   string   `json:"token,omitempty"`
	Origins []string `json:"origins,omitempty"`
}

type Service struct {
	sync.RWMutex
	c2STokens map[string][]string
	s2STokens map[string][]string
}

func NewService() (*Service, error) {
	service := &Service{}

	reloadSec := viper.GetInt("server.server.auth_reload_sec")
	if reloadSec == 0 {
		return nil, errors.New("server.server.auth_reload_sec can't be empty")
	}

	c2sTokens, err := load("server.auth", service.updateC2STokens, reloadSec)
	if err != nil {
		return nil, fmt.Errorf("Error loading server.auth tokens: %v", err)
	}

	s2sTokens, err := load("server.s2s_auth", service.updateS2STokens, reloadSec)
	if err != nil {
		return nil, fmt.Errorf("Error loading server.s2s_auth tokens: %v", err)
	}

	if len(c2sTokens) == 0 && len(s2sTokens) == 0 {
		//autogenerated
		generatedToken := uuid.New().String()
		c2sTokens[generatedToken] = []string{}
		s2sTokens[generatedToken] = []string{}
		log.Println("Empty 'server.tokens' config key. Auto generate token:", generatedToken)
	}

	service.c2STokens = c2sTokens
	service.s2STokens = s2sTokens

	return service, nil
}

func (s *Service) GetC2SOrigins(token string) ([]string, bool) {
	s.RLock()
	defer s.RUnlock()

	origins, ok := s.c2STokens[token]
	return origins, ok
}

func (s *Service) GetAllTokens() map[string]bool {
	s.RLock()
	defer s.RUnlock()

	result := map[string]bool{}
	for k := range s.c2STokens {
		result[k] = true
	}
	for k := range s.s2STokens {
		result[k] = true
	}
	return result
}

func (s *Service) GetS2SOrigins(token string) ([]string, bool) {
	s.RLock()
	defer s.RUnlock()

	origins, ok := s.s2STokens[token]
	return origins, ok
}

func (s *Service) updateC2STokens(c2sTokens map[string][]string) {
	s.Lock()
	s.c2STokens = c2sTokens
	s.Unlock()
}

func (s *Service) updateS2STokens(s2sTokens map[string][]string) {
	s.Lock()
	s.s2STokens = s2sTokens
	s.Unlock()
}

func load(viperKey string, updateFunc func(map[string][]string), reloadSec int) (map[string][]string, error) {
	authSource := viper.GetString(viperKey)
	if strings.Contains(authSource, "http://") || strings.Contains(authSource, "https://") {
		return Watcher(authSource, resources.LoadFromHttp, updateFunc, reloadSec)
	} else if strings.Contains(authSource, "file://") {
		return Watcher(strings.Replace(authSource, "file://", "", 1), resources.LoadFromFile, updateFunc, reloadSec)
	} else if strings.HasPrefix(authSource, "[") && strings.HasSuffix(authSource, "]") {
		return parseFromBytes("app config json array", []byte(authSource))
	} else {
		return loadFromConfig(viperKey)
	}
}

func loadFromConfig(viperKey string) (map[string][]string, error) {
	tokensArr := viper.GetStringSlice(viperKey)
	tokensOrigins := map[string][]string{}
	for _, t := range tokensArr {
		trimmed := strings.TrimSpace(t)
		if trimmed != "" {
			tokensOrigins[trimmed] = []string{}
		}
	}

	return tokensOrigins, nil
}

func parseFromBytes(source string, b []byte) (map[string][]string, error) {
	var tokens []Token
	err := json.Unmarshal(b, &tokens)
	if err != nil {
		//try to unmarshal into []string
		var strTokens []string
		err := json.Unmarshal(b, &strTokens)
		if err != nil {
			return nil, fmt.Errorf("Error unmarshalling tokens from %s. Payload must be json array or string array: %v", source, err)
		}
		for _, t := range strTokens {
			tokens = append(tokens, Token{Value: t})
		}
	}

	tokensOrigins := map[string][]string{}
	for _, tokenObj := range tokens {
		trimmed := strings.TrimSpace(tokenObj.Value)
		if trimmed != "" {
			tokensOrigins[trimmed] = tokenObj.Origins
		}
	}

	return tokensOrigins, nil
}
