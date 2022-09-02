package cache

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/francoispqt/gojay"
	"github.com/google/uuid"
	"github.com/viant/afs"
	"github.com/viant/afs/option"
	"github.com/viant/sqlx/io"
	"github.com/viant/xunsafe"
	"hash/fnv"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	NotExistStatus = iota
	InUseStatus
	ErrorStatus
	ExistsStatus
)

type (
	ScannerFn func(args ...interface{}) error
	Service   struct {
		storage   string
		afs       afs.Service
		ttl       time.Duration
		extension string

		scanTypes []reflect.Type
		dataTypes []string

		mux       sync.RWMutex
		signature string
		canWrite  map[string]bool
		stream    *option.Stream
		recorder  Recorder
	}
)

//NewCache creates new cache.
func NewCache(URL string, ttl time.Duration, signature string, stream *option.Stream, options ...interface{}) (*Service, error) {
	var recorder Recorder
	for _, anOption := range options {
		switch actual := anOption.(type) {
		case Recorder:
			recorder = actual
		}
	}

	if URL[len(URL)-1] != '/' {
		URL += "/"
	}
	cache := &Service{
		afs:       afs.New(),
		ttl:       ttl,
		storage:   URL,
		extension: ".json",
		signature: signature,
		canWrite:  map[string]bool{},
		stream:    stream,
		recorder:  recorder,
	}

	return cache, nil
}

func (c *Service) Get(ctx context.Context, SQL string, args []interface{}) (*Entry, error) {
	URL, err := c.generateURL(SQL, args)
	if err != nil {
		return nil, err
	}

	if c.mark(URL) {
		return nil, nil
	}

	entry, err := c.getEntry(ctx, SQL, args, err, URL)
	if err != nil || entry == nil {
		c.unmark(URL)
		return entry, err
	}

	if entry.Has() {
		c.unmark(URL)
	}

	return entry, err
}

func (c *Service) getEntry(ctx context.Context, SQL string, args []interface{}, err error, URL string) (*Entry, error) {
	argsMarshal, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	entry := &Entry{
		Meta: Meta{
			SQL:       SQL,
			Args:      argsMarshal,
			url:       URL,
			Signature: c.signature,
		},
	}

	status, err := c.updateEntry(ctx, err, URL, entry)
	if err != nil {
		return nil, err
	}

	switch status {
	case InUseStatus:
		return nil, nil
	case ErrorStatus:
		return nil, err
	}

	return entry, nil
}

func (c *Service) updateEntry(ctx context.Context, err error, URL string, entry *Entry) (int, error) {
	status, err := c.readData(ctx, entry)
	if status == NotExistStatus || status == InUseStatus || err != nil {
		if status == NotExistStatus {
			id := strings.ReplaceAll(uuid.New().String(), "-", "")
			entry.Meta.url += id
			entry.Id = id
		}

		if err == nil {
			c.mux.RLock()
			c.canWrite[URL] = false
			c.mux.RUnlock()
		}

		return status, err
	}

	metaCorrect, err := c.checkMeta(entry.reader, &entry.Meta)
	if !metaCorrect || err != nil {
		return status, c.afs.Delete(ctx, URL)
	}
	return status, nil
}

func (c *Service) checkMeta(dataReader *bufio.Reader, entryMeta *Meta) (bool, error) {
	data, err := readLine(dataReader)
	meta := Meta{}
	if err = json.Unmarshal(data, &meta); err != nil {
		return false, nil
	}

	if c.expired(meta) || c.wrongSignature(meta, entryMeta) || c.wrongSQL(meta, entryMeta) || c.wrongArgs(meta, entryMeta) {
		return false, nil
	}

	entryMeta.Type = meta.Type
	entryMeta.Fields = meta.Fields

	for _, field := range entryMeta.Fields {
		if err = field.init(); err != nil {
			return false, err
		}
	}

	return true, nil
}

func (c *Service) generateURL(SQL string, args []interface{}) (string, error) {
	argMarshal, err := json.Marshal(args)
	if err != nil {
		return "", err
	}

	hasher := fnv.New64()
	_, err = hasher.Write(append([]byte(SQL), argMarshal...))

	if err != nil {
		return "", err
	}

	entryKey := strconv.Itoa(int(hasher.Sum64()))
	return c.storage + entryKey + c.extension, nil
}

func (c *Service) readData(ctx context.Context, entry *Entry) (int, error) {
	if ok, err := c.afs.Exists(ctx, entry.Meta.url); !ok || err != nil {
		return NotExistStatus, nil
	}

	afsReader, err := c.afs.OpenURL(ctx, entry.Meta.url, c.stream)
	if isRateError(err) || isPreConditionError(err) {
		return InUseStatus, nil
	}

	if err != nil {
		return ErrorStatus, nil
	}

	reader := bufio.NewReader(afsReader)
	if err != nil {
		return ErrorStatus, err
	}

	entry.reader = reader
	entry.readCloser = afsReader
	return ExistsStatus, nil
}

func (c *Service) wrongArgs(meta Meta, entryMeta *Meta) bool {
	return !bytes.Equal(meta.Args, entryMeta.Args)
}

func (c *Service) wrongSQL(meta Meta, entryMeta *Meta) bool {
	return meta.SQL != entryMeta.SQL
}

func (c *Service) wrongSignature(meta Meta, entryMeta *Meta) bool {
	return meta.Signature != entryMeta.Signature
}

func (c *Service) expired(meta Meta) bool {
	return int(Now().UnixNano()) > meta.TimeToLive
}

