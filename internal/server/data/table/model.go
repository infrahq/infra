//go:build template

package table

type TemplateTable struct{}

func (t TemplateTable) Columns() []string {
	return []string{}
}

func (t TemplateTable) Values() []any {
	return []any{}
}

func (t *TemplateTable) ScanFields() []any {
	return []any{}
}
