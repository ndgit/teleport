/*
Copyright 2021 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package desktop

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
	"github.com/sirupsen/logrus"

	"github.com/gravitational/trace"

	"github.com/gravitational/teleport"
	"github.com/gravitational/teleport/api/client/proto"
	apidefaults "github.com/gravitational/teleport/api/defaults"
	"github.com/gravitational/teleport/api/types"
	"github.com/gravitational/teleport/lib/auth"
	"github.com/gravitational/teleport/lib/defaults"
	"github.com/gravitational/teleport/lib/limiter"
	"github.com/gravitational/teleport/lib/srv"
	"github.com/gravitational/teleport/lib/srv/desktop/deskproto"
	"github.com/gravitational/teleport/lib/srv/desktop/rdp/rdpclient"
	"github.com/gravitational/teleport/lib/utils"
)

// WindowsService implements the RDP-based Windows desktop access service.
//
// This service accepts mTLS connections from the proxy, establishes RDP
// connections to Windows hosts and translates RDP into Teleport's desktop
// protocol.
type WindowsService struct {
	cfg        WindowsServiceConfig
	middleware *auth.Middleware

	closeCtx context.Context
	close    func()
}

// WindowsServiceConfig contains all necessary configuration values for a
// WindowsService.
type WindowsServiceConfig struct {
	// Log is the logging sink for the service.
	Log logrus.FieldLogger
	// Clock provides current time.
	Clock clockwork.Clock
	// TLS is the TLS server configuration.
	TLS *tls.Config
	// AccessPoint is the Auth API client (with caching).
	AccessPoint auth.AccessPoint
	// AuthClient is the Auth API client (without caching).
	AuthClient auth.ClientI
	// ConnLimiter limits the number of active connections per client IP.
	ConnLimiter *limiter.ConnectionsLimiter
	// Heartbeat contains configuration for service heartbeats.
	Heartbeat HeartbeatConfig
	// LDAPConfig contains parameters for connecting to an LDAP server.
	LDAPConfig
}

// LDAPConfig contains parameters for connecting to an LDAP server.
type LDAPConfig struct {
	Addr     string
	Domain   string
	Username string
	Password string
}

func (cfg LDAPConfig) check() error {
	if cfg.Addr == "" {
		return trace.BadParameter("missing Addr in LDAPConfig")
	}
	if cfg.Domain == "" {
		return trace.BadParameter("missing Domain in LDAPConfig")
	}
	if cfg.Username == "" {
		return trace.BadParameter("missing Username in LDAPConfig")
	}
	if cfg.Password == "" {
		return trace.BadParameter("missing Password in LDAPConfig")
	}
	return nil
}

func (cfg LDAPConfig) connect() (ldap.Client, error) {
	con, err := ldap.Dial("tcp", cfg.Addr)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	// TODO(awly): should we get a CA cert for the LDAP cert validation?
	if err := con.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil {
		con.Close()
		return nil, trace.Wrap(err)
	}
	if err := con.Bind(cfg.Username, cfg.Password); err != nil {
		con.Close()
		return nil, trace.Wrap(err)
	}
	return con, nil
}

// HeartbeatConfig contains the configuration for service heartbeats.
type HeartbeatConfig struct {
	// HostUUID is the UUID of the host that this service runs on. Used as the
	// name of the created API object.
	HostUUID string
	// PublicAddr is the public address of this service.
	PublicAddr string
	// OnHeartbeat is called after each heartbeat attempt.
	OnHeartbeat func(error)
	// StaticHosts is an optional list of static Windows hosts to register.
	StaticHosts []utils.NetAddr
}

func (cfg *WindowsServiceConfig) CheckAndSetDefaults() error {
	if cfg.Log == nil {
		cfg.Log = logrus.New().WithField(trace.Component, teleport.ComponentWindowsDesktop)
	}
	if cfg.Clock == nil {
		cfg.Clock = clockwork.NewRealClock()
	}
	if cfg.TLS == nil {
		return trace.BadParameter("WindowsServiceConfig is missing TLS")
	}
	if cfg.AccessPoint == nil {
		return trace.BadParameter("WindowsServiceConfig is missing AccessPoint")
	}
	if cfg.AuthClient == nil {
		return trace.BadParameter("WindowsServiceConfig is missing AuthClient")
	}
	if cfg.ConnLimiter == nil {
		return trace.BadParameter("WindowsServiceConfig is missing ConnLimiter")
	}
	if err := cfg.Heartbeat.CheckAndSetDefaults(); err != nil {
		return trace.Wrap(err)
	}
	if err := cfg.LDAPConfig.check(); err != nil {
		return trace.Wrap(err)
	}
	return nil
}

func (cfg *HeartbeatConfig) CheckAndSetDefaults() error {
	if cfg.HostUUID == "" {
		return trace.BadParameter("HeartbeatConfig is missing HostUUID")
	}
	if cfg.PublicAddr == "" {
		return trace.BadParameter("HeartbeatConfig is missing PublicAddr")
	}
	if cfg.OnHeartbeat == nil {
		return trace.BadParameter("HeartbeatConfig is missing OnHeartbeat")
	}
	return nil
}

// NewWindowsService initializes a new WindowsService.
//
// To start serving connections, call Serve.
// When done serving connections, call Close.
func NewWindowsService(cfg WindowsServiceConfig) (*WindowsService, error) {
	if err := cfg.CheckAndSetDefaults(); err != nil {
		return nil, trace.Wrap(err)
	}
	ctx, close := context.WithCancel(context.Background())
	s := &WindowsService{
		cfg: cfg,
		middleware: &auth.Middleware{
			AccessPoint:   cfg.AccessPoint,
			AcceptedUsage: []string{teleport.UsageWindowsDesktopOnly},
		},
		closeCtx: ctx,
		close:    close,
	}

	// TODO(awly): session recording.
	// TODO(awly): user locking.

	if err := s.startServiceHeartbeat(); err != nil {
		return nil, trace.Wrap(err)
	}

	// TODO(awly): fetch registered hosts automatically.
	if err := s.startStaticHostHeartbeats(); err != nil {
		return nil, trace.Wrap(err)
	}

	// TODO(awly): publish Teleport CA cert.

	// Push an empty valid CRL into LDAP once at startup.
	// The CRL should be valid for a year, so we don't have to periodically
	// update it.
	if err := s.updateCRL(ctx); err != nil {
		return nil, trace.Wrap(err)
	}

	return s, nil
}

func (s *WindowsService) startServiceHeartbeat() error {
	heartbeat, err := srv.NewHeartbeat(srv.HeartbeatConfig{
		Context:         s.closeCtx,
		Component:       teleport.ComponentWindowsDesktop,
		Mode:            srv.HeartbeatModeWindowsDesktopService,
		Announcer:       s.cfg.AccessPoint,
		GetServerInfo:   s.getServiceHeartbeatInfo,
		KeepAlivePeriod: apidefaults.ServerKeepAliveTTL,
		AnnouncePeriod:  apidefaults.ServerAnnounceTTL/2 + utils.RandomDuration(apidefaults.ServerAnnounceTTL/10),
		CheckPeriod:     defaults.HeartbeatCheckPeriod,
		ServerTTL:       apidefaults.ServerAnnounceTTL,
		OnHeartbeat:     s.cfg.Heartbeat.OnHeartbeat,
	})
	if err != nil {
		return trace.Wrap(err)
	}
	go func() {
		if err := heartbeat.Run(); err != nil {
			s.cfg.Log.WithError(err).Error("Heartbeat ended with error")
		}
	}()
	return nil
}

// startStaticHostHeartbeats spawns heartbeat routines for all static hosts in
// this service. We use heartbeats instead of registering once at startup to
// support expiration.
//
// When a WindowsService with a list of static hosts disappears, those hosts
// should eventually get cleaned up. But they should exist as long as the
// service itself is running.
func (s *WindowsService) startStaticHostHeartbeats() error {
	for _, host := range s.cfg.Heartbeat.StaticHosts {
		heartbeat, err := srv.NewHeartbeat(srv.HeartbeatConfig{
			Context:         s.closeCtx,
			Component:       teleport.ComponentWindowsDesktop,
			Mode:            srv.HeartbeatModeWindowsDesktop,
			Announcer:       s.cfg.AccessPoint,
			GetServerInfo:   s.getHostHeartbeatInfo(host),
			KeepAlivePeriod: apidefaults.ServerKeepAliveTTL,
			AnnouncePeriod:  apidefaults.ServerAnnounceTTL/2 + utils.RandomDuration(apidefaults.ServerAnnounceTTL/10),
			CheckPeriod:     defaults.HeartbeatCheckPeriod,
			ServerTTL:       apidefaults.ServerAnnounceTTL,
			OnHeartbeat:     s.cfg.Heartbeat.OnHeartbeat,
		})
		if err != nil {
			return trace.Wrap(err)
		}
		go func() {
			if err := heartbeat.Run(); err != nil {
				s.cfg.Log.WithError(err).Error("Heartbeat ended with error")
			}
		}()
	}
	return nil
}

// Close instructs the server to stop accepting new connections and abort all
// established ones. Close does not wait for the connections to be finished.
func (s *WindowsService) Close() error {
	s.close()
	return nil
}

// Serve starts serving TLS connections for plainLis. plainLis should be a TCP
// listener and Serve will handle TLS internally.
func (s *WindowsService) Serve(plainLis net.Listener) error {
	lis := tls.NewListener(plainLis, s.cfg.TLS)
	defer lis.Close()
	for {
		select {
		case <-s.closeCtx.Done():
			return trace.Wrap(s.closeCtx.Err())
		default:
		}

		con, err := lis.Accept()
		if err != nil {
			if utils.IsOKNetworkError(err) || trace.IsConnectionProblem(err) {
				return nil
			}
			return trace.Wrap(err)
		}

		go s.handleConnection(con)
	}
}

func (s *WindowsService) handleConnection(con net.Conn) {
	defer con.Close()
	log := s.cfg.Log

	// Check connection limits.
	remoteAddr, _, err := net.SplitHostPort(con.RemoteAddr().String())
	if err != nil {
		log.WithError(err).Errorf("Could not parse client IP from %q", con.RemoteAddr().String())
		return
	}
	log = log.WithField("client-ip", remoteAddr)
	if err := s.cfg.ConnLimiter.AcquireConnection(remoteAddr); err != nil {
		log.WithError(err).Warning("Connection limit exceeded, rejecting connection")
		return
	}
	defer s.cfg.ConnLimiter.ReleaseConnection(remoteAddr)

	// Authenticate the client.
	tlsCon, ok := con.(*tls.Conn)
	if !ok {
		log.Errorf("Got %T from TLS listener, expected *tls.Conn", con)
		return
	}
	ctx, err := s.middleware.WrapContextWithUser(s.closeCtx, tlsCon)
	if err != nil {
		log.WithError(err).Warning("mTLS authentication failed for incoming connection")
		return
	}
	log.Debug("Authenticated Windows desktop connection")

	desktopUUID := strings.TrimSuffix(tlsCon.ConnectionState().ServerName, SNISuffix)
	log = log.WithField("desktop-uuid", desktopUUID)
	desktop, err := s.cfg.AccessPoint.GetWindowsDesktop(ctx, desktopUUID)
	if err != nil {
		log.WithError(err).Warning("Failed to fetch desktop by UUID")
		return
	}
	log = log.WithField("desktop-addr", desktop.GetAddr())
	log.Debug("Connecting to Windows desktop")
	defer log.Debug("Windows desktop disconnected")

	// TODO(awly): authorization

	if err := s.connectRDP(ctx, log, tlsCon, desktop); err != nil {
		log.WithError(err).Error("RDP connection failed")
		return
	}
}

func (s *WindowsService) connectRDP(ctx context.Context, log logrus.FieldLogger, con net.Conn, desktop types.WindowsDesktop) error {
	dpc := deskproto.NewConn(con)
	rdpc, err := rdpclient.New(ctx, rdpclient.Config{
		Log: log,
		GenerateUserCert: func(ctx context.Context, username string) (certDER, keyDER []byte, err error) {
			return s.generateCredentials(ctx, username, desktop.GetDomain())
		},
		Addr:          desktop.GetAddr(),
		InputMessage:  dpc.InputMessage,
		OutputMessage: dpc.OutputMessage,
	})
	if err != nil {
		return trace.Wrap(err)
	}
	return trace.Wrap(rdpc.Wait())
}

func (s *WindowsService) getServiceHeartbeatInfo() (types.Resource, error) {
	srv, err := types.NewWindowsDesktopServiceV3(
		s.cfg.Heartbeat.HostUUID,
		types.WindowsDesktopServiceSpecV3{
			Addr:            s.cfg.Heartbeat.PublicAddr,
			TeleportVersion: teleport.Version,
		})
	if err != nil {
		return nil, trace.Wrap(err)
	}
	srv.SetExpiry(s.cfg.Clock.Now().UTC().Add(apidefaults.ServerAnnounceTTL))
	return srv, nil
}

func (s *WindowsService) getHostHeartbeatInfo(netAddr utils.NetAddr) func() (types.Resource, error) {
	return func() (types.Resource, error) {
		addr := netAddr.String()
		name, err := s.nameForStaticHost(addr)
		if err != nil {
			return nil, trace.Wrap(err)
		}
		desktop, err := types.NewWindowsDesktopV3(
			name,
			nil, // TODO(awly): set RBAC labels.
			types.WindowsDesktopSpecV3{
				Addr:   addr,
				Domain: s.cfg.Domain,
			})
		if err != nil {
			return nil, trace.Wrap(err)
		}
		desktop.SetExpiry(s.cfg.Clock.Now().UTC().Add(apidefaults.ServerAnnounceTTL))
		return desktop, nil
	}
}

// nameForStaticHost attempts to find the UUID of an existing Windows desktop
// with the same address. If no matching address is found, a new UUID is
// generated.
//
// The list of WindowsDesktop objects should be read from the local cache. It
// should be reasonably fast to do this scan on every heartbeat. However, with
// a very large number of desktops in the cluster, this may use up a lot of CPU
// time.
//
// TODO(awly): think of an alternative way to not duplicate desktop objects
// coming from different windows_desktop_services.
func (s *WindowsService) nameForStaticHost(addr string) (string, error) {
	desktops, err := s.cfg.AccessPoint.GetWindowsDesktops(s.closeCtx)
	if err != nil {
		return "", trace.Wrap(err)
	}
	for _, d := range desktops {
		if d.GetAddr() == addr {
			return d.GetName(), nil
		}
	}
	return uuid.New().String(), nil
}

// crlDN returns the distinguished name (DN) of the CRL entry in the LDAP
// server (ActiveDirectory), along with its parent container entry DN.
func (s *WindowsService) crlDN(ctx context.Context, domain string) (dn string, container string, err error) {
	clusterName, err := s.cfg.AccessPoint.GetClusterName()
	if err != nil {
		return "", "", trace.Wrap(err, "fetching cluster name: %v", err)
	}

	// Here's an example DN:
	//
	// CN=mycluster,CN=Teleport,CN=CDP,CN=Public Key Services,CN=Services,CN=Configuration,DC=example,DC=com
	//
	// You read it backwards:
	// - DC=example,DC=com means "example.com" domain
	// - CN=mycluster,CN=Teleport,CN=CDP,CN=Public Key Services,CN=Services,CN=Configuration
	//   means "Configuration/Services/Public Key Services/CDP/Teleport/mycluster"
	//   entry, where "mycluster" is the Teleport cluster name
	//
	// CDP stands for CRL Distribution Point, this is where the Windows CA
	// stores its own CRLs and we mimic it.

	domainParts := strings.Split(domain, ".")
	parentDN := "CN=Teleport,CN=CDP,CN=Public Key Services,CN=Services,CN=Configuration"
	for _, dc := range domainParts {
		parentDN += fmt.Sprintf(",DC=%s", dc)
	}

	crlDN := fmt.Sprintf("CN=%s,%s", clusterName.GetClusterName(), parentDN)
	return crlDN, parentDN, nil
}

func (s *WindowsService) updateCRL(ctx context.Context) error {
	// Publish the CRL for current cluster CA. For trusted clusters, their
	// respective windows_desktop_services will publish CRLs of their CAs so we
	// don't have to do it here.
	crlDER, err := s.cfg.AccessPoint.GenerateCertAuthorityCRL(ctx, types.UserCA)
	if err != nil {
		return trace.Wrap(err, "generating CRL: %v", err)
	}
	crlDN, parentDN, err := s.crlDN(ctx, s.cfg.LDAPConfig.Domain)
	if err != nil {
		return trace.Wrap(err)
	}
	con, err := s.cfg.LDAPConfig.connect()
	if err != nil {
		return trace.Wrap(err, "connecting to LDAP server: %v", err)
	}
	defer con.Close()

	// Try to create the parent container first, ignore the error if it already
	// exists.
	areq := ldap.NewAddRequest(parentDN, nil)
	areq.Attribute("objectClass", []string{"container"})
	if err := con.Add(areq); err != nil && !ldap.IsErrorWithCode(err, ldap.LDAPResultEntryAlreadyExists) {
		return trace.Wrap(err, "creating container: %v", err)
	}

	// Try to create the CRL entry.
	// If it exists, issue a Modify request to replace the data.
	areq = ldap.NewAddRequest(crlDN, nil)
	areq.Attribute("objectClass", []string{"cRLDistributionPoint"})
	areq.Attribute("certificateRevocationList", []string{string(crlDER)})
	if err := con.Add(areq); err != nil {
		if !ldap.IsErrorWithCode(err, ldap.LDAPResultEntryAlreadyExists) {
			return trace.Wrap(err, "creating CRL: %v", err)
		}
		mreq := ldap.NewModifyRequest(crlDN, nil)
		mreq.Replace("certificateRevocationList", []string{string(crlDER)})
		if err := con.Modify(mreq); err != nil {
			if err != nil {
				return trace.Wrap(err, "updating CRL: %v", err)
			}
		}
		s.cfg.Log.Info("Updated CRL for Windows logins via LDAP")
	} else {
		s.cfg.Log.Info("Added CRL for Windows logins via LDAP")
	}
	return nil
}

func (s *WindowsService) generateCredentials(ctx context.Context, username, domain string) (certDER, keyDER []byte, err error) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}
	keyDER = x509.MarshalPKCS1PrivateKey(rsaKey)

	// Generate the Windows-compatible certificate, see
	// https://docs.microsoft.com/en-us/troubleshoot/windows-server/windows-security/enabling-smart-card-logon-third-party-certification-authorities
	// for requirements.
	san, err := subjectAltNameExtension(username, domain)
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}
	csr := &x509.CertificateRequest{
		Subject: pkix.Name{CommonName: username},
		// We have to pass SAN and ExtKeyUsage as raw extensions because
		// crypto/x509 doesn't support what we need:
		// - x509.ExtKeyUsage doesn't have the Smartcard Logon variant
		// - x509.CertificateRequest doesn't have OtherName SAN fields (which
		//   is a type of SAN distinct from DNSNames, EmailAddresses, IPAddresses
		//   and URIs)
		ExtraExtensions: []pkix.Extension{
			extKeyUsageExtension,
			san,
		},
	}
	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, csr, rsaKey)
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}
	csrPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrBytes})
	// Note: this CRL DN may or may not be the same DN published in updateCRL.
	//
	// There can be multiple AD domains connected to Teleport. Each
	// windows_desktop_service is connected to a single AD domain and publishes
	// CRLs in it. Each service can also handle RDP connections for a different
	// domain, with the assumption that some other windows_desktop_service
	// published a CRL there.
	//
	// In other words, the domain var below may not be the same as
	// s.cfg.LDAPConfig.Domain.
	crlDN, _, err := s.crlDN(ctx, domain)
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}
	genResp, err := s.cfg.AuthClient.GenerateWindowsDesktopCert(ctx, &proto.WindowsDesktopCertRequest{
		CSR: csrPEM,
		// LDAP URI pointing at the CRL created with updateCRL.
		//
		// The full format is:
		// ldap://domain_controller_addr/distinguished_name_and_parameters.
		//
		// Using ldap:///distinguished_name_and_parameters (with empty
		// domain_controller_addr) will cause Windows to fetch the CRL from any
		// of its current domain controllers.
		CRLEndpoint: fmt.Sprintf("ldap:///%s?certificateRevocationList?base?objectClass=cRLDistributionPoint", crlDN),
		TTL:         proto.Duration(5 * time.Minute),
	})
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}
	certBlock, _ := pem.Decode(genResp.Cert)
	certDER = certBlock.Bytes
	return certDER, keyDER, nil
}

var extKeyUsageExtension = pkix.Extension{
	Id: asn1.ObjectIdentifier{2, 5, 29, 37}, // Extended Key Usage OID.
	Value: func() []byte {
		val, err := asn1.Marshal([]asn1.ObjectIdentifier{
			{1, 3, 6, 1, 5, 5, 7, 3, 2},       // Client Authentication OID.
			{1, 3, 6, 1, 4, 1, 311, 20, 2, 2}, // Smartcard Logon OID.
		})
		if err != nil {
			panic(err)
		}
		return val
	}(),
}

func subjectAltNameExtension(user, domain string) (pkix.Extension, error) {
	// Setting otherName SAN according to
	// https://samfira.com/2020/05/16/golang-x-509-certificates-and-othername/
	//
	// othernName SAN is needed to pass the UPN of the user, per
	// https://docs.microsoft.com/en-us/troubleshoot/windows-server/windows-security/enabling-smart-card-logon-third-party-certification-authorities
	ext := pkix.Extension{Id: asn1.ObjectIdentifier{2, 5, 29, 17}}
	var err error
	ext.Value, err = asn1.Marshal(
		subjectAltName{
			OtherName: otherName{
				OID: asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 311, 20, 2, 3}, // UPN OID
				Value: upn{
					Value: fmt.Sprintf("%s@%s", user, domain), // TODO(awly): sanitize username to avoid domain spoofing
				},
			},
		},
	)
	if err != nil {
		return ext, trace.Wrap(err)
	}
	return ext, nil
}

// Types for ASN.1 SAN serialization.

type subjectAltName struct {
	OtherName otherName `asn1:"tag:0"`
}

type otherName struct {
	OID   asn1.ObjectIdentifier
	Value upn `asn1:"tag:0"`
}

type upn struct {
	Value string `asn1:"utf8"`
}
