package driver

import (
	"context"
	"strings"
	"time"

	"github.com/ory/kratos/schema"

	"github.com/cenkalti/backoff"
	"github.com/gobuffalo/pop"
	"github.com/gorilla/sessions"
	"github.com/justinas/nosurf"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/ory/x/dbal"
	"github.com/ory/x/healthx"
	"github.com/ory/x/sqlcon"

	"github.com/ory/x/tracing"

	"github.com/ory/x/logrusx"

	"github.com/ory/kratos/courier"
	"github.com/ory/kratos/persistence"
	"github.com/ory/kratos/persistence/sql"
	"github.com/ory/kratos/selfservice/flow/login"
	"github.com/ory/kratos/selfservice/flow/logout"
	"github.com/ory/kratos/selfservice/flow/profile"
	"github.com/ory/kratos/selfservice/flow/registration"
	"github.com/ory/kratos/selfservice/strategy/oidc"

	"github.com/ory/herodot"

	"github.com/ory/kratos/driver/configuration"
	"github.com/ory/kratos/identity"
	"github.com/ory/kratos/selfservice/errorx"
	password2 "github.com/ory/kratos/selfservice/strategy/password"
	"github.com/ory/kratos/session"
)

var _ Registry = new(RegistryDefault)

func init() {
	dbal.RegisterDriver(func() dbal.Driver {
		return NewRegistryDefault()
	})
}

type RegistryDefault struct {
	l              logrus.FieldLogger
	c              configuration.Provider
	nosurf         *nosurf.CSRFHandler
	trc            *tracing.Tracer
	writer         herodot.Writer
	healthxHandler *healthx.Handler

	courier   *courier.Courier
	persister persistence.Persister

	identityHandler   *identity.Handler
	identityValidator *identity.Validator

	schemaHandler *schema.Handler

	sessionHandler *session.Handler
	sessionsStore  sessions.Store
	sessionManager session.Manager

	passwordHasher    password2.Hasher
	passwordValidator password2.Validator

	errorHandler *errorx.Handler
	errorManager *errorx.Manager

	selfserviceRegistrationExecutor            *registration.HookExecutor
	selfserviceRegistrationHandler             *registration.Handler
	seflserviceRegistrationErrorHandler        *registration.ErrorHandler
	selfserviceRegistrationRequestErrorHandler *registration.ErrorHandler

	selfserviceLoginExecutor            *login.HookExecutor
	selfserviceLoginHandler             *login.Handler
	selfserviceLoginRequestErrorHandler *login.ErrorHandler

	selfserviceProfileManagementHandler          *profile.Handler
	selfserviceProfileRequestRequestErrorHandler *profile.ErrorHandler

	selfserviceLogoutHandler *logout.Handler

	selfserviceStrategies []selfServiceStrategy

	buildVersion string
	buildHash    string
	buildDate    string
}

func NewRegistryDefault() *RegistryDefault {
	return &RegistryDefault{}
}

func (m *RegistryDefault) WithBuildInfo(version, hash, date string) Registry {
	m.buildVersion = version
	m.buildHash = hash
	m.buildDate = date
	return m
}

func (m *RegistryDefault) BuildVersion() string {
	return m.buildVersion
}

func (m *RegistryDefault) BuildDate() string {
	return m.buildDate
}

func (m *RegistryDefault) BuildHash() string {
	return m.buildHash
}

func (m *RegistryDefault) WithLogger(l logrus.FieldLogger) Registry {
	m.l = l
	return m
}

func (m *RegistryDefault) ProfileManagementHandler() *profile.Handler {
	if m.selfserviceProfileManagementHandler == nil {
		m.selfserviceProfileManagementHandler = profile.NewHandler(m, m.c)
	}
	return m.selfserviceProfileManagementHandler
}

func (m *RegistryDefault) ProfileRequestRequestErrorHandler() *profile.ErrorHandler {
	if m.selfserviceProfileRequestRequestErrorHandler == nil {
		m.selfserviceProfileRequestRequestErrorHandler = profile.NewErrorHandler(m, m.c)
	}
	return m.selfserviceProfileRequestRequestErrorHandler
}

