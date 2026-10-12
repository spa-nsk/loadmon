// Package ldaptest — встроенный тестовый LDAP-сервер (glauth/ldap): привязка учётной записи поиска и
// пользователей, поиск по фильтрам RFC 4515, атрибуты AD и FreeIPA (sAMAccountName/uid, memberOf,
// userPrincipalName). Нужен для тестов входа без внешнего каталога.
package ldaptest

import (
	"net"
	"strings"
	"sync"
	"testing"

	gl "github.com/glauth/ldap"
)

const (
	BaseDN      = "dc=corp,dc=local"
	ServiceDN   = "cn=svc,dc=corp,dc=local"
	ServicePass = "svcpass"
)

// User — запись каталога.
type User struct {
	UID      string
	Display  string
	Mail     string
	Password string
	Groups   []string // DN групп (memberOf)
}

// DN пользователя.
func (u User) DN() string { return "uid=" + u.UID + ",ou=people," + BaseDN }

// Server — запущенный тестовый каталог.
type Server struct {
	URL string

	mu      sync.Mutex
	users   []User
	srv     *gl.Server
	Binds   int  // число привязок пользователей (проверка, что каталог вообще опрашивался)
	Failing bool // все привязки завершаются «сервер недоступен»
}

// Start запускает сервер на свободном порту и останавливает его по окончании теста.
func Start(t testing.TB, users []User) *Server {
	t.Helper()
	s := &Server{users: users}
	s.srv = gl.NewServer()
	s.srv.BindFunc("", s)
	s.srv.SearchFunc("", s)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ldaptest: %v", err)
	}
	s.URL = "ldap://" + ln.Addr().String()
	go func() { _ = s.srv.Serve(ln) }()
	t.Cleanup(func() { s.srv.Close(); ln.Close() })
	return s
}

// SetUsers заменяет содержимое каталога (например, чтобы изменить группы пользователя).
func (s *Server) SetUsers(users []User) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users = users
}

// Bind реализует gl.Binder.
func (s *Server) Bind(bindDN, pw string, _ net.Conn) (gl.LDAPResultCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Failing {
		return gl.LDAPResultUnavailable, nil
	}
	if strings.EqualFold(bindDN, ServiceDN) {
		if pw == ServicePass {
			return gl.LDAPResultSuccess, nil
		}
		return gl.LDAPResultInvalidCredentials, nil
	}
	s.Binds++
	for _, u := range s.users {
		if strings.EqualFold(u.DN(), bindDN) && pw != "" && u.Password == pw {
			return gl.LDAPResultSuccess, nil
		}
	}
	return gl.LDAPResultInvalidCredentials, nil
}

// Search реализует gl.Searcher: возвращает записи пользователей, подходящие под фильтр.
func (s *Server) Search(_ string, req gl.SearchRequest, _ net.Conn) (gl.ServerSearchResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := gl.CompileFilter(req.Filter)
	if err != nil {
		return gl.ServerSearchResult{ResultCode: gl.LDAPResultOperationsError}, err
	}
	var out []*gl.Entry
	for _, u := range s.users {
		e := &gl.Entry{DN: u.DN(), Attributes: []*gl.EntryAttribute{
			{Name: "objectClass", Values: []string{"top", "person", "user", "posixAccount"}},
			{Name: "objectCategory", Values: []string{"person"}},
			{Name: "uid", Values: []string{u.UID}},
			{Name: "sAMAccountName", Values: []string{u.UID}},
			{Name: "userPrincipalName", Values: []string{u.UID + "@corp.local"}},
			{Name: "cn", Values: []string{u.UID}},
			{Name: "displayName", Values: []string{u.Display}},
			{Name: "mail", Values: []string{u.Mail}},
			{Name: "memberOf", Values: u.Groups},
		}}
		if ok, _ := gl.ServerApplyFilter(f, e); ok {
			out = append(out, e)
		}
	}
	return gl.ServerSearchResult{Entries: out, ResultCode: gl.LDAPResultSuccess}, nil
}
