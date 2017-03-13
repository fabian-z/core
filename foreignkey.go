package core

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"hash/fnv"
	"sort"
)

// ForeignKey represents a foreign key constraint
type ForeignKey struct {
	ColumnName   []string
	TargetTable  string
	TargetColumn []string
	UpdateAction string
	DeleteAction string
}

func (fk *ForeignKey) Equal(dst *ForeignKey) bool {
	if len(fk.ColumnName) != len(dst.ColumnName) {
		return false
	}
	if len(fk.TargetColumn) != len(dst.TargetColumn) {
		return false
	}
	if fk.TargetTable != dst.TargetTable {
		return false
	}
	if fk.UpdateAction != dst.UpdateAction ||
		fk.DeleteAction != dst.DeleteAction {
		return false
	}

	sort.StringSlice(fk.ColumnName).Sort()
	sort.StringSlice(dst.ColumnName).Sort()
	for i := 0; i < len(fk.ColumnName); i++ {
		if fk.ColumnName[i] != dst.ColumnName[i] {
			return false
		}
	}

	sort.StringSlice(fk.TargetColumn).Sort()
	sort.StringSlice(dst.TargetColumn).Sort()
	for i := 0; i < len(fk.TargetColumn); i++ {
		if fk.TargetColumn[i] != dst.TargetColumn[i] {
			return false
		}
	}

	return true
}

func (fk *ForeignKey) Name(tableName string) (index string, constraint string) {

	base := tableName + "_" + fk.ColumnName[0] + "_"

	hashStruct := struct {
		tn string
		fk ForeignKey
	}{
		tableName,
		*fk,
	}

	var encodeBuf bytes.Buffer
	enc := gob.NewEncoder(&encodeBuf)
	err := enc.Encode(hashStruct)
	if err != nil {
		panic(err)
	}

	h := fnv.New32a()
	h.Write([]byte(encodeBuf.Bytes()))
	hash := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	index = "FK_IDX_" + base + hash
	constraint = "FK_" + base + hash

	return

}