func (m *RegistryDefault) LogoutHandler() *logout.Handler {
	if m.selfserviceLogoutHandler == nil {
		m.selfserviceLogoutHandler = logout.NewHandler(m, m.c)
	}
	return m.selfserviceLogoutHandler
}

func (m *RegistryDefault) HealthHandler() *healthx.Handler {
	if m.healthxHandler == nil {
		m.healthxHandler = healthx.NewHandler(m.Writer(), m.BuildVersion(), healthx.ReadyCheckers{
			"database": m.Ping,
		})
	}

	return m.healthxHandler
}

func (m *RegistryDefault) WithCSRFHandler(c *nosurf.CSRFHandler) {
	m.nosurf = c
}

func (m *RegistryDefault) CSRFHandler() *nosurf.CSRFHandler {
	if m.nosurf == nil {
		panic("csrf handler is not set")
	}
	return m.nosurf
}

func (m *RegistryDefault) selfServiceStrategies() []selfServiceStrategy {
	if m.selfserviceStrategies == nil {
		m.selfserviceStrategies = []selfServiceStrategy{
			password2.NewStrategy(m, m.c),
			oidc.NewStrategy(m, m.c),
		}
	}

	return m.selfserviceStrategies
}

func (m *RegistryDefault) RegistrationStrategies() registration.Strategies {
	strategies := make([]registration.Strategy, len(m.selfServiceStrategies()))
	for i := range strategies {
		strategies[i] = m.selfServiceStrategies()[i]
	}
	return strategies
}

func (m *RegistryDefault) LoginStrategies() login.Strategies {
	strategies := make([]login.Strategy, len(m.selfServiceStrategies()))
	for i := range strategies {
		strategies[i] = m.selfServiceStrategies()[i]
	}
	return strategies
}

func (m *RegistryDefault) IdentityValidator() *identity.Validator {
	if m.identityValidator == nil {
		m.identityValidator = identity.NewValidator(m)
	}
	return m.identityValidator
}

func (m *RegistryDefault) WithConfig(c configuration.Provider) Registry {
	m.c = c
	return m
}

func (m *RegistryDefault) Writer() herodot.Writer {
	if m.writer == nil {
		h := herodot.NewJSONWriter(m.Logger())
		m.writer = h
	}
	return m.writer
}

func (m *RegistryDefault) Logger() logrus.FieldLogger {
	if m.l == nil {
		m.l = logrusx.New()
	}
	return m.l
}

func (m *RegistryDefault) IdentityHandler() *identity.Handler {
	if m.identityHandler == nil {
		m.identityHandler = identity.NewHandler(m.c, m)
	}
	return m.identityHandler
}

func (m *RegistryDefault) SchemaHandler() *schema.Handler {
	if m.schemaHandler == nil {
		m.schemaHandler = schema.NewHandler(m)
	}
	return m.schemaHandler
}

func (m *RegistryDefault) SessionHandler() *session.Handler {
	if m.sessionHandler == nil {
		m.sessionHandler = session.NewHandler(m)
	}
	return m.sessionHandler
}

func (m *RegistryDefault) PasswordHasher() password2.Hasher {
	if m.passwordHasher == nil {
		m.passwordHasher = password2.NewHasherArgon2(m.c)
	}
	return m.passwordHasher
}

func (m *RegistryDefault) PasswordValidator() password2.Validator {
	if m.passwordValidator == nil {
		m.passwordValidator = password2.NewDefaultPasswordValidatorStrategy()
	}
	return m.passwordValidator
}

func (m *RegistryDefault) SelfServiceErrorHandler() *errorx.Handler {
	if m.errorHandler == nil {
		m.errorHandler = errorx.NewHandler(m)
	}
	return m.errorHandler
}

func (m *RegistryDefault) CookieManager() sessions.Store {
	if m.sessionsStore == nil {
		cs := sessions.NewCookieStore(m.c.SessionSecrets()...)
		cs.Options.Secure = !m.c.IsInsecureDevMode()
		cs.Options.HttpOnly = true
		m.sessionsStore = cs
	}
	return m.sessionsStore
}

