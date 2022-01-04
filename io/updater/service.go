package updater

import (
	"context"
	"database/sql"
	"github.com/viant/sqlx/io"
	"github.com/viant/sqlx/io/config"
	"github.com/viant/sqlx/option"
	"reflect"
	"sync"
)

//Service represents updater
type Service struct {
	*config.Config
	*session
	mux sync.Mutex
	db  *sql.DB
}

func (s *Service) Exec(ctx context.Context, any interface{}, options ...option.Option) (int64, error) {
	recordsFn, _, err := io.Iterator(any)
	if err != nil {
		return 0, err
	}
	record := recordsFn()
	var sess *session
	if sess, err = s.ensureSession(record); err != nil {
		return 0, err
	}
	if err = sess.begin(ctx, s.db, options); err != nil {
		return 0, err
	}

	if err = sess.prepare(ctx); err != nil {
		return 0, err
	}

	rowsAffected, err := s.update(ctx, record, recordsFn)
	err = s.end(err)
	return rowsAffected, err

}

func (s *Service) ensureSession(record interface{}) (*session, error) {
	s.mux.Lock()
	defer s.mux.Unlock()
	rType := reflect.TypeOf(record)
	if sess := s.session; sess != nil && sess.rType == rType {
		return &session{
			rType:         rType,
			identityIndex: sess.identityIndex,
			Config:        s.Config,
		}, nil
	}
	result := &session{
		rType:  rType,
		Config: s.Config,
	}
	err := result.init(record)
	if err == nil {
		s.session = result
	}
	return result, err
}

//New creates an updater
func New(ctx context.Context, db *sql.DB, tableName string, options ...option.Option) (*Service, error) {
	var columnMapper io.ColumnMapper
	if !option.Assign(options, &columnMapper) {
		columnMapper = io.StructColumnMapper
	}
	updater := &Service{
		Config: config.New(tableName),
		db:     db,
	}
	err := updater.ApplyOption(ctx, db, options...)
	return updater, err
}
