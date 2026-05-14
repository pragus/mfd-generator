package model

import (
	"bytes"
	"strings"
	"testing"

	"github.com/vmkteam/mfd-generator/mfd"
)

func TestPackNamespace_EnumDedup(t *testing.T) {
	attr1 := &mfd.Attribute{
		Name:       "Status",
		DBName:     "status",
		DBType:     "varchar",
		GoType:     "ScanStatus",
		IsEnum:     true,
		EnumType:   "scan_status",
		EnumValues: "pending,running,finished",
	}
	attr2 := &mfd.Attribute{
		Name:       "PrevStatus",
		DBName:     "prev_status",
		DBType:     "varchar",
		GoType:     "ScanStatus",
		IsEnum:     true,
		EnumType:   "scan_status",
		EnumValues: "pending,running,finished",
	}

	ns := &mfd.Namespace{
		Name: "test",
		Entities: []*mfd.Entity{
			{
				Name:       "Task",
				Namespace:  "test",
				Table:      "tasks",
				Attributes: mfd.Attributes{attr1, attr2},
			},
		},
	}

	data := PackNamespace([]*mfd.Namespace{ns}, Options{Package: "db", GoPGVer: mfd.GoPG10})

	if !data.HasEnums {
		t.Fatal("expected HasEnums=true")
	}
	if len(data.EnumTypes) != 1 {
		t.Fatalf("expected 1 enum type, got %d", len(data.EnumTypes))
	}
	e := data.EnumTypes[0]
	if e.Name != "ScanStatus" {
		t.Errorf("enum name = %q", e.Name)
	}
	if len(e.Values) != 3 {
		t.Fatalf("expected 3 values, got %d", len(e.Values))
	}
	if e.Values[0].ConstName != "ScanStatusPending" || e.Values[0].Value != "pending" {
		t.Errorf("values[0] = %+v", e.Values[0])
	}
}

func TestModelTemplate_EnumOutput(t *testing.T) {
	attr := &mfd.Attribute{
		Name:       "Status",
		DBName:     "status",
		DBType:     "varchar",
		GoType:     "ScanStatus",
		IsEnum:     true,
		EnumType:   "scan_status",
		EnumValues: "pending,running",
	}

	ns := &mfd.Namespace{
		Name: "test",
		Entities: []*mfd.Entity{
			{
				Name:       "Scan",
				Namespace:  "test",
				Table:      "scans",
				Attributes: mfd.Attributes{attr},
			},
		},
	}

	data := PackNamespace([]*mfd.Namespace{ns}, Options{Package: "db", GoPGVer: mfd.GoPG10})

	var buf bytes.Buffer
	if err := mfd.Render(&buf, modelDefaultTemplate, data); err != nil {
		t.Fatal(err)
	}

	output := buf.String()

	if !strings.Contains(output, "type ScanStatus string") {
		t.Error("missing enum type definition")
	}
	if !strings.Contains(output, `ScanStatusPending ScanStatus = "pending"`) {
		t.Error("missing enum const pending")
	}
	if !strings.Contains(output, `ScanStatusRunning ScanStatus = "running"`) {
		t.Error("missing enum const running")
	}
	if !strings.Contains(output, "Status ScanStatus") {
		t.Error("missing ScanStatus field in struct")
	}
}