func (m *RegistryDefault) Tracer() *tracing.Tracer {
	if m.trc == nil {
		m.trc = &tracing.Tracer{
			ServiceName:  m.c.TracingServiceName(),
			JaegerConfig: m.c.TracingJaegerConfig(),
			Provider:     m.c.TracingProvider(),
			Logger:       m.Logger(),
		}

		if err := m.trc.Setup(); err != nil {
			m.Logger().WithError(err).Fatalf("Unable to initialize Tracer.")
		}
	}

	return m.trc
}

func (m *RegistryDefault) SessionManager() session.Manager {
	if m.sessionManager == nil {
		m.sessionManager = session.NewManagerHTTP(m.c, m)
	}
	return m.sessionManager
}

func (m *RegistryDefault) SelfServiceErrorManager() *errorx.Manager {
	if m.errorManager == nil {
		m.errorManager = errorx.NewManager(m, m.c)
	}
	return m.errorManager
}

func (m *RegistryDefault) CanHandle(dsn string) bool {
	return dsn == "memory" ||
		strings.HasPrefix(dsn, "mysql") ||
		strings.HasPrefix(dsn, "sqlite") ||
		strings.HasPrefix(dsn, "sqlite3") ||
		strings.HasPrefix(dsn, "postgres") ||
		strings.HasPrefix(dsn, "postgresql") ||
		strings.HasPrefix(dsn, "cockroach") ||
		strings.HasPrefix(dsn, "cockroachdb") ||
		strings.HasPrefix(dsn, "crdb")
}

func (m *RegistryDefault) Init() error {
	if m.persister != nil {
		panic("RegistryDefault.Init() must not be called more than once.")
	}

	bc := backoff.NewExponentialBackOff()
	bc.MaxElapsedTime = time.Minute * 5
	bc.Reset()
	return errors.WithStack(
		backoff.Retry(func() error {
			pool, idlePool, connMaxLifetime := sqlcon.ParseConnectionOptions(m.l, m.c.DSN())
			c, err := pop.NewConnection(&pop.ConnectionDetails{
				URL:             m.c.DSN(),
				IdlePool:        idlePool,
				ConnMaxLifetime: connMaxLifetime,
				Pool:            pool,
			})
			if err != nil {
				m.Logger().WithError(err).Warnf("Unable to connect to database, retrying.")
				return errors.WithStack(err)
			}
			if err := c.Open(); err != nil {
				m.Logger().WithError(err).Warnf("Unable to open database, retrying.")
				return errors.WithStack(err)
			}
			p, err := sql.NewPersister(m, m.c, c)
			if err != nil {
				m.Logger().WithError(err).Warnf("Unable to initialize persister, retrying.")
				return err
			}
			if err := p.Ping(context.Background()); err != nil {
				m.Logger().WithError(err).Warnf("Unable to ping database, retrying.")
				return err
			}
			m.persister = p
			return nil
		}, bc),
	)
}

func (m *RegistryDefault) Courier() *courier.Courier {
	if m.courier == nil {
		m.courier = courier.NewSMTP(m, m.c)
	}
	return m.courier
}

func (m *RegistryDefault) IdentityPool() identity.Pool {
	return m.persister
}

func (m *RegistryDefault) RegistrationRequestPersister() registration.RequestPersister {
	return m.persister
}

func (m *RegistryDefault) LoginRequestPersister() login.RequestPersister {
	return m.persister
}

func (m *RegistryDefault) ProfileRequestPersister() profile.RequestPersister {
	return m.persister
}

func (m *RegistryDefault) SelfServiceErrorPersister() errorx.Persister {
	return m.persister
}

func (m *RegistryDefault) SessionPersister() session.Persister {
	return m.persister
}

func (m *RegistryDefault) CourierPersister() courier.Persister {
	return m.persister
}

func (m *RegistryDefault) Persister() persistence.Persister {
	return m.persister
}

func (m *RegistryDefault) Ping() error {
	return m.persister.Ping(context.Background())
}
