package core

import (
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
