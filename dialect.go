package core

import (
	"fmt"
	"strings"
	"time"
)

type DbType string

type Uri struct {
	DbType  DbType
	Proto   string
	Host    string
	Port    string
	DbName  string
	User    string
	Passwd  string
	Charset string
	Laddr   string
	Raddr   string
	Timeout time.Duration
	Schema  string
}

// a dialect is a driver's wrapper
type Dialect interface {
	SetLogger(logger ILogger)
	Init(*DB, *Uri, string, string) error
	URI() *Uri
	DB() *DB
	DBType() DbType
	SqlType(*Column) string
	FormatBytes(b []byte) string

	DriverName() string
	DataSourceName() string

	QuoteStr() string
	IsReserved(string) bool
	Quote(string) string
	AndStr() string
	OrStr() string
	EqStr() string
	RollBackStr() string
	AutoIncrStr() string

	SupportInsertMany() bool
	SupportEngine() bool
	SupportCharset() bool
	SupportDropIfExists() bool
	SupportForeignKeys() bool
	IndexOnTable() bool
	ShowCreateNull() bool

	IndexCheckSql(tableName, idxName string) (string, []interface{})
	TableCheckSql(tableName string) (string, []interface{})

	IsColumnExist(tableName string, colName string) (bool, error)

	CreateTableSql(table *Table, tableName, storeEngine, charset string) string
	DropTableSql(tableName string) string
	CreateIndexSql(tableName string, index *Index) string
	DropIndexSql(tableName string, index *Index) string
	CreateForeignKeySql(tableName string, foreignKey *ForeignKey) string
	DropForeignKeySql(tableName string, foreignKey *ForeignKey) string

	ModifyColumnSql(tableName string, col *Column) string

	ForUpdateSql(query string) string

	//CreateTableIfNotExists(table *Table, tableName, storeEngine, charset string) error
	//MustDropTable(tableName string) error

	GetColumns(tableName string) ([]string, map[string]*Column, []*ForeignKey, error)
	GetTables() ([]*Table, error)
	GetIndexes(tableName string) (map[string]*Index, error)

	Filters() []Filter
}

func OpenDialect(dialect Dialect) (*DB, error) {
	return Open(dialect.DriverName(), dialect.DataSourceName())
}

type Base struct {
	db             *DB
	dialect        Dialect
	driverName     string
	dataSourceName string
	logger         ILogger
	*Uri
}

func (b *Base) DB() *DB {
	return b.db
}

func (b *Base) SetLogger(logger ILogger) {
	b.logger = logger
}

func (b *Base) Init(db *DB, dialect Dialect, uri *Uri, drivername, dataSourceName string) error {
	b.db, b.dialect, b.Uri = db, dialect, uri
	b.driverName, b.dataSourceName = drivername, dataSourceName
	return nil
}

func (b *Base) URI() *Uri {
	return b.Uri
}

func (b *Base) DBType() DbType {
	return b.Uri.DbType
}

func (b *Base) FormatBytes(bs []byte) string {
	return fmt.Sprintf("0x%x", bs)
}

func (b *Base) DriverName() string {
	return b.driverName
}

func (b *Base) ShowCreateNull() bool {
	return true
}

func (b *Base) DataSourceName() string {
	return b.dataSourceName
}

func (b *Base) AndStr() string {
	return "AND"
}

func (b *Base) OrStr() string {
	return "OR"
}

func (b *Base) EqStr() string {
	return "="
}

func (db *Base) RollBackStr() string {
	return "ROLL BACK"
}

func (db *Base) SupportDropIfExists() bool {
	return true
}

func (db *Base) DropTableSql(tableName string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS `%s`", tableName)
}

func (db *Base) HasRecords(query string, args ...interface{}) (bool, error) {
	db.LogSQL(query, args)
	rows, err := db.DB().Query(query, args...)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	if rows.Next() {
		return true, nil
	}
	return false, nil
}

func (db *Base) IsColumnExist(tableName, colName string) (bool, error) {
	query := "SELECT `COLUMN_NAME` FROM `INFORMATION_SCHEMA`.`COLUMNS` WHERE `TABLE_SCHEMA` = ? AND `TABLE_NAME` = ? AND `COLUMN_NAME` = ?"
	query = strings.Replace(query, "`", db.dialect.QuoteStr(), -1)
	return db.HasRecords(query, db.DbName, tableName, colName)
}

/*
func (db *Base) CreateTableIfNotExists(table *Table, tableName, storeEngine, charset string) error {
	sql, args := db.dialect.TableCheckSql(tableName)
	rows, err := db.DB().Query(sql, args...)
	if db.Logger != nil {
		db.Logger.Info("[sql]", sql, args)
	}
	if err != nil {
		return err
	}
	defer rows.Close()

	if rows.Next() {
		return nil
	}

	sql = db.dialect.CreateTableSql(table, tableName, storeEngine, charset)
	_, err = db.DB().Exec(sql)
	if db.Logger != nil {
		db.Logger.Info("[sql]", sql)
	}
	return err
}*/