func (c *Service) writeMeta(ctx context.Context, m *Entry) (*bufio.Writer, error) {
	writer, err := c.afs.NewWriter(ctx, m.Meta.url, 0644, &option.SkipChecksum{Skip: true})
	if err != nil {
		return nil, err
	}

	m.writeCloser = writer
	bufioWriter := bufio.NewWriterSize(writer, 2048)

	m.Meta.TimeToLive = int(Now().Add(c.ttl).UnixNano())
	data, err := json.Marshal(m.Meta)
	if err != nil {
		return nil, err
	}

	if err = c.write(bufioWriter, data, false); err != nil {
		return bufioWriter, err
	}

	return bufioWriter, nil
}

func (c *Service) write(bufioWriter *bufio.Writer, data []byte, addNewLine bool) error {
	if addNewLine {
		if err := bufioWriter.WriteByte('\n'); err != nil {
			return err
		}
	}

	_, err := bufioWriter.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func (c *Service) UpdateType(ctx context.Context, entry *Entry, values []interface{}) (bool, error) {
	c.initializeCacheType(values)

	if len(entry.Meta.Type) > 0 && !c.matchesType(entry.Meta.Type) {
		return false, c.Delete(ctx, entry)
	}

	entry.Meta.Type = c.dataTypes
	return true, nil
}

func (c *Service) initializeCacheType(values []interface{}) {
	c.mux.Lock()
	if len(c.scanTypes) > 0 {
		c.mux.Unlock()
		return
	}

	defer c.mux.Unlock()
	c.scanTypes = make([]reflect.Type, len(values))
	c.dataTypes = make([]string, len(values))
	for i, value := range values {
		rValue := reflect.ValueOf(value)
		valueType := rValue.Type()
		c.scanTypes[i] = valueType.Elem()
		c.dataTypes[i] = c.scanTypes[i].String()
	}
}

func (c *Service) Delete(ctx context.Context, entry *Entry) error {
	return c.afs.Delete(ctx, entry.Meta.url)
}

func (c *Service) mark(URL string) bool {
	c.mux.RLock()
	_, isInMap := c.canWrite[URL]
	c.canWrite[URL] = false
	c.mux.RUnlock()
	return isInMap
}

func (c *Service) unmark(url string) {
	c.mux.RLock()
	delete(c.canWrite, url)
	c.mux.RUnlock()
}

func (c *Service) scanner(e *Entry) ScannerFn {
	var decoder *Decoder

	var err error
	return func(values ...interface{}) error {
		if c.recorder != nil {
			c.recorder.ScanValues(values)
		}

		if len(values) != len(c.scanTypes) {
			return fmt.Errorf("invalid cache format, expected to have %v values but got %v", len(values), len(c.scanTypes))
		}

		if decoder == nil {
			decoder = NewDecoder(c.scanTypes)
		}

		if err = gojay.UnmarshalJSONArray(e.Data, decoder); err != nil {
			return err
		}

		for i, cachedValue := range decoder.values {
			destPtr := xunsafe.AsPointer(values[i])
			srcPtr := xunsafe.AsPointer(cachedValue)
			if destPtr == nil || srcPtr == nil {
				continue
			}

			xunsafe.Copy(destPtr, srcPtr, int(c.scanTypes[i].Size()))
		}

		e.index++
		decoder.reset()
		return err
	}
}

func (c *Service) Close(ctx context.Context, e *Entry) error {
	err := c.close(ctx, e)
	if err != nil {
		_ = c.Delete(ctx, e)
		return err
	}

	return nil
}

func (c *Service) close(ctx context.Context, e *Entry) error {
	if e.Has() {
		return e.readCloser.Close()
	}

	if e.writer == nil {
		return nil
	}

	actualURL := strings.ReplaceAll(e.Meta.url, ".json"+e.Id, ".json")
	c.unmark(actualURL)
	if err := e.writer.Flush(); err != nil {
		return err
	}

	if err := e.writeCloser.Close(); err != nil {
		return err
	}

	if err := c.afs.Move(ctx, e.Meta.url, actualURL); err != nil {
		return err
	}

	return nil
}

func (c *Service) AddValues(ctx context.Context, e *Entry, values []interface{}) error {
	if c.recorder != nil {
		c.recorder.AddValues(values)
	}

	err := c.addRow(ctx, e, values)
	if err != nil && e.writeCloser != nil {
		_ = e.writeCloser.Close()
	}

	return err
}

func (c *Service) AssignRows(entry *Entry, rows *sql.Rows) error {
	if len(entry.Meta.Fields) > 0 {
		return nil
	}

	types, err := rows.ColumnTypes()
	if err != nil {
		return err
	}

	ioColumns := io.TypesToColumns(types)
	entry.Meta.Fields = make([]*Field, len(ioColumns))

	for i, column := range ioColumns {
		length, _ := column.Length()
		precision, scale, _ := column.DecimalSize()
		nullable, _ := column.Nullable()
		entry.Meta.Fields[i] = &Field{
			ColumnName:         column.Name(),
			ColumnLength:       length,
			ColumnPrecision:    precision,
			ColumnScale:        scale,
			ColumnScanType:     column.ScanType().String(),
			_columnScanType:    column.ScanType(),
			ColumnNullable:     nullable,
			ColumnDatabaseName: column.DatabaseTypeName(),
			ColumnTag:          column.Tag(),
		}
	}

	return nil
}

func (c *Service) matchesType(actualTypes []string) bool {
	if len(actualTypes) != len(c.dataTypes) {
		return false
	}

	for i, dataType := range c.dataTypes {
		if dataType != actualTypes[i] {
			return false
		}
	}
	return true
}
