package xml

import (
	"testing"

	"github.com/dizzyfool/genna/model"
)

func TestNewAttribute_Enum(t *testing.T) {
	entity := model.Entity{
		GoName: "Task",
		Columns: []model.Column{
			model.NewColumn("status", "varchar", "", false, false, false, false, 0, false, false, 0, []string{"pending", "running"}, true, "scan_status", 10, nil),
		},
	}

	attr := newAttribute(entity, entity.Columns[0])

	if !attr.IsEnum {
		t.Error("expected IsEnum=true")
	}
	if attr.EnumType != "scan_status" {
		t.Errorf("EnumType = %q", attr.EnumType)
	}
	if attr.EnumValues != "pending,running" {
		t.Errorf("EnumValues = %q", attr.EnumValues)
	}
	if attr.GoType != "ScanStatus" {
		t.Errorf("GoType = %q, want ScanStatus", attr.GoType)
	}
	if attr.DBType != "varchar" {
		t.Errorf("DBType = %q, want varchar", attr.DBType)
	}
}

func TestNewAttribute_EnumArray(t *testing.T) {
	entity := model.Entity{
		GoName: "Task",
		Columns: []model.Column{
			model.NewColumn("tags", "varchar", "", false, false, false, true, 1, false, false, 0, []string{"a", "b"}, true, "tag_type", 10, nil),
		},
	}

	attr := newAttribute(entity, entity.Columns[0])

	if !attr.IsEnum {
		t.Error("expected IsEnum=true")
	}
	if attr.GoType != "[]TagType" {
		t.Errorf("GoType = %q, want []TagType", attr.GoType)
	}
	if !attr.IsArray {
		t.Error("expected IsArray=true")
	}
}

func TestNewAttribute_NonEnum(t *testing.T) {
	entity := model.Entity{
		GoName: "Task",
		Columns: []model.Column{
			model.NewColumn("title", "varchar", "", false, false, false, false, 0, false, false, 255, nil, false, "", 10, nil),
		},
	}

	attr := newAttribute(entity, entity.Columns[0])

	if attr.IsEnum {
		t.Error("expected IsEnum=false")
	}
	if attr.EnumType != "" {
		t.Errorf("EnumType = %q", attr.EnumType)
	}
}