func (db *Base) CreateIndexSql(tableName string, index *Index) string {
	quote := db.dialect.Quote
	var unique string
	var idxName string
	if index.Type == UniqueType {
		unique = " UNIQUE"
	}
	idxName = index.XName(tableName)
	return fmt.Sprintf("CREATE%s INDEX %v ON %v (%v)", unique,
		quote(idxName), quote(tableName),
		quote(strings.Join(index.Cols, quote(","))))
}

func (db *Base) DropIndexSql(tableName string, index *Index) string {
	quote := db.dialect.Quote
	var name string
	if index.IsRegular {
		name = index.XName(tableName)
	} else {
		name = index.Name
	}
	return fmt.Sprintf("DROP INDEX %v ON %s", quote(name), quote(tableName))
}

func (db *Base) SupportForeignKeys() bool {
	return true
}

func (db *Base) CreateForeignKeySql(tableName string, foreignKey *ForeignKey) string {
	quote := db.dialect.Quote
	_, constraintNameUnquoted := foreignKey.Name(tableName)
	constraintName := quote(constraintNameUnquoted)

	var fkColLocQuoted string
	for _, v := range foreignKey.ColumnName {
		fkColLocQuoted += quote(v) + ", "
	}
	fkColLocQuoted = strings.TrimSuffix(fkColLocQuoted, ", ")

	var fkColName string
	for _, v := range foreignKey.TargetColumn {
		fkColName += quote(v) + ", "
	}
	fkColName = strings.TrimSuffix(fkColName, ", ")

	fkColTable := quote(foreignKey.TargetTable)

	var fkUpdate string
	if foreignKey.UpdateAction != "" {
		fkUpdate = foreignKey.UpdateAction
	} else {
		fkUpdate = "CASCADE"
	}

	var fkDelete string
	if foreignKey.DeleteAction != "" {
		fkDelete = foreignKey.DeleteAction
	} else {
		fkDelete = "RESTRICT"
	}

	return fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s) ON UPDATE %s ON DELETE %s",
		quote(tableName), constraintName, fkColLocQuoted, fkColTable, fkColName, fkUpdate, fkDelete)
}

func (db *Base) DropForeignKeySql(tableName string, foreignKey *ForeignKey) string {
	quote := db.dialect.Quote
	_, constraintName := foreignKey.Name(tableName)
	return fmt.Sprintf("ALTER TABLE %s DROP FOREIGN KEY %s;", quote(tableName), quote(constraintName))
}

func (db *Base) ModifyColumnSql(tableName string, col *Column) string {
	return fmt.Sprintf("alter table %s MODIFY COLUMN %s", tableName, col.StringNoPk(db.dialect))
}

func (b *Base) CreateTableSql(table *Table, tableName, storeEngine, charset string) string {
	var sql string
	sql = "CREATE TABLE IF NOT EXISTS "
	if tableName == "" {
		tableName = table.Name
	}

	sql += b.dialect.Quote(tableName)
	sql += " ("

	if len(table.ColumnsSeq()) > 0 {
		pkList := table.PrimaryKeys

		for _, colName := range table.ColumnsSeq() {
			col := table.GetColumn(colName)
			if col.IsPrimaryKey && len(pkList) == 1 {
				sql += col.String(b.dialect)
			} else {
				sql += col.StringNoPk(b.dialect)
			}
			sql = strings.TrimSpace(sql)
			sql += ", "
		}

		if len(pkList) > 1 {
			sql += "PRIMARY KEY ( "
			sql += b.dialect.Quote(strings.Join(pkList, b.dialect.Quote(",")))
			sql += " ), "
		}

		sql = sql[:len(sql)-2]
	}
	sql += ")"

	if b.dialect.SupportEngine() && storeEngine != "" {
		sql += " ENGINE=" + storeEngine
	}
	if b.dialect.SupportCharset() {
		if len(charset) == 0 {
			charset = b.dialect.URI().Charset
		}
		if len(charset) > 0 {
			sql += " DEFAULT CHARSET " + charset
		}
	}

	return sql
}

func (b *Base) ForUpdateSql(query string) string {
	return query + " FOR UPDATE"
}

func (b *Base) LogSQL(sql string, args []interface{}) {
	if b.logger != nil && b.logger.IsShowSQL() {
		if len(args) > 0 {
			b.logger.Info("[sql]", sql, args)
		} else {
			b.logger.Info("[sql]", sql)
		}
	}
}

var (
	dialects = map[string]func() Dialect{}
)

// RegisterDialect register database dialect
func RegisterDialect(dbName DbType, dialectFunc func() Dialect) {
	if dialectFunc == nil {
		panic("core: Register dialect is nil")
	}
	dialects[strings.ToLower(string(dbName))] = dialectFunc // !nashtsai! allow override dialect
}

// QueryDialect query if registed database dialect
func QueryDialect(dbName DbType) Dialect {
	if d, ok := dialects[strings.ToLower(string(dbName))]; ok {
		return d()
	}
	return nil
}
