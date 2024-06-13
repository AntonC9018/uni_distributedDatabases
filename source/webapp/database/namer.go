package database

import (
	"strings"

	"gorm.io/gorm/schema"
)

type myNamer struct {
}

var MyNamer myNamer

func (n *myNamer) TableName(table string) string {
    return strings.ToLower(table)
}
func (n *myNamer) SchemaName(table string) string {
    return strings.ToLower(table)
}
func (n *myNamer) ColumnName(table, column string) string {
    return strings.ToLower(column)
}
func (n *myNamer) JoinTableName(joinTable string) string {
    return strings.ToLower(joinTable)
}
func (n *myNamer) RelationshipFKName(rel schema.Relationship) string {
	return n.formatName("fk", rel.Schema.Table, rel.Name)
}
func (n *myNamer) CheckerName(table, column string) string {
	return n.formatName("chk", table, strings.ToLower(column))
}
func (n *myNamer) IndexName(table, column string) string {
	return n.formatName("idx", table, strings.ToLower(column))
}
func (n *myNamer) UniqueName(table, column string) string {
	return n.formatName("uni", table, strings.ToLower(column))
}
func (n *myNamer) formatName(prefix, table, name string) string {
	formattedName := strings.Join([]string{ prefix, table, name }, "_")
	return formattedName
}
